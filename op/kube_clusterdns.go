package op

import (
	"bytes"
	"context"
	"html/template"
	"strings"

	"github.com/cybozu-go/cke"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sYaml "k8s.io/apimachinery/pkg/util/yaml"
)

// CoreDNSTemplateVersion is the version of CoreDNS template
const CoreDNSTemplateVersion = "1"

// retrieved from https://github.com/kelseyhightower/kubernetes-the-hard-way
var deploymentText = `
metadata:
  name: cluster-dns
  namespace: kube-system
  annotations:
    cke.cybozu.com/image: ` + cke.CoreDNSImage.Name() + `
    cke.cybozu.com/template-version: ` + CoreDNSTemplateVersion + `
spec:
  replicas: 2
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxUnavailable: 1
  selector:
    matchLabels:
      cke.cybozu.com/appname: cluster-dns
  template:
    metadata:
      labels:
        cke.cybozu.com/appname: cluster-dns
    spec:
      serviceAccountName: cluster-dns
      tolerations:
        - key: node-role.kubernetes.io/master
          effect: NoSchedule
        - key: "CriticalAddonsOnly"
          operator: "Exists"
      containers:
      - name: coredns
        image: ` + cke.CoreDNSImage.Name() + `
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
            name: ` + clusterDNSAppName + `
            items:
            - key: Corefile
              path: Corefile
`

type kubeClusterDNSCreateServiceAccountOp struct {
	apiserver *cke.Node
	finished  bool
}

// KubeClusterDNSCreateServiceAccountOp returns an Operator to create serviceaccount for CoreDNS.
func KubeClusterDNSCreateServiceAccountOp(apiserver *cke.Node) cke.Operator {
	return &kubeClusterDNSCreateServiceAccountOp{
		apiserver: apiserver,
	}
}

func (o *kubeClusterDNSCreateServiceAccountOp) Name() string {
	return "create-cluster-dns-serviceaccount"
}

func (o *kubeClusterDNSCreateServiceAccountOp) NextCommand() cke.Commander {
	if o.finished {
		return nil
	}
	o.finished = true
	return createClusterDNSServiceAccountCommand{o.apiserver}
}

type createClusterDNSServiceAccountCommand struct {
	apiserver *cke.Node
}

func (c createClusterDNSServiceAccountCommand) Run(ctx context.Context, inf cke.Infrastructure) error {
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

	return nil
}

func (c createClusterDNSServiceAccountCommand) Command() cke.Command {
	return cke.Command{
		Name:   "createClusterDNSServiceAccountCommand",
		Target: "kube-system",
	}
}

type kubeClusterDNSCreateRBACRoleOp struct {
	apiserver *cke.Node
	finished  bool
}

// KubeClusterDNSCreateRBACRoleOp returns an Operator to create RBAC Role for CoreDNS.
func KubeClusterDNSCreateRBACRoleOp(apiserver *cke.Node) cke.Operator {
	return &kubeClusterDNSCreateRBACRoleOp{
		apiserver: apiserver,
	}
}

func (o *kubeClusterDNSCreateRBACRoleOp) Name() string {
	return "create-cluster-dns-rbac-role"
}

func (o *kubeClusterDNSCreateRBACRoleOp) NextCommand() cke.Commander {
	if o.finished {
		return nil
	}
	o.finished = true
	return createClusterDNSRBACRoleCommand{o.apiserver}
}

type createClusterDNSRBACRoleCommand struct {
	apiserver *cke.Node
}

