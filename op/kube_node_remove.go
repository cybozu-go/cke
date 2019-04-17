package op

import (
	"context"

	"github.com/cybozu-go/cke"
	corev1 "k8s.io/api/core/v1"
)

type kubeNodeRemove struct {
	apiserver *cke.Node
	nodes     []*corev1.Node
	done      bool
}

// KubeNodeRemoveOp removes k8s Node resources.
func KubeNodeRemoveOp(apiserver *cke.Node, nodes []*corev1.Node) cke.Operator {
	return &kubeNodeRemove{apiserver: apiserver, nodes: nodes}
}

func (o *kubeNodeRemove) Name() string {
	return "remove-node"
}

func (o *kubeNodeRemove) NextCommand() cke.Commander {
	if o.done {
		return nil
	}

	o.done = true
	return nodeRemoveCommand{o.apiserver, o.nodes}
}

func (o *kubeNodeRemove) Nodes() []string {
	ips := []string{}
	for _, n := range o.nodes {
		ips = append(ips, n.Name)
	}
	return ips
}

type nodeRemoveCommand struct {
	apiserver *cke.Node
	nodes     []*corev1.Node
}

func (c nodeRemoveCommand) Run(ctx context.Context, inf cke.Infrastructure) error {
	cs, err := inf.K8sClient(ctx, c.apiserver)
	if err != nil {
		return err
	}

	nodesAPI := cs.CoreV1().Nodes()
	for _, n := range c.nodes {
		err := nodesAPI.Delete(n.Name, nil)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c nodeRemoveCommand) Command() cke.Command {
	return cke.Command{
		Name: "removeNode",
	}
}
