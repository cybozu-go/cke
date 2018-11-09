package op

import (
	"context"
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/util/yaml"

	"github.com/cybozu-go/cke"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// retrieved from https://github.com/kelseyhightower/kubernetes-the-hard-way
var deploymentTemplate = `
metadata:
  name: cluster-dns
  namespace: kube-system
  labels:
    k8s-app: cluster-dns
    kubernetes.io/name: "CoreDNS"
spec:
  replicas: 2
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxUnavailable: 1
  selector:
    matchLabels:
      k8s-app: cluster-dns
  template:
    metadata:
      labels:
        k8s-app: cluster-dns
    spec:
      serviceAccountName: cluster-dns
      tolerations:
        - key: node-role.kubernetes.io/master
          effect: NoSchedule
        - key: "CriticalAddonsOnly"
          operator: "Exists"
      containers:
      - name: coredns
        image: %s
        imagePullPolicy: IfNotPresent
        resources:
          limits:
            memory: 170Mi
          requests:
            cpu: 100m
            memory: 70Mi
        args: [ "-conf", "/etc/coredns/Corefile" ]
        volumeMounts:
        - name: config-volume
          mountPath: /etc/coredns
          readOnly: true
        ports:
        - containerPort: 53
          name: dns
          protocol: UDP
        - containerPort: 53
          name: dns-tcp
          protocol: TCP
        securityContext:
          allowPrivilegeEscalation: false
          capabilities:
            add:
            - NET_BIND_SERVICE
            drop:
            - all
          readOnlyRootFilesystem: true
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
            scheme: HTTP
          initialDelaySeconds: 60
          timeoutSeconds: 5
          successThreshold: 1
          failureThreshold: 5
      dnsPolicy: Default
      volumes:
        - name: config-volume
          configMap:
            name: ` + clusterDNSConfigMapName + `
            items:
            - key: Corefile
              path: Corefile
`

type kubeClusterDNSCreateOp struct {
	apiserver  *cke.Node
	domain     string
	dnsServers []string
	finished   bool
}

// KubeClusterDNSCreateOp returns an Operator to create cluster resolver.
func KubeClusterDNSCreateOp(apiserver *cke.Node, domain string, dnsServers []string) cke.Operator {
	return &kubeClusterDNSCreateOp{
		apiserver:  apiserver,
		domain:     domain,
		dnsServers: dnsServers,
	}
}

func (o *kubeClusterDNSCreateOp) Name() string {
	return "create-cluster-dns"
}

func (o *kubeClusterDNSCreateOp) NextCommand() cke.Commander {
	if o.finished {
		return nil
	}
	o.finished = true
	return createClusterDNSCommand{o.apiserver, o.domain, o.dnsServers}
}

type createClusterDNSCommand struct {
	apiserver  *cke.Node
	domain     string
	dnsServers []string
}

func (c createClusterDNSCommand) Run(ctx context.Context, inf cke.Infrastructure) error {
	cs, err := inf.K8sClient(ctx, c.apiserver)
	if err != nil {
		return err
	}

	// ServiceAccount
	accounts := cs.CoreV1().ServiceAccounts("kube-system")
	_, err = accounts.Get(clusterDNSAppName, metav1.GetOptions{})
	switch {
	case err == nil:
	case errors.IsNotFound(err):
		_, err = accounts.Create(&corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name:      clusterDNSAppName,
				Namespace: "kube-system",
			},
		})
		if err != nil {
			return err
		}
	default:
		return err
	}

	// ClusterRole
	clusterRoles := cs.RbacV1().ClusterRoles()
	_, err = clusterRoles.Get(clusterDNSRBACRoleName, metav1.GetOptions{})
	switch {
	case err == nil:
	case errors.IsNotFound(err):
		_, err = clusterRoles.Create(&rbacv1.ClusterRole{
			ObjectMeta: metav1.ObjectMeta{
				Name: clusterDNSRBACRoleName,
				Labels: map[string]string{
					"kubernetes.io/bootstrapping": "rbac-defaults",
				},
			},
			Rules: []rbacv1.PolicyRule{
				{
					APIGroups: []string{""},
					Resources: []string{
						"endpoints",
						"services",
						"pods",
						"namespaces",
					},
					Verbs: []string{
						"list",
						"watch",
					},
				},
			},
		})
		if err != nil {
			return err
		}
	default:
		return err
	}

	// ClusterRoleBinding
	clusterRoleBindings := cs.RbacV1().ClusterRoleBindings()
	_, err = clusterRoleBindings.Get(clusterDNSRBACRoleName, metav1.GetOptions{})
	switch {
	case err == nil:
	case errors.IsNotFound(err):
		_, err = clusterRoleBindings.Create(&rbacv1.ClusterRoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name: clusterDNSRBACRoleName,
				Labels: map[string]string{
					"kubernetes.io/bootstrapping": "rbac-defaults",
				},
				Annotations: map[string]string{
					// turn on auto-reconciliation
					// https://kubernetes.io/docs/reference/access-authn-authz/rbac/#auto-reconciliation
					rbacv1.AutoUpdateAnnotationKey: "true",
				},
			},
			RoleRef: rbacv1.RoleRef{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "ClusterRole",
				Name:     clusterDNSRBACRoleName,
			},
			Subjects: []rbacv1.Subject{
				{
					Kind:      rbacv1.ServiceAccountKind,
					Name:      clusterDNSAppName,
					Namespace: "kube-system",
				},
			},
		})
		if err != nil {
			return err
		}
	default:
		return err
	}

	// ConfigMap
	configs := cs.CoreV1().ConfigMaps("kube-system")
	_, err = configs.Get(clusterDNSConfigMapName, metav1.GetOptions{})
	switch {
	case err == nil:
	case errors.IsNotFound(err):
		_, err = configs.Create(getClusterDNSConfigMap(c.domain, c.dnsServers))
		if err != nil {
			return err
		}
	default:
		return err
	}

	// Deployment
	deployments := cs.AppsV1().Deployments("kube-system")
	_, err = deployments.Get(clusterDNSAppName, metav1.GetOptions{})
	switch {
	case err == nil:
	case errors.IsNotFound(err):
		deploymentText := fmt.Sprintf(deploymentTemplate, cke.CoreDNSImage)
		deployment := new(appsv1.Deployment)
		err = yaml.NewYAMLToJSONDecoder(strings.NewReader(deploymentText)).Decode(deployment)
		if err != nil {
			return err
		}
		_, err = deployments.Create(deployment)
		if err != nil {
			return err
		}
	default:
		return err
	}

	// Service
	services := cs.CoreV1().Services("kube-system")
	_, err = services.Get(clusterDNSAppName, metav1.GetOptions{})
	switch {
	case err == nil:
	case errors.IsNotFound(err):
		_, err = services.Create(getClusterDNSService())
		if err != nil {
			return err
		}
	default:
		return err
	}

	return nil
}

