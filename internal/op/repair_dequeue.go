package op

import (
	"context"

	"github.com/cybozu-go/cke"
)

type repairDequeueOp struct {
	finished bool

	entry *cke.RepairQueueEntry
}

func RepairDequeueOp(entry *cke.RepairQueueEntry) cke.Operator {
	return &repairDequeueOp{
		entry: entry,
	}
}

func (o *repairDequeueOp) Name() string {
	return "repair-dequeue"
}

func (o *repairDequeueOp) NextCommand() cke.Commander {
	if o.finished {
		return nil
	}

	o.finished = true
	return repairDequeueCommand{
		entry: o.entry,
	}
}

func (o *repairDequeueOp) Targets() []string {
	return []string{o.entry.Address}
}

type repairDequeueCommand struct {
	entry *cke.RepairQueueEntry
}

func (c repairDequeueCommand) Run(ctx context.Context, inf cke.Infrastructure, leaderKey string) error {
	return inf.Storage().DeleteRepairsEntry(ctx, leaderKey, c.entry.Index)
}

func (c repairDequeueCommand) Command() cke.Command {
	return cke.Command{
		Name:   "repairDequeueCommand",
		Target: c.entry.Address,
	}
}
