package op

import (
	"bytes"
	"context"
	"strings"
	"text/template"

	"github.com/cybozu-go/cke"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sYaml "k8s.io/apimachinery/pkg/util/yaml"
)

type unboundConfigTemplate struct {
	Domain    string
	ClusterIP string
	Upstreams []string
}

const unboundConfigTemplateText = `
server:
  interface: 0.0.0.0
  interface-automatic: yes
  access-control: 0.0.0.0/0 allow
  chroot: ""
  username: ""
  directory: "/etc/unbound"
  logfile: ""
  use-syslog: no
  log-time-ascii: yes
  log-queries: yes
  log-replies: yes
  log-local-actions: yes
  log-servfail: yes
  pidfile: "/tmp/unbound.pid"
  infra-host-ttl: 60
  prefetch: yes
stub-zone:
  name: "{{ .Domain }}"
  stub-addr: {{ .ClusterIP }}
forward-zone:
  name: "in-addr.arpa."
  forward-addr: {{ .ClusterIP }}
forward-zone:
  name: "ip6.arpa."
  forward-addr: {{ .ClusterIP }}
{{- if .Upstreams }}
forward-zone:
  name: "."
  {{- range .Upstreams }}
  forward-addr: {{ . }}
  {{- end }}
{{- end }}
`

const UnboundTemplateVersion = "1"

var unboundDaemonSetText = `
metadata:
  name: node-dns
  namespace: kube-system
  annotations:
    cke.cybozu.com/image: ` + cke.UnboundImage.Name() + `
    cke.cybozu.com/template-version: ` + UnboundTemplateVersion + `
spec:
  selector:
    matchLabels:
      cke.cybozu.com/appname: node-dns
  updateStrategy:
    type: RollingUpdate
    rollingUpdate:
      maxUnavailable: 1
  template:
    metadata:
      labels:
        cke.cybozu.com/appname: node-dns
    spec:
      priorityClassName: system-node-critical
      nodeSelector:
        beta.kubernetes.io/os: linux
      hostNetwork: true
      tolerations:
        # Make sure unbound gets scheduled on all nodes.
        - effect: NoSchedule
          operator: Exists
        # Mark the pod as a critical add-on for rescheduling.
        - key: CriticalAddonsOnly
          operator: Exists
        - effect: NoExecute
          operator: Exists
      terminationGracePeriodSeconds: 0
      containers:
        - name: unbound
          image: ` + cke.UnboundImage.Name() + `
          args:
            - -c
            - /etc/unbound/unbound.conf
          securityContext:
            allowPrivilegeEscalation: false
            capabilities:
              add:
              - NET_BIND_SERVICE
              drop:
              - all
            readOnlyRootFilesystem: true
          resources:
            limits:
              memory: 170Mi
            requests:
              cpu: 100m
              memory: 70Mi
          livenessProbe:
            tcpSocket:
              port: 53
              host: localhost
            periodSeconds: 1
            initialDelaySeconds: 1
            failureThreshold: 6
          volumeMounts:
            - name: config-volume
              mountPath: /etc/unbound
        - name: reload
          image: ` + cke.UnboundImage.Name() + `
          command:
          - /usr/local/bin/reload-unbound
          securityContext:
            allowPrivilegeEscalation: false
            capabilities:
              drop:
              - all
            readOnlyRootFilesystem: true
          volumeMounts:
            - name: config-volume
              mountPath: /etc/unbound
      volumes:
        - name: config-volume
          configMap:
            name: node-dns
            items:
            - key: unbound.conf
              path: unbound.conf
`

type kubeNodeDNSCreateOp struct {
	apiserver  *cke.Node
	clusterIP  string
	domain     string
	dnsServers []string
	finished   bool
}

// KubeNodeDNSCreateOp returns an Operator to create unbound as Node local resolver.
func KubeNodeDNSCreateOp(apiserver *cke.Node, clusterIP, domain string, dnsServers []string) cke.Operator {
	return &kubeNodeDNSCreateOp{
		apiserver:  apiserver,
		clusterIP:  clusterIP,
		domain:     domain,
		dnsServers: dnsServers,
	}
}

func (o *kubeNodeDNSCreateOp) Name() string {
	return "create-node-dns"
}