func (c createClusterDNSCommand) Command() cke.Command {
	return cke.Command{
		Name:   "createClusterDNSCommand",
		Target: "kube-system",
	}
}

type kubeClusterDNSUpdateOp struct {
	apiserver  *cke.Node
	domain     string
	dnsServers []string
	finished   bool
}

// KubeClusterDNSUpdateOp returns an Operator to update cluster resolver.
func KubeClusterDNSUpdateOp(apiserver *cke.Node, domain string, dnsServers []string) cke.Operator {
	return &kubeClusterDNSUpdateOp{
		apiserver:  apiserver,
		domain:     domain,
		dnsServers: dnsServers,
	}
}

func (o *kubeClusterDNSUpdateOp) Name() string {
	return "update-cluster-dns"
}

func (o *kubeClusterDNSUpdateOp) NextCommand() cke.Commander {
	if o.finished {
		return nil
	}
	o.finished = true
	return updateClusterDNSCommand{o.apiserver, o.domain, o.dnsServers}
}

type updateClusterDNSCommand struct {
	apiserver  *cke.Node
	domain     string
	dnsServers []string
}

func (c updateClusterDNSCommand) Run(ctx context.Context, inf cke.Infrastructure) error {
	cs, err := inf.K8sClient(ctx, c.apiserver)
	if err != nil {
		return err
	}

	configs := cs.CoreV1().ConfigMaps("kube-system")
	err = configs.Delete(clusterDNSAppName, &metav1.DeleteOptions{})
	if err != nil {
		return err
	}
	_, err = configs.Create(getClusterDNSConfigMap(c.domain, c.dnsServers))
	if err != nil {
		return err
	}

	services := cs.CoreV1().Services("kube-system")
	err = services.Delete(clusterDNSAppName, &metav1.DeleteOptions{})
	if err != nil {
		return err
	}
	_, err = services.Create(getClusterDNSService())
	return err
}

func (c updateClusterDNSCommand) Command() cke.Command {
	return cke.Command{
		Name:   "updateClusterDNSCommand",
		Target: "kube-system",
	}
}

func getClusterDNSConfigMap(domain string, dnsServers []string) *corev1.ConfigMap {
	var coreFileContents string
	if len(dnsServers) == 0 {
		coreFileContents = fmt.Sprintf(`.:53 {
    errors
    health
    log
    kubernetes %s in-addr.arpa ip6.arpa {
      pods verified
    }
    cache 30
    reload
    loadbalance
}
`, domain)
	} else {
		coreFileContents = fmt.Sprintf(`.:53 {
    errors
    health
    log
    kubernetes %s in-addr.arpa ip6.arpa {
      pods verified
      upstream
      fallthrough in-addr.arpa ip6.arpa
    }
    proxy . %s
    cache 30
    loop
    reload
    loadbalance
}
`, domain, strings.Join(dnsServers, " "))
	}

	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      clusterDNSConfigMapName,
			Namespace: "kube-system",
			Labels: map[string]string{
				"cke-domain":      domain,
				"cke-dns-servers": strings.Join(dnsServers, "_"),
			},
		},
		Data: map[string]string{
			"Corefile": coreFileContents,
		},
	}
}

func getClusterDNSService() *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      clusterDNSAppName,
			Namespace: "kube-system",
			Labels: map[string]string{
				"k8s-app":                       clusterDNSAppName,
				"kubernetes.io/cluster-service": "true",
				"kubernetes.io/name":            "CoreDNS",
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"k8s-app": clusterDNSAppName,
			},
			Ports: []corev1.ServicePort{
				{
					Name:     "dns",
					Port:     53,
					Protocol: corev1.ProtocolUDP,
				},
				{
					Name:     "dns-tcp",
					Port:     53,
					Protocol: corev1.ProtocolTCP,
				},
			},
		},
	}
}