func (c createClusterDNSRBACRoleCommand) Run(ctx context.Context, inf cke.Infrastructure) error {
	cs, err := inf.K8sClient(ctx, c.apiserver)
	if err != nil {
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
	return nil
}

func (c createClusterDNSRBACRoleCommand) Command() cke.Command {
	return cke.Command{
		Name:   "createClusterDNSRBACRoleCommand",
		Target: "kube-system",
	}
}

type kubeClusterDNSCreateRBACRoleBindingOp struct {
	apiserver  *cke.Node
	domain     string
	dnsServers []string
	finished   bool
}

// KubeClusterDNSCreateRBACRoleBindingOp returns an Operator to create RBAC Role Binding for CoreDNS.
func KubeClusterDNSCreateRBACRoleBindingOp(apiserver *cke.Node) cke.Operator {
	return &kubeClusterDNSCreateRBACRoleBindingOp{
		apiserver: apiserver,
	}
}

func (o *kubeClusterDNSCreateRBACRoleBindingOp) Name() string {
	return "create-cluster-dns-rbac-role-binding"
}

func (o *kubeClusterDNSCreateRBACRoleBindingOp) NextCommand() cke.Commander {
	if o.finished {
		return nil
	}
	o.finished = true
	return createClusterDNSRBACRoleBindingCommand{o.apiserver}
}

type createClusterDNSRBACRoleBindingCommand struct {
	apiserver *cke.Node
}

func (c createClusterDNSRBACRoleBindingCommand) Run(ctx context.Context, inf cke.Infrastructure) error {
	cs, err := inf.K8sClient(ctx, c.apiserver)
	if err != nil {
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

	return nil
}

func (c createClusterDNSRBACRoleBindingCommand) Command() cke.Command {
	return cke.Command{
		Name:   "createClusterDNSRBACRoleBindingCommand",
		Target: "kube-system",
	}
}

type kubeClusterDNSCreateConfigMapOp struct {
	apiserver  *cke.Node
	domain     string
	dnsServers []string
	finished   bool
}

// KubeClusterDNSCreateConfigMapOp returns an Operator to create ConfigMap for CoreDNS.
func KubeClusterDNSCreateConfigMapOp(apiserver *cke.Node, domain string, dnsServers []string) cke.Operator {
	return &kubeClusterDNSCreateConfigMapOp{
		apiserver:  apiserver,
		domain:     domain,
		dnsServers: dnsServers,
	}
}

func (o *kubeClusterDNSCreateConfigMapOp) Name() string {
	return "create-cluster-dns-configmap"
}

func (o *kubeClusterDNSCreateConfigMapOp) NextCommand() cke.Commander {
	if o.finished {
		return nil
	}
	o.finished = true
	return createClusterDNSConfigMapCommand{o.apiserver, o.domain, o.dnsServers}
}

type createClusterDNSConfigMapCommand struct {
	apiserver  *cke.Node
	domain     string
	dnsServers []string
}

func (c createClusterDNSConfigMapCommand) Run(ctx context.Context, inf cke.Infrastructure) error {
	cs, err := inf.K8sClient(ctx, c.apiserver)
	if err != nil {
		return err
	}

	// ConfigMap
	configs := cs.CoreV1().ConfigMaps("kube-system")
	_, err = configs.Get(clusterDNSAppName, metav1.GetOptions{})
	switch {
	case err == nil:
	case errors.IsNotFound(err):
		_, err = configs.Create(ClusterDNSConfigMap(c.domain, c.dnsServers))
		if err != nil {
			return err
		}
	default:
		return err
	}

	return nil
}

func (c createClusterDNSConfigMapCommand) Command() cke.Command {
	return cke.Command{
		Name:   "createClusterDNSConfigMapCommand",
		Target: "kube-system",
	}
}

type kubeClusterDNSCreateDeploymentOp struct {
	apiserver *cke.Node
	finished  bool
}

// KubeClusterDNSCreateDeploymentOp returns an Operator to create deployment of CoreDNS.
func KubeClusterDNSCreateDeploymentOp(apiserver *cke.Node) cke.Operator {
	return &kubeClusterDNSCreateDeploymentOp{
		apiserver: apiserver,
	}
}

func (o *kubeClusterDNSCreateDeploymentOp) Name() string {
	return "create-cluster-dns-deployment"
}

func (o *kubeClusterDNSCreateDeploymentOp) NextCommand() cke.Commander {
	if o.finished {
		return nil
	}
	o.finished = true
	return createClusterDNSDeploymentCommand{o.apiserver}
}

type createClusterDNSDeploymentCommand struct {
	apiserver *cke.Node
}

func (c createClusterDNSDeploymentCommand) Run(ctx context.Context, inf cke.Infrastructure) error {
	cs, err := inf.K8sClient(ctx, c.apiserver)
	if err != nil {
		return err
	}

	// Deployment
	deployments := cs.AppsV1().Deployments("kube-system")
	_, err = deployments.Get(clusterDNSAppName, metav1.GetOptions{})
	switch {
	case err == nil:
	case errors.IsNotFound(err):
		deployment := new(appsv1.Deployment)
		err = k8sYaml.NewYAMLToJSONDecoder(strings.NewReader(deploymentText)).Decode(deployment)
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
	return nil
}

func (c createClusterDNSDeploymentCommand) Command() cke.Command {
	return cke.Command{
		Name:   "createClusterDNSDeploymentCommand",
		Target: "kube-system",
	}
}

type kubeClusterDNSCreateServiceOp struct {
	apiserver *cke.Node
	finished  bool
}

// KubeClusterDNSCreateOp returns an Operator to create cluster resolver.
func KubeClusterDNSCreateOp(apiserver *cke.Node) cke.Operator {
	return &kubeClusterDNSCreateServiceOp{
		apiserver: apiserver,
	}
}

func (o *kubeClusterDNSCreateServiceOp) Name() string {
	return "create-cluster-dns-service"
}

func (o *kubeClusterDNSCreateServiceOp) NextCommand() cke.Commander {
	if o.finished {
		return nil
	}
	o.finished = true
	return createClusterDNSServiceCommand{o.apiserver}
}

type createClusterDNSServiceCommand struct {
	apiserver *cke.Node
}

func (c createClusterDNSServiceCommand) Run(ctx context.Context, inf cke.Infrastructure) error {
	cs, err := inf.K8sClient(ctx, c.apiserver)
	if err != nil {
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

func (c createClusterDNSServiceCommand) Command() cke.Command {
	return cke.Command{
		Name:   "createClusterDNSServiceCommand",
		Target: "kube-system",
	}
}

type kubeClusterDNSUpdateConfigMapOp struct {
	apiserver *cke.Node
	configmap *corev1.ConfigMap
	finished  bool
}

// KubeClusterDNSUpdateConfigMapOp returns an Operator to update ConfigMap for CoreDNS.
func KubeClusterDNSUpdateConfigMapOp(apiserver *cke.Node, configmap *corev1.ConfigMap) cke.Operator {
	return &kubeClusterDNSUpdateConfigMapOp{
		apiserver: apiserver,
		configmap: configmap,
	}
}

func (o *kubeClusterDNSUpdateConfigMapOp) Name() string {
	return "update-cluster-dns-configmap"
}

func (o *kubeClusterDNSUpdateConfigMapOp) NextCommand() cke.Commander {
	if o.finished {
		return nil
	}
	o.finished = true
	return updateClusterDNSConfigMapCommand{o.apiserver, o.configmap}
}

type updateClusterDNSConfigMapCommand struct {
	apiserver *cke.Node
	configmap *corev1.ConfigMap
}

func (c updateClusterDNSConfigMapCommand) Run(ctx context.Context, inf cke.Infrastructure) error {
	cs, err := inf.K8sClient(ctx, c.apiserver)
	if err != nil {
		return err
	}

	// ConfigMap
	configs := cs.CoreV1().ConfigMaps("kube-system")
	_, err = configs.Update(c.configmap)
	return err
}

func (c updateClusterDNSConfigMapCommand) Command() cke.Command {
	return cke.Command{
		Name:   "updateClusterDNSConfigMapCommand",
		Target: "kube-system",
	}
}

type kubeClusterDNSUpdateDeploymentOp struct {
	apiserver *cke.Node
	finished  bool
}

// KubeClusterDNSUpdateDeploymentOp returns an Operator to update deployment of CoreDNS.
func KubeClusterDNSUpdateDeploymentOp(apiserver *cke.Node) cke.Operator {
	return &kubeClusterDNSUpdateDeploymentOp{
		apiserver: apiserver,
	}
}

func (o *kubeClusterDNSUpdateDeploymentOp) Name() string {
	return "update-cluster-dns-deployment"
}

func (o *kubeClusterDNSUpdateDeploymentOp) NextCommand() cke.Commander {
	if o.finished {
		return nil
	}
	o.finished = true
	return updateClusterDNSDeploymentCommand{o.apiserver}
}

type updateClusterDNSDeploymentCommand struct {
	apiserver *cke.Node
}

func (c updateClusterDNSDeploymentCommand) Run(ctx context.Context, inf cke.Infrastructure) error {
	cs, err := inf.K8sClient(ctx, c.apiserver)
	if err != nil {
		return err
	}

	// Deployment
	deployments := cs.AppsV1().Deployments("kube-system")
	deployment := new(appsv1.Deployment)
	err = k8sYaml.NewYAMLToJSONDecoder(strings.NewReader(deploymentText)).Decode(deployment)
	if err != nil {
		return err
	}
	_, err = deployments.Update(deployment)
	return err
}

func (c updateClusterDNSDeploymentCommand) Command() cke.Command {
	return cke.Command{
		Name:   "updateClusterDNSDeploymentCommand",
		Target: "kube-system",
	}
}

var clusterDNSTemplate = template.Must(template.New("").Parse(`.:53 {
    errors
    health
    log
    kubernetes {{ .Domain }} in-addr.arpa ip6.arpa {
      pods verified
{{- if .Upstreams }}
      upstream
      fallthrough in-addr.arpa ip6.arpa
{{- end }}
    }
{{- if .Upstreams }}
    proxy . {{ .Upstreams }}
{{- end }}
    cache 30
    reload
    loadbalance
}
`))

// ClusterDNSConfigMap returns ConfigMap for CoreDNS
func ClusterDNSConfigMap(domain string, dnsServers []string) *corev1.ConfigMap {
	buf := new(bytes.Buffer)
	err := clusterDNSTemplate.Execute(buf, struct {
		Domain    string
		Upstreams string
	}{
		Domain:    domain,
		Upstreams: strings.Join(dnsServers, " "),
	})
	if err != nil {
		panic(err)
	}

	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      clusterDNSAppName,
			Namespace: "kube-system",
		},
		Data: map[string]string{
			"Corefile": buf.String(),
		},
	}
}

func getClusterDNSService() *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      clusterDNSAppName,
			Namespace: "kube-system",
			Labels: map[string]string{
				CKELabelAppName: clusterDNSAppName,
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				CKELabelAppName: clusterDNSAppName,
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
