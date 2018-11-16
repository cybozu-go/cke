package op

import (
	"context"
	"strings"

	"github.com/cybozu-go/cke"

	yaml "gopkg.in/yaml.v2"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sYaml "k8s.io/apimachinery/pkg/util/yaml"
)

const confdToml = `
[template]
src = "unbound.conf.tmpl"
dest = "/etc/unbound/unbound.conf"
keys = [ "/" ]
reload_cmd="kill -HUP $(cat /tmp/unbound.pid)"
`

const unboundConfTemplate = `
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
  name: "{{ getv "/domain" }}"
  stub-addr: {{ getv "/cluster_dns" }}
forward-zone:
  name: "in-addr.arpa."
  forward-addr: {{ getv "/cluster_dns" }}
forward-zone:
  name: "ip6.arpa."
  forward-addr: {{ getv "/cluster_dns" }}
{{- if ls "/upstream_dns" }}
forward-zone:
  name: "."
  {{- range getvs "/upstream_dns/*" }}
  forward-addr: {{ . }}
  {{- end }}
{{- end }}
`

var daemonSetText = `
metadata:
  name: node-dns
  namespace: kube-system
  labels:
    k8s-app: node-dns
spec:
  selector:
    matchLabels:
      k8s-app: node-dns
  updateStrategy:
    type: RollingUpdate
    rollingUpdate:
      maxUnavailable: 1
  template:
    metadata:
      labels:
        k8s-app: node-dns
    spec:
      shareProcessNamespace: true
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
            - name: shared-volume
              mountPath: /etc/unbound
            - name: temporary-volume
              mountPath: /tmp
        - name: confd
          image: ` + cke.ConfdImage.Name() + `
          args:
            - "-backend=file"
            - "-file=/etc/confd/kvs.yml"
            - "-interval=5"
          securityContext:
            allowPrivilegeEscalation: false
            capabilities:
              drop:
              - all
            readOnlyRootFilesystem: true
          volumeMounts:
            - name: shared-volume
              mountPath: /etc/unbound
            - name: confd-volume
              mountPath: /etc/confd/conf.d
            - name: confd-volume
              mountPath: /etc/confd
            - name: temporary-volume
              mountPath: /tmp
      initContainers:
        - name: init-confd
          image: ` + cke.ConfdImage.Name() + `
          args:
            - "-backend=file"
            - "-file=/etc/confd/kvs.yml"
            - "-onetime"
            - "-sync-only"
          securityContext:
            allowPrivilegeEscalation: false
            capabilities:
              drop:
              - all
            readOnlyRootFilesystem: true
          volumeMounts:
            - name: shared-volume
              mountPath: /etc/unbound
            - name: confd-volume
              mountPath: /etc/confd/conf.d
            - name: confd-volume
              mountPath: /etc/confd
      volumes:
        - name: shared-volume
          emptyDir: {}
        - name: temporary-volume
          emptyDir: {}
        - name: confd-volume
          configMap:
            name: node-dns
            items:
            - key: unbound.toml
              path: unbound.toml
            - key: unbound.conf.tmpl
              path: templates/unbound.conf.tmpl
            - key: kvs.yml
              path: kvs.yml
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
		configMap, err := GenerateNodeDNSConfig(c.clusterIP, c.domain, c.dnsServers)
		if err != nil {
			return err
		}
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
		err = k8sYaml.NewYAMLToJSONDecoder(strings.NewReader(daemonSetText)).Decode(daemonSet)
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

type kubeNodeDNSUpdateOp struct {
	apiserver  *cke.Node
	clusterIP  string
	domain     string
	dnsServers []string
	finished   bool
}

// KubeNodeDNSUpdateOp returns an Operator to update unbound as Node local resolver.
func KubeNodeDNSUpdateOp(apiserver *cke.Node, clusterIP, domain string, dnsServers []string) cke.Operator {
	return &kubeNodeDNSUpdateOp{
		apiserver:  apiserver,
		clusterIP:  clusterIP,
		domain:     domain,
		dnsServers: dnsServers,
	}
}

func (o *kubeNodeDNSUpdateOp) Name() string {
	return "update-node-dns"
}

func (o *kubeNodeDNSUpdateOp) NextCommand() cke.Commander {
	if o.finished {
		return nil
	}
	o.finished = true
	return updateNodeDNSCommand{o.apiserver, o.clusterIP, o.domain, o.dnsServers}
}

type updateNodeDNSCommand struct {
	apiserver  *cke.Node
	clusterIP  string
	domain     string
	dnsServers []string
}

func (c updateNodeDNSCommand) Run(ctx context.Context, inf cke.Infrastructure) error {
	cs, err := inf.K8sClient(ctx, c.apiserver)
	if err != nil {
		return err
	}

	configs := cs.CoreV1().ConfigMaps("kube-system")
	configMap, err := GenerateNodeDNSConfig(c.clusterIP, c.domain, c.dnsServers)
	if err != nil {
		return err
	}
	_, err = configs.Update(configMap)
	if err != nil {
		return err
	}

	return nil
}

func (c updateNodeDNSCommand) Command() cke.Command {
	return cke.Command{
		Name:   "updateNodeDNSCommand",
		Target: "kube-system",
	}
}

// GenerateNodeDNSConfig returns ConfigMap of node-dns
func GenerateNodeDNSConfig(clusterIP, domain string, dnsServers []string) (*corev1.ConfigMap, error) {
	kvs := map[string]interface{}{
		"domain":       domain,
		"cluster_dns":  clusterIP,
		"upstream_dns": dnsServers,
	}
	kvsBytes, err := yaml.Marshal(kvs)
	if err != nil {
		return nil, err
	}
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nodeDNSAppName,
			Namespace: "kube-system",
		},
		Data: map[string]string{
			"unbound.toml":      confdToml,
			"unbound.conf.tmpl": unboundConfTemplate,
			"kvs.yml":           string(kvsBytes),
		},
	}, nil
}
