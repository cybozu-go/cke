package op

import (
	"context"

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/cke/common"
	"k8s.io/client-go/tools/clientcmd"
)

type schedulerBootOp struct {
	nodes []*cke.Node

	cluster string
	params  cke.ServiceParams

	step  int
	files *common.FilesBuilder
}

// SchedulerBootOp returns an Operator to bootstrap kube-scheduler
func SchedulerBootOp(nodes []*cke.Node, cluster string, params cke.ServiceParams) cke.Operator {
	return &schedulerBootOp{
		nodes:   nodes,
		cluster: cluster,
		params:  params,
		files:   common.NewFilesBuilder(nodes),
	}
}

func (o *schedulerBootOp) Name() string {
	return "kube-scheduler-bootstrap"
}

func (o *schedulerBootOp) NextCommand() cke.Commander {
	switch o.step {
	case 0:
		o.step++
		return common.ImagePullCommand(o.nodes, cke.HyperkubeImage)
	case 1:
		o.step++
		dirs := []string{
			"/var/log/kubernetes/scheduler",
		}
		return common.MakeDirsCommand(o.nodes, dirs)
	case 2:
		o.step++
		return prepareSchedulerFilesCommand{o.cluster, o.files}
	case 3:
		o.step++
		return o.files
	case 4:
		o.step++
		return common.RunContainerCommand(o.nodes, kubeSchedulerContainerName, cke.HyperkubeImage,
			common.WithParams(SchedulerParams()),
			common.WithExtra(o.params))
	default:
		return nil
	}
}

type prepareSchedulerFilesCommand struct {
	cluster string
	files   *common.FilesBuilder
}

func (c prepareSchedulerFilesCommand) Run(ctx context.Context, inf cke.Infrastructure) error {
	const kubeconfigPath = "/etc/kubernetes/scheduler/kubeconfig"
	storage := inf.Storage()

	ca, err := storage.GetCACertificate(ctx, "kubernetes")
	if err != nil {
		return err
	}
	g := func(ctx context.Context, n *cke.Node) ([]byte, error) {
		crt, key, err := cke.KubernetesCA{}.IssueForScheduler(ctx, inf)
		if err != nil {
			return nil, err
		}
		cfg := schedulerKubeconfig(c.cluster, ca, crt, key)
		return clientcmd.Write(*cfg)
	}
	return c.files.AddFile(ctx, kubeconfigPath, g)
}

func (c prepareSchedulerFilesCommand) Command() cke.Command {
	return cke.Command{
		Name: "prepare-scheduler-files",
	}
}

// SchedulerParams returns parameters for kube-scheduler.
func SchedulerParams() cke.ServiceParams {
	args := []string{
		"scheduler",
		"--kubeconfig=/etc/kubernetes/scheduler/kubeconfig",
		"--log-dir=/var/log/kubernetes/scheduler",
		"--logtostderr=false",
	}
	return cke.ServiceParams{
		ExtraArguments: args,
		ExtraBinds: []cke.Mount{
			{
				Source:      "/etc/machine-id",
				Destination: "/etc/machine-id",
				ReadOnly:    true,
				Propagation: "",
				Label:       "",
			},
			{
				Source:      "/etc/kubernetes",
				Destination: "/etc/kubernetes",
				ReadOnly:    true,
				Propagation: "",
				Label:       cke.LabelShared,
			},
			{
				Source:      "/var/log/kubernetes/scheduler",
				Destination: "/var/log/kubernetes/scheduler",
				ReadOnly:    false,
				Propagation: "",
				Label:       cke.LabelPrivate,
			},
		},
	}
}
