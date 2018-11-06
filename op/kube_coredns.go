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
var deploymentText = `
metadata:
  name: coredns
  namespace: kube-system
  labels:
    k8s-app: coredns
    kubernetes.io/name: "CoreDNS"
spec:
  replicas: 2
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxUnavailable: 1
  selector:
    matchLabels:
      k8s-app: coredns
  template:
    metadata:
      labels:
        k8s-app: coredns
    spec:
      serviceAccountName: coredns
      tolerations:
        - key: node-role.kubernetes.io/master
          effect: NoSchedule
        - key: "CriticalAddonsOnly"
          operator: "Exists"
      containers:
      - name: coredns
        image: coredns/coredns:1.2.2
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
            name: coredns
            items:
            - key: Corefile
              path: Corefile
`

type kubeCoreDNSCreateOp struct {
	apiserver *cke.Node
	params    cke.KubeletParams
	finished  bool
}

// KubeCoreDNSCreateOp returns an Operator to create CoreDNS.
func KubeCoreDNSCreateOp(apiserver *cke.Node, params cke.KubeletParams) cke.Operator {
	return &kubeCoreDNSCreateOp{
		apiserver: apiserver,
		params:    params,
	}
}

func (o *kubeCoreDNSCreateOp) Name() string {
	return "create-coredns"
}

func (o *kubeCoreDNSCreateOp) NextCommand() cke.Commander {
	if o.finished {
		return nil
	}
	o.finished = true
	return createCoreDNSCommand{o.apiserver, o.params}
}

type createCoreDNSCommand struct {
	apiserver *cke.Node
	params    cke.KubeletParams
}

func (c createCoreDNSCommand) Run(ctx context.Context, inf cke.Infrastructure) error {
	cs, err := inf.K8sClient(ctx, c.apiserver)
	if err != nil {
		return err
	}

	// ServiceAccount
	accounts := cs.CoreV1().ServiceAccounts("kube-system")
	_, err = accounts.Get(coreDNSAppName, metav1.GetOptions{})
	switch {
	case err == nil:
	case errors.IsNotFound(err):
		_, err = accounts.Create(&corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name:      coreDNSAppName,
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
	_, err = cs.RbacV1().ClusterRoles().Create(&rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: coreDNSRBACRoleName,
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

	// ClusterRoleBinding
	_, err = cs.RbacV1().ClusterRoleBindings().Create(&rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: coreDNSRBACRoleName,
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
			Name:     coreDNSRBACRoleName,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      rbacv1.ServiceAccountKind,
				Name:      coreDNSAppName,
				Namespace: "kube-system",
			},
		},
	})
	if err != nil {
		return err
	}

	// ConfigMap
	configs := cs.CoreV1().ConfigMaps("kube-system")
	_, err = configs.Get(coreDNSAppName, metav1.GetOptions{})
	switch {
	case err == nil:
	case errors.IsNotFound(err):
		_, err = configs.Create(getCoreDNSConfigMap(c.params.Domain))
		if err != nil {
			return err
		}
	default:
		return err
	}

	// Deployment
	deployments := cs.AppsV1().Deployments("kube-system")
	_, err = deployments.Get(coreDNSAppName, metav1.GetOptions{})
	switch {
	case err == nil:
	case errors.IsNotFound(err):
		deployment := new(appsv1.Deployment)
		err = deployment.Unmarshal([]byte(deploymentText))
		err = yaml.NewYAMLToJSONDecoder(strings.NewReader(deploymentText)).Decode(&deployment)
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
	_, err = services.Get(coreDNSAppName, metav1.GetOptions{})
	switch {
	case err == nil:
	case errors.IsNotFound(err):
		_, err = services.Create(getCoreDNSService(c.params.DNS))
		if err != nil {
			return err
		}
	default:
		return err
	}

	return err
}

func (c createCoreDNSCommand) Command() cke.Command {
	return cke.Command{
		Name:   "createCoreDNSCommand",
		Target: "kube-system",
	}
}

type kubeCoreDNSUpdateOp struct {
	apiserver *cke.Node
	params    cke.KubeletParams
	finished  bool
}

// KubeCoreDNSUpdateOp returns an Operator to update CoreDNS.
func KubeCoreDNSUpdateOp(apiserver *cke.Node, params cke.KubeletParams) cke.Operator {
	return &kubeCoreDNSUpdateOp{
		apiserver: apiserver,
		params:    params,
	}
}

func (o *kubeCoreDNSUpdateOp) Name() string {
	return "update-coredns"
}

func (o *kubeCoreDNSUpdateOp) NextCommand() cke.Commander {
	if o.finished {
		return nil
	}
	o.finished = true
	return updateCoreDNSCommand{o.apiserver, o.params}
}

type updateCoreDNSCommand struct {
	apiserver *cke.Node
	params    cke.KubeletParams
}

func (c updateCoreDNSCommand) Run(ctx context.Context, inf cke.Infrastructure) error {
	cs, err := inf.K8sClient(ctx, c.apiserver)
	if err != nil {
		return err
	}

	configs := cs.CoreV1().ConfigMaps("kube-system")
	err = configs.Delete(coreDNSAppName, &metav1.DeleteOptions{})
	if err != nil {
		return err
	}
	_, err = configs.Create(getCoreDNSConfigMap(c.params.Domain))

	services := cs.CoreV1().Services("kube-system")
	err = services.Delete(coreDNSAppName, &metav1.DeleteOptions{})
	if err != nil {
		return err
	}
	_, err = services.Create(getCoreDNSService(c.params.DNS))
	return err
}

func (c updateCoreDNSCommand) Command() cke.Command {
	return cke.Command{
		Name:   "updateCoreDNSCommand",
		Target: "kube-system",
	}
}

func getCoreDNSConfigMap(domain string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      coreDNSAppName,
			Namespace: "kube-system",
		},
		Data: map[string]string{
			"Corefile": fmt.Sprintf(`.:53 {
    errors
    health
    log
    kubernetes %s in-addr.arpa ip6.arpa {
      pods verified
      upstream
      fallthrough in-addr.arpa ip6.arpa
    }
    proxy . /etc/resolv.conf
    cache 30
    loop
    reload
    loadbalance
}
`, domain),
		},
	}
}

func getCoreDNSService(dns string) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      coreDNSAppName,
			Namespace: "kube-system",
			Labels: map[string]string{
				"k8s-app":                       coreDNSAppName,
				"kubernetes.io/cluster-service": "true",
				"kubernetes.io/name":            "CoreDNS",
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"k8s-app": coreDNSAppName,
			},
			ClusterIP: dns,
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
