package op

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/log"
	"github.com/cybozu-go/well"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
)

type kubeNodeRemove struct {
	apiserver *cke.Node
	nodes     []*corev1.Node
	config    *cke.Retire
	done      bool
}

// KubeNodeRemoveOp removes k8s Node resources.
func KubeNodeRemoveOp(apiserver *cke.Node, nodes []*corev1.Node, config *cke.Retire) cke.Operator {
	return &kubeNodeRemove{apiserver: apiserver, nodes: nodes, config: config}
}

func (o *kubeNodeRemove) Name() string {
	return "remove-node"
}

func (o *kubeNodeRemove) NextCommand() cke.Commander {
	if o.done {
		return nil
	}

	o.done = true
	return nodeRemoveCommand{
		o.apiserver,
		o.nodes,
		o.config.ShutdownCommand,
		o.config.CheckCommand,
		o.config.CommandTimeoutSeconds,
		o.config.CheckTimeoutSeconds,
	}
}

func (o *kubeNodeRemove) Targets() []string {
	return []string{
		o.apiserver.Address,
	}
}

type nodeRemoveCommand struct {
	apiserver           *cke.Node
	nodes               []*corev1.Node
	shutdownCommand     []string
	checkCommand        []string
	timeoutSeconds      *int
	checkTimeoutSeconds *int
}

func (c nodeRemoveCommand) Run(ctx context.Context, inf cke.Infrastructure, _ string) error {
	cs, err := inf.K8sClient(ctx, c.apiserver)
	if err != nil {
		return err
	}
	nodesAPI := cs.CoreV1().Nodes()
	for _, n := range c.nodes {
		if !n.DeletionTimestamp.IsZero() {
			continue
		}
		if !n.Spec.Unschedulable {
			oldData, err := json.Marshal(n)
			if err != nil {
				return err
			}
			n.Spec.Unschedulable = true
			newData, err := json.Marshal(n)
			if err != nil {
				return err
			}
			patchBytes, err := strategicpatch.CreateTwoWayMergePatch(oldData, newData, n)
			if err != nil {
				return fmt.Errorf("failed to create patch for node %s: %v", n.Name, err)
			}
			_, err = nodesAPI.Patch(ctx, n.Name, types.StrategicMergePatchType, patchBytes, metav1.PatchOptions{})
			if err != nil {
				return fmt.Errorf("failed to patch node %s: %v", n.Name, err)
			}
		}
		err := func() error {
			ctx := ctx
			timeout := cke.DefaultRetireCommandTimeoutSeconds
			if c.timeoutSeconds != nil {
				timeout = *c.timeoutSeconds
			}
			if timeout != 0 {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, time.Second*time.Duration(timeout))
				defer cancel()
			}
			args := append(c.shutdownCommand[1:], n.Name)
			command := well.CommandContext(ctx, c.shutdownCommand[0], args...)
			return command.Run()
		}()
		if err != nil {
			return fmt.Errorf("failed to shutdown node %s: %v", n.Name, err)
		}

		err = func() error {
			ctx := ctx
			checkTimeout := cke.DefaultRetireCheckTimeoutSeconds
			if c.checkTimeoutSeconds != nil {
				checkTimeout = *c.checkTimeoutSeconds
			}
			timeout := time.After(time.Duration(checkTimeout) * time.Second)
			ticker := time.NewTicker(10 * time.Second)
			for {
				select {
				case <-timeout:
					return fmt.Errorf("timeout")
				case <-ticker.C:
					args := append(c.checkCommand[1:], n.Name)
					command := well.CommandContext(ctx, c.checkCommand[0], args...)
					stdout, err := command.Output()
					if err != nil {
						log.Warn("failed to check shutdown status of node", map[string]interface{}{
							log.FnError: err,
							"node":      n.Name,
						})
						continue
					}
					if strings.TrimSuffix(string(stdout), "\n") == "Off" {
						return nil
					}
				}
			}
		}()
		if err != nil {
			return fmt.Errorf("failed to check shutdown status of node %s: %v", n.Name, err)
		}
		shutdownTaint := corev1.Taint{
			Key:    "node.kubernetes.io/out-of-service",
			Value:  "nodeshutdown",
			Effect: corev1.TaintEffectNoExecute,
		}
		n.Spec.Taints = append(n.Spec.Taints, shutdownTaint)
		_, err = nodesAPI.Update(ctx, n, metav1.UpdateOptions{})
		if err != nil {
			return fmt.Errorf("failed to update node %s: %v", n.Name, err)
		}

		err = nodesAPI.Delete(ctx, n.Name, metav1.DeleteOptions{})
		if err != nil {
			return fmt.Errorf("failed to delete node %s: %v", n.Name, err)
		}
	}
	return nil
}

func (c nodeRemoveCommand) Command() cke.Command {
	names := make([]string, len(c.nodes))
	for i, n := range c.nodes {
		names[i] = n.Name
	}
	return cke.Command{
		Name:   "removeNode",
		Target: strings.Join(names, ","),
	}
}
