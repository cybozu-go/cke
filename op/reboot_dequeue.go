package op

import (
	"context"
	"strconv"

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/cke/metrics"
)

type rebootDequeueOp struct {
	index    int64
	finished bool
}

// RebootDequeueOp returns an Operator to dequeue a reboot entry.
func RebootDequeueOp(index int64) cke.Operator {
	return &rebootDequeueOp{
		index: index,
	}
}

func (o *rebootDequeueOp) Name() string {
	return "rebootDequeue"
}

func (o *rebootDequeueOp) NextCommand() cke.Commander {
	if o.finished {
		return nil
	}

	o.finished = true
	return rebootDequeueCommand{index: o.index}
}

func (o *rebootDequeueOp) Targets() []string {
	return nil
}

type rebootDequeueCommand struct {
	index int64
}

func (c rebootDequeueCommand) Run(ctx context.Context, inf cke.Infrastructure, leaderKey string) error {
	err := inf.Storage().DeleteRebootsEntry(ctx, leaderKey, c.index)
	if err != nil {
		return err
	}
	metrics.DecrementReboot()
	return nil
}

func (c rebootDequeueCommand) Command() cke.Command {
	return cke.Command{
		Name:   "rebootDequeueCommand",
		Target: strconv.FormatInt(c.index, 10),
	}
}
