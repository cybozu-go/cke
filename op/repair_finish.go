package op

import (
	"context"
	"time"

	"github.com/cybozu-go/log"

	"github.com/cybozu-go/cke"
)

type repairFinishOp struct {
	finished bool

	entry     *cke.RepairQueueEntry
	succeeded bool
	cluster   *cke.Cluster
}

func RepairFinishOp(entry *cke.RepairQueueEntry, succeeded bool, cluster *cke.Cluster) cke.Operator {
	return &repairFinishOp{
		entry:     entry,
		succeeded: succeeded,
		cluster:   cluster,
	}
}

func (o *repairFinishOp) Name() string {
	return "repair-finish"
}

func (o *repairFinishOp) NextCommand() cke.Commander {
	if o.finished {
		return nil
	}
	o.finished = true

	return repairFinishCommand{
		entry:     o.entry,
		succeeded: o.succeeded,
		cluster:   o.cluster,
	}
}

func (o *repairFinishOp) Targets() []string {
	return []string{o.entry.Address}
}

type repairFinishCommand struct {
	entry     *cke.RepairQueueEntry
	succeeded bool
	cluster   *cke.Cluster
}

func (c repairFinishCommand) Run(ctx context.Context, inf cke.Infrastructure, _ string) error {
	return repairFinish(ctx, inf, c.entry, c.succeeded, c.cluster)
}

func (c repairFinishCommand) Command() cke.Command {
	return cke.Command{
		Name:   "repairFinishCommand",
		Target: c.entry.Address,
	}
}

func repairFinish(ctx context.Context, inf cke.Infrastructure, entry *cke.RepairQueueEntry, succeeded bool, cluster *cke.Cluster) error {
	if succeeded {
		entry.Status = cke.RepairStatusSucceeded
		// execute Success command
		err := func() error {
			op, err := entry.GetMatchingRepairOperation(cluster)
			if err != nil {
				return err
			}
			if op.SuccessCommand == nil {
				return nil
			}
			timeout := cke.DefaultRepairSuccessCommandTimeoutSeconds
			if op.SuccessCommandTimeout != nil {
				timeout = *op.SuccessCommandTimeout
			}
			_, err = runCommand(ctx, timeout, fnTypeRepair, append(op.SuccessCommand, entry.Address))
			return err
		}()
		if err != nil {
			entry.Status = cke.RepairStatusFailed
			log.Warn("SuccessCommand failed", map[string]any{
				log.FnType:  fnTypeRepair,
				log.FnError: err,
				"index":     entry.Index,
				"address":   entry.Address,
			})
		}
	} else {
		entry.Status = cke.RepairStatusFailed
	}
	entry.LastTransitionTime = time.Now().Truncate(time.Second).UTC()
	return inf.Storage().UpdateRepairsEntry(ctx, entry)
}
