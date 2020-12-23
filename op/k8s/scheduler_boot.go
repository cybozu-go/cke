package k8s

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/cke/op"
	"github.com/cybozu-go/cke/op/common"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/yaml"

	schedulerv1alpha1 "k8s.io/kube-scheduler/config/v1alpha1"
	schedulerv1alpha2 "k8s.io/kube-scheduler/config/v1alpha2"
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
		return common.ImagePullCommand(o.nodes, cke.KubernetesImage)
	case 1:
		o.step++
		return prepareSchedulerFilesCommand{o.cluster, o.files, o.params}
	case 2:
		o.step++
		return o.files
	case 3:
		o.step++
		return common.RunContainerCommand(o.nodes, op.KubeSchedulerContainerName, cke.KubernetesImage,
			common.WithParams(SchedulerParams()),
			common.WithExtra(o.params.ServiceParams))
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

func (c prepareSchedulerFilesCommand) Run(ctx context.Context, inf cke.Infrastructure, _ string) error {
	storage := inf.Storage()

	ca, err := storage.GetCACertificate(ctx, cke.CAKubernetes)
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
	err = c.files.AddFile(ctx, op.SchedulerKubeConfigPath, g)
	if err != nil {
		return err
	}

	version, err := c.params.GetAPIversion()
	if err != nil {
		return err
	}
	switch version {
	case schedulerv1alpha1.SchemeGroupVersion.String():
		// Create v1 Policy for scheduler extender
		err := c.files.AddFile(ctx, op.PolicyConfigPath, func(ctx context.Context, n *cke.Node) ([]byte, error) {
			policy, err := GenerateSchedulerPolicyV1(c.params)
			if err != nil {
				return nil, err
			}
			return json.Marshal(policy)
		})
		if err != nil {
			return err
		}

		// Create v1alpha1 KubeSchedulerConfiguration
		return c.files.AddFile(ctx, op.SchedulerConfigPath, func(ctx context.Context, n *cke.Node) ([]byte, error) {
			return []byte(fmt.Sprintf(`apiVersion: kubescheduler.config.k8s.io/v1alpha1
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
`, op.SchedulerKubeConfigPath, op.PolicyConfigPath)), nil
		})

	case schedulerv1alpha2.SchemeGroupVersion.String():
		return c.files.AddFile(ctx, op.SchedulerConfigPath, func(ctx context.Context, n *cke.Node) ([]byte, error) {
			cfg := GenerateSchedulerConfigurationV1Alpha2(c.params)
			return yaml.Marshal(cfg)
		})
	default:
		return errors.New("unsupported scheduler API version was given: " + version)
	}
}

func (c prepareSchedulerFilesCommand) Command() cke.Command {
	return cke.Command{
		Name: "prepare-scheduler-files",
	}
}

// SchedulerParams returns parameters for kube-scheduler.
func SchedulerParams() cke.ServiceParams {
	args := []string{
		"kube-scheduler",
		"--config=" + op.SchedulerConfigPath,
		// for healthz service
		"--tls-cert-file=" + op.K8sPKIPath("apiserver.crt"),
		"--tls-private-key-file=" + op.K8sPKIPath("apiserver.key"),
		"--port=0",
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