func (o *kubeNodeDNSCreateOp) NextCommand() cke.Commander {
	if o.finished {
		return nil
	}
	o.finished = true
	return createNodeDNSCommand{o.apiserver, o.clusterIP, o.domain, o.dnsServers}
}

type createNodeDNSCommand struct {
	apiserver  *cke.Node
	clusterIP  string
	domain     string
	dnsServers []string
}

func (c createNodeDNSCommand) Run(ctx context.Context, inf cke.Infrastructure) error {
	cs, err := inf.K8sClient(ctx, c.apiserver)
	if err != nil {
		return err
	}

	// ConfigMap
	configs := cs.CoreV1().ConfigMaps("kube-system")
	_, err = configs.Get(nodeDNSAppName, metav1.GetOptions{})
	switch {
	case err == nil:
	case errors.IsNotFound(err):
		configMap := GenerateNodeDNSConfig(c.clusterIP, c.domain, c.dnsServers)
		_, err = configs.Create(configMap)
		if err != nil {
			return err
		}
	default:
		return err
	}

	// DaemonSet
	daemonSets := cs.AppsV1().DaemonSets("kube-system")
	_, err = daemonSets.Get(nodeDNSAppName, metav1.GetOptions{})
	switch {
	case err == nil:
	case errors.IsNotFound(err):
		daemonSet := new(appsv1.DaemonSet)
		err = k8sYaml.NewYAMLToJSONDecoder(strings.NewReader(unboundDaemonSetText)).Decode(daemonSet)
		if err != nil {
			return err
		}
		_, err = daemonSets.Create(daemonSet)
		if err != nil {
			return err
		}
	default:
		return err
	}

	return nil
}

func (c createNodeDNSCommand) Command() cke.Command {
	return cke.Command{
		Name:   "createNodeDNSCommand",
		Target: "kube-system",
	}
}

type kubeNodeDNSUpdateConfigMapOp struct {
	apiserver *cke.Node
	configMap *corev1.ConfigMap
	finished  bool
}

// KubeNodeDNSUpdateConfigMapOp returns an Operator to update unbound as Node local resolver.
func KubeNodeDNSUpdateConfigMapOp(apiserver *cke.Node, configMap *corev1.ConfigMap) cke.Operator {
	return &kubeNodeDNSUpdateConfigMapOp{
		apiserver: apiserver,
		configMap: configMap,
	}
}

func (o *kubeNodeDNSUpdateConfigMapOp) Name() string {
	return "update-node-dns-configmap"
}

func (o *kubeNodeDNSUpdateConfigMapOp) NextCommand() cke.Commander {
	if o.finished {
		return nil
	}
	o.finished = true
	return updateNodeDNSConfigMapCommand{o.apiserver, o.configMap}
}

type updateNodeDNSConfigMapCommand struct {
	apiserver *cke.Node
	configMap *corev1.ConfigMap
}

func (c updateNodeDNSConfigMapCommand) Run(ctx context.Context, inf cke.Infrastructure) error {
	cs, err := inf.K8sClient(ctx, c.apiserver)
	if err != nil {
		return err
	}

	configs := cs.CoreV1().ConfigMaps("kube-system")
	_, err = configs.Update(c.configMap)
	return err
}

func (c updateNodeDNSConfigMapCommand) Command() cke.Command {
	return cke.Command{
		Name:   "updateNodeDNSConfigMapCommand",
		Target: "kube-system",
	}
}

// GenerateNodeDNSConfig returns ConfigMap of node-dns
func GenerateNodeDNSConfig(clusterIP, domain string, dnsServers []string) *corev1.ConfigMap {
	var confTempl unboundConfigTemplate
	confTempl.Domain = domain
	confTempl.ClusterIP = clusterIP
	confTempl.Upstreams = dnsServers

	tmpl := template.Must(template.New("").Parse(unboundConfigTemplateText))
	unboundConf := new(bytes.Buffer)
	err := tmpl.Execute(unboundConf, confTempl)
	if err != nil {
		panic(err)
	}
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nodeDNSAppName,
			Namespace: "kube-system",
		},
		Data: map[string]string{
			"unbound.conf": unboundConf.String(),
		},
	}
}
