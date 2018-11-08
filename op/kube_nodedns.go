package op

import (
	"context"
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/util/yaml"

	"github.com/cybozu-go/cke"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var unboundConfTemplate = `
server:
  interface: 0.0.0.0
  chroot: ""
  username: ""
  logfile: ""
  use-syslog: no
  log-time-ascii: yes
  log-queries: yes
  log-replies: yes
  log-local-actions: yes
  log-servfail: yes
  pidfile: "/tmp/unbound.pid"
stub-zone:
  name: "%s"
  stub-addr: %s
forward-zone:
  name: "in-addr.arpa."
  forward-addr: %s
forward-zone:
  name: "ip6.arpa."
  forward-addr: %s
forward-zone:
  name: "."
  %s
`

var daemonSetTemplate = `
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
          image: %s
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
            - name: temporary-volume
              mountPath: /tmp
      volumes:
        - name: config-volume
          configMap:
            name: node-dns
            items:
            - key: unbound.conf
              path: unbound.conf
        - name: temporary-volume
          emptyDir: {}
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
		configMap := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      nodeDNSAppName,
				Namespace: "kube-system",
			},
			Data: map[string]string{
				"unbound.conf": GenerateNodeDNSConfig(c.clusterIP, c.domain, c.dnsServers),
			},
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
		daemonSetText := fmt.Sprintf(daemonSetTemplate, cke.UnboundImage.Name())
		daemonSet := new(appsv1.DaemonSet)
		err = yaml.NewYAMLToJSONDecoder(strings.NewReader(daemonSetText)).Decode(daemonSet)
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
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nodeDNSAppName,
			Namespace: "kube-system",
		},
		Data: map[string]string{
			"unbound.conf": GenerateNodeDNSConfig(c.clusterIP, c.domain, c.dnsServers),
		},
	}
	_, err = configs.Update(configMap)
	if err != nil {
		return err
	}

	// TODO: restart or reload

	return nil
}

func (c updateNodeDNSCommand) Command() cke.Command {
	return cke.Command{
		Name:   "updateNodeDNSCommand",
		Target: "kube-system",
	}
}

func GenerateNodeDNSConfig(clusterIP, domain string, dnsServers []string) string {
	dnsServersText := strings.Join(dnsServers, "\n  ")
	return fmt.Sprintf(unboundConfTemplate, domain, clusterIP, clusterIP, clusterIP, dnsServersText)
}
