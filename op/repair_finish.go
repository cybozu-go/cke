package op

import (
	"context"
	"time"

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/log"
	"github.com/cybozu-go/well"
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
	if c.succeeded {

	}
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
		//execute Success command
		err := func() error {
			op, err := entry.GetMatchingRepairOperation(cluster)
			if err != nil {
				return err
			}
			if op.SuccessCommand != nil {
				err := func() error {
					ctx := ctx
					timeout := cke.DefaultRepairSuccessCommandTimeoutSeconds
					if op.SuccessCommandTimeout != nil {
						timeout = *op.SuccessCommandTimeout
					}
					if timeout != 0 {
						var cancel context.CancelFunc
						ctx, cancel = context.WithTimeout(ctx, time.Second*time.Duration(timeout))
						defer cancel()
					}
					args := append(op.SuccessCommand[1:], entry.Address)
					command := well.CommandContext(ctx, op.SuccessCommand[0], args...)
					return command.Run()
				}()
				if err != nil {
					return err
				}
			}
			return nil
		}()
		if err != nil {
			entry.Status = cke.RepairStatusFailed
			log.Warn("SuccessCommand failed", map[string]interface{}{
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
