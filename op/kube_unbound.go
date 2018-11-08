package op

import (
	"context"

	"github.com/cybozu-go/cke"
)

type kubeUnboundCreateOp struct {
	apiserver  *cke.Node
	domain     string
	dnsServers []string
	finished   bool
}

var configMapTemplate = `
`
var serviceTemplate = `
apiVersion: v1
kind: ServiceAccount
metadata:
  name: node-local-resolver
namespace: kube-system
`
var daemonsetTemplate = `
metadata:
  name: node-local-resolver
  namespace: kube-system
  labels:
    k8s-app: node-local-resolver
spec:
  selector:
    matchLabels:
      k8s-app: node-local-resolver
  updateStrategy:
    type: RollingUpdate
    rollingUpdate:
      maxUnavailable: 1
  template:
    metadata:
      labels:
        k8s-app: node-local-resolver
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
      serviceAccountName: node-local-resolver
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
      volumes:
        - name: config-volume
          configMap:
            name: unbound
            items:
            - key: unbound.conf
              path: unbound.conf
`

// KubeUnboundCreateOp returns an Operator to create unbound as Node local resolver.
func KubeUnboundCreateOp(apiserver *cke.Node, domain string, dnsServers []string) cke.Operator {
	return &kubeUnboundCreateOp{
		apiserver:  apiserver,
		domain:     domain,
		dnsServers: dnsServers,
	}
}

func (o *kubeUnboundCreateOp) Name() string {
	return "create-unbound"
}

func (o *kubeUnboundCreateOp) NextCommand() cke.Commander {
	if o.finished {
		return nil
	}
	o.finished = true
	return createUnboundCommand{o.apiserver, o.domain, o.dnsServers}
}

type createUnboundCommand struct {
	apiserver  *cke.Node
	domain     string
	dnsServers []string
}

func (c createUnboundCommand) Run(ctx context.Context, inf cke.Infrastructure) error {
	// DaemonSet

	return nil
}

func (c createUnboundCommand) Command() cke.Command {
	return cke.Command{
		Name:   "createUnboundCommand",
		Target: "kube-system",
	}
}

type kubeUnboundUpdateOp struct {
	apiserver  *cke.Node
	domain     string
	dnsServers []string
	finished   bool
}

// KubeUnboundUpdateOp returns an Operator to update unbound as Node local resolver.
func KubeUnboundUpdateOp(apiserver *cke.Node, domain string, dnsServers []string) cke.Operator {
	return &kubeUnboundUpdateOp{
		apiserver:  apiserver,
		domain:     domain,
		dnsServers: dnsServers,
	}
}

func (o *kubeUnboundUpdateOp) Name() string {
	return "update-unbound"
}

func (o *kubeUnboundUpdateOp) NextCommand() cke.Commander {
	if o.finished {
		return nil
	}
	o.finished = true
	return updateUnboundCommand{o.apiserver, o.domain, o.dnsServers}
}

type updateUnboundCommand struct {
	apiserver  *cke.Node
	domain     string
	dnsServers []string
}

func (c updateUnboundCommand) Run(ctx context.Context, inf cke.Infrastructure) error {
	return nil
}

func (c updateUnboundCommand) Command() cke.Command {
	return cke.Command{
		Name:   "updateUnboundCommand",
		Target: "kube-system",
	}
}
