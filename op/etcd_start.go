package op

import (
	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/cke/common"
)

type etcdStartOp struct {
	nodes  []*cke.Node
	params cke.EtcdParams
	step   int
	files  *common.FilesBuilder
	domain string
}

// EtcdStartOp returns an Operator to start etcd containers.
func EtcdStartOp(nodes []*cke.Node, params cke.EtcdParams, domain string) cke.Operator {
	return &etcdStartOp{
		nodes:  nodes,
		params: params,
		files:  common.NewFilesBuilder(nodes),
		domain: domain,
	}
}

func (o *etcdStartOp) Name() string {
	return "etcd-start"
}

func (o *etcdStartOp) NextCommand() cke.Commander {
	switch o.step {
	case 0:
		o.step++
		return prepareEtcdCertificatesCommand{o.files, o.domain}
	case 1:
		o.step++
		return o.files
	case 2:
		o.step++
		opts := []string{
			"--mount",
			"type=volume,src=" + etcdVolumeName(o.params) + ",dst=/var/lib/etcd",
		}
		paramsMap := make(map[string]cke.ServiceParams)
		for _, n := range o.nodes {
			paramsMap[n.Address] = EtcdBuiltInParams(n, nil, "")
		}
		return common.RunContainerCommand(o.nodes, etcdContainerName, cke.EtcdImage,
			common.WithOpts(opts),
			common.WithParamsMap(paramsMap),
			common.WithExtra(o.params.ServiceParams))
	case 3:
		o.step++
		return waitEtcdSyncCommand{etcdEndpoints(o.nodes), false}
	default:
		return nil
	}
}
