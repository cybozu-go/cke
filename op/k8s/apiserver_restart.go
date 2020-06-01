package k8s

import (
	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/cke/op"
	"github.com/cybozu-go/cke/op/common"
)

type apiServerRestartOp struct {
	nodes []*cke.Node
	cps   []*cke.Node

	serviceSubnet string
	domain        string
	params        cke.APIServerParams

	step  int
	files *common.FilesBuilder
}

// APIServerRestartOp returns an Operator to restart kube-apiserver
func APIServerRestartOp(nodes, cps []*cke.Node, serviceSubnet, domain string, params cke.APIServerParams) cke.Operator {
	return &apiServerRestartOp{
		nodes:         nodes,
		cps:           cps,
		serviceSubnet: serviceSubnet,
		domain:        domain,
		params:        params,
		files:         common.NewFilesBuilder(nodes),
	}
}

func (o *apiServerRestartOp) Name() string {
	return "kube-apiserver-restart"
}

func (o *apiServerRestartOp) NextCommand() cke.Commander {
	switch o.step {
	case 0:
		o.step++
		return common.ImagePullCommand(o.nodes, cke.KubernetesImage)
	case 1:
		o.step++
		return prepareAPIServerFilesCommand{o.files, o.serviceSubnet, o.domain, o.params}
	case 2:
		o.step++
		return o.files
	case 3:
		o.step++
		return common.StopContainersCommand(o.nodes, op.KubeAPIServerContainerName)
	case 4:
		if len(o.nodes) == 0 {
			return nil
		}
		opts := []string{
			"--mount", "type=tmpfs,dst=/run/kubernetes",
		}
		paramsMap := make(map[string]cke.ServiceParams)
		for _, n := range o.nodes {
			paramsMap[n.Address] = APIServerParams(o.cps, n.Address, o.serviceSubnet, o.params.AuditLogEnabled, o.params.AuditLogPolicy)
		}
		return common.RunContainerCommand(o.nodes,
			op.KubeAPIServerContainerName, cke.KubernetesImage,
			common.WithOpts(opts),
			common.WithParamsMap(paramsMap),
			common.WithExtra(o.params.ServiceParams))
	}

	panic("unreachable")
}

func (o *apiServerRestartOp) Targets() []string {
	ips := make([]string, len(o.nodes))
	for i, n := range o.nodes {
		ips[i] = n.Address
	}
	return ips
}
