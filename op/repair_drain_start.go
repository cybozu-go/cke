package op

import (
	"context"
	"fmt"
	"time"

	"github.com/cybozu-go/cke"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

type repairDrainStartOp struct {
	finished bool

	entry     *cke.RepairQueueEntry
	config    *cke.Repair
	apiserver *cke.Node
}

func RepairDrainStartOp(apiserver *cke.Node, entry *cke.RepairQueueEntry, config *cke.Repair) cke.Operator {
	return &repairDrainStartOp{
		entry:     entry,
		config:    config,
		apiserver: apiserver,
	}
}

func (o *repairDrainStartOp) Name() string {
	return "repair-drain-start"
}

func (o *repairDrainStartOp) NextCommand() cke.Commander {
	if o.finished {
		return nil
	}
	o.finished = true

	attempts := 1
	if o.config.EvictRetries != nil {
		attempts = *o.config.EvictRetries + 1
	}
	interval := 0 * time.Second
	if o.config.EvictInterval != nil {
		interval = time.Second * time.Duration(*o.config.EvictInterval)
	}

	return repairDrainStartCommand{
		entry:               o.entry,
		protectedNamespaces: o.config.ProtectedNamespaces,
		apiserver:           o.apiserver,
		evictAttempts:       attempts,
		evictInterval:       interval,
	}
}

func (o *repairDrainStartOp) Targets() []string {
	return []string{o.entry.Address}
}

type repairDrainStartCommand struct {
	entry               *cke.RepairQueueEntry
	protectedNamespaces *metav1.LabelSelector
	apiserver           *cke.Node
	evictAttempts       int
	evictInterval       time.Duration
}

func (c repairDrainStartCommand) Run(ctx context.Context, inf cke.Infrastructure, _ string) error {
	cs, err := inf.K8sClient(ctx, c.apiserver)
	if err != nil {
		return err
	}
	nodesAPI := cs.CoreV1().Nodes()

	protected, err := listProtectedNamespaces(ctx, cs, c.protectedNamespaces)
	if err != nil {
		return err
	}

	err = func() error {
		c.entry.Status = cke.RepairStatusProcessing
		c.entry.StepStatus = cke.RepairStepStatusDraining
		c.entry.LastTransitionTime = time.Now().Truncate(time.Second).UTC()
		err := inf.Storage().UpdateRepairsEntry(ctx, c.entry)
		if err != nil {
			return err
		}

		err = checkJobPodNotExist(ctx, cs, c.entry.Nodename)
		if err != nil {
			return err
		}

		// Note: The annotation name is shared with reboot operations.
		_, err = nodesAPI.Patch(ctx, c.entry.Nodename, types.StrategicMergePatchType, []byte(`
{
	"metadata":{"annotations":{"`+CKEAnnotationReboot+`": "true"}},
	"spec":{"unschedulable": true}
}
`), metav1.PatchOptions{})
		if err != nil {
			return fmt.Errorf("failed to cordon node %s: %v", c.entry.Address, err)
		}

		return nil
	}()
	if err != nil {
		return repairDrainBackOff(ctx, inf, c.entry, err)
	}

	err = evictOrDeleteNodePod(ctx, cs, c.entry.Nodename, protected, c.evictAttempts, c.evictInterval)
	if err != nil {
		return repairDrainBackOff(ctx, inf, c.entry, err)
	}

	return nil
}

func (c repairDrainStartCommand) Command() cke.Command {
	return cke.Command{
		Name:   "repairDrainStartCommand",
		Target: c.entry.Address,
	}
}
