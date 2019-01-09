package nodedns

import (
	"context"
	"strings"

	"github.com/cybozu-go/cke"
	"k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
)

type updateDaemonSetOp struct {
	apiserver *cke.Node
	finished  bool
}

// UpdateDaemonSetOp returns an Operator to update unbound daemonset.
func UpdateDaemonSetOp(apiserver *cke.Node) cke.Operator {
	return &updateDaemonSetOp{
		apiserver: apiserver,
	}
}

func (o *updateDaemonSetOp) Name() string {
	return "update-node-dns-daemonset"
}

func (o *updateDaemonSetOp) NextCommand() cke.Commander {
	if o.finished {
		return nil
	}
	o.finished = true
	return updateDaemonSetCommand{o.apiserver}
}

type updateDaemonSetCommand struct {
	apiserver *cke.Node
}

func (c updateDaemonSetCommand) Run(ctx context.Context, inf cke.Infrastructure) error {
	cs, err := inf.K8sClient(ctx, c.apiserver)
	if err != nil {
		return err
	}

	daemonSet := new(v1.DaemonSet)
	err = yaml.NewYAMLToJSONDecoder(strings.NewReader(unboundDaemonSetText)).Decode(daemonSet)
	if err != nil {
		return err
	}

	_, err = cs.AppsV1().DaemonSets("kube-system").Update(daemonSet)
	return err
}

func (c updateDaemonSetCommand) Command() cke.Command {
	return cke.Command{
		Name:   "updateNodeDNSDaemonSet",
		Target: "kube-system/node-dns",
	}
}
