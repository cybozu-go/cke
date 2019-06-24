package k8s

import (
	"context"
	"fmt"
	"strings"

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/cke/op"
	"github.com/cybozu-go/cke/op/common"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	// KubeconfigPath is a path for kubeconfig
	KubeconfigPath = "/etc/kubernetes/scheduler/kubeconfig"
	// PolicyConfigPath is a path for scheduler extender policy
	PolicyConfigPath = "/etc/kubernetes/scheduler/policy.cfg.json"
	// SchedulerConfigPath is a path for scheduler extender config
	SchedulerConfigPath = "/etc/kubernetes/scheduler/config.yml"
)

type schedulerBootOp struct {
	nodes []*cke.Node

	cluster string
	params  cke.SchedulerParams

	step  int
	files *common.FilesBuilder
}

// SchedulerBootOp returns an Operator to bootstrap kube-scheduler
func SchedulerBootOp(nodes []*cke.Node, cluster string, params cke.SchedulerParams) cke.Operator {
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
		return prepareSchedulerFilesCommand{o.cluster, o.files, o.params}
	case 2:
		o.step++
		return o.files
	case 3:
		o.step++
		return common.RunContainerCommand(o.nodes, op.KubeSchedulerContainerName, cke.HyperkubeImage,
			common.WithParams(SchedulerParams(len(o.params.Extenders) > 0)),
			common.WithSchedulerExtra(o.params))
	default:
		return nil
	}
}

func (o *schedulerBootOp) Targets() []string {
	ips := make([]string, len(o.nodes))
	for i, n := range o.nodes {
		ips[i] = n.Address
	}
	return ips
}

type prepareSchedulerFilesCommand struct {
	cluster string
	files   *common.FilesBuilder
	params  cke.SchedulerParams
}

func (c prepareSchedulerFilesCommand) Run(ctx context.Context, inf cke.Infrastructure) error {
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
	err = c.files.AddFile(ctx, KubeconfigPath, g)
	if err != nil {
		return err
	}

	policyConfigTmpl := `{
		"kind" : "Policy",
		"apiVersion" : "v1",
		"extenders" :
		  [%s]
	   }
	`
	schedulerConfig := fmt.Sprintf(`apiVersion: kubescheduler.config.k8s.io/v1alpha1
kind: KubeSchedulerConfiguration
schedulerName: default-scheduler
clientConnection:
  kubeconfig: %s
algorithmSource:
  policy:
    file:
      path: %s
leaderElection:
  leaderElect: true
`, KubeconfigPath, PolicyConfigPath)

	if len(c.params.Extenders) != 0 {
		err = c.files.AddFile(ctx, PolicyConfigPath, func(ctx context.Context, n *cke.Node) ([]byte, error) {
			policies := strings.Join(c.params.Extenders, ",")
			return []byte(fmt.Sprintf(policyConfigTmpl, policies)), nil
		})
		if err != nil {
			return err
		}

		return c.files.AddFile(ctx, SchedulerConfigPath, func(ctx context.Context, n *cke.Node) ([]byte, error) {
			return []byte(schedulerConfig), nil
		})
	}
	return nil
}

func (c prepareSchedulerFilesCommand) Command() cke.Command {
	return cke.Command{
		Name: "prepare-scheduler-files",
	}
}

// SchedulerParams returns parameters for kube-scheduler.
func SchedulerParams(withExtender bool) cke.ServiceParams {
	args := []string{
		"scheduler",
		"--kubeconfig=/etc/kubernetes/scheduler/kubeconfig",
		// for healthz service
		"--tls-cert-file=" + op.K8sPKIPath("apiserver.crt"),
		"--tls-private-key-file=" + op.K8sPKIPath("apiserver.key"),
		"--port=0",
	}
	if withExtender {
		args = append(args, "--config=/etc/kubernetes/scheduler/config.yml")
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
		},
	}
}
