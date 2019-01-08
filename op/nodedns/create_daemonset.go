package nodedns

import (
	"context"
	"strings"

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/cke/op"
	v12 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
)

type createDaemonSetOp struct {
	apiserver *cke.Node
	finished  bool
}

// CreateDaemonSetOp returns an Operator to create unbound daemonset.
func CreateDaemonSetOp(apiserver *cke.Node) cke.Operator {
	return &createDaemonSetOp{
		apiserver: apiserver,
	}
}

func (o *createDaemonSetOp) Name() string {
	return "create-node-dns-daemonset"
}

func (o *createDaemonSetOp) NextCommand() cke.Commander {
	if o.finished {
		return nil
	}
	o.finished = true
	return createDaemonSetCommand{o.apiserver}
}

type createDaemonSetCommand struct {
	apiserver *cke.Node
}

func (c createDaemonSetCommand) Run(ctx context.Context, inf cke.Infrastructure) error {
	cs, err := inf.K8sClient(ctx, c.apiserver)
	if err != nil {
		return err
	}

	// DaemonSet
	daemonSets := cs.AppsV1().DaemonSets("kube-system")
	_, err = daemonSets.Get(op.NodeDNSAppName, v1.GetOptions{})
	switch {
	case err == nil:
	case errors.IsNotFound(err):
		daemonSet := new(v12.DaemonSet)
		err = yaml.NewYAMLToJSONDecoder(strings.NewReader(unboundDaemonSetText)).Decode(daemonSet)
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

func (c createDaemonSetCommand) Command() cke.Command {
	return cke.Command{
		Name:   "createDaemonSetCommand",
		Target: "kube-system",
	}
}
