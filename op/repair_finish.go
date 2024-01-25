package op

import (
	"context"
	"time"

	"github.com/cybozu-go/cke"
)

type repairFinishOp struct {
	finished bool

	entry     *cke.RepairQueueEntry
	succeeded bool
}

func RepairFinishOp(entry *cke.RepairQueueEntry, succeeded bool) cke.Operator {
	return &repairFinishOp{
		entry:     entry,
		succeeded: succeeded,
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
	}
}

func (o *repairFinishOp) Targets() []string {
	return []string{o.entry.Address}
}

type repairFinishCommand struct {
	entry     *cke.RepairQueueEntry
	succeeded bool
}

func (c repairFinishCommand) Run(ctx context.Context, inf cke.Infrastructure, _ string) error {
	return repairFinish(ctx, inf, c.entry, c.succeeded)
}

func (c repairFinishCommand) Command() cke.Command {
	return cke.Command{
		Name:   "repairFinishCommand",
		Target: c.entry.Address,
	}
}

func repairFinish(ctx context.Context, inf cke.Infrastructure, entry *cke.RepairQueueEntry, succeeded bool) error {
	if succeeded {
		entry.Status = cke.RepairStatusSucceeded
	} else {
		entry.Status = cke.RepairStatusFailed
	}
	entry.LastTransitionTime = time.Now().Truncate(time.Second).UTC()
	return inf.Storage().UpdateRepairsEntry(ctx, entry)
}
