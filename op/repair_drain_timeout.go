package op

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/log"
)

type repairDrainTimeoutOp struct {
	finished bool

	entry *cke.RepairQueueEntry
}

func RepairDrainTimeoutOp(entry *cke.RepairQueueEntry) cke.Operator {
	return &repairDrainTimeoutOp{
		entry: entry,
	}
}

func (o *repairDrainTimeoutOp) Name() string {
	return "repair-drain-timeout"
}

func (o *repairDrainTimeoutOp) NextCommand() cke.Commander {
	if o.finished {
		return nil
	}
	o.finished = true

	return repairDrainTimeoutCommand{
		entry: o.entry,
	}
}

func (o *repairDrainTimeoutOp) Targets() []string {
	return []string{o.entry.Address}
}

type repairDrainTimeoutCommand struct {
	entry *cke.RepairQueueEntry
}

func (c repairDrainTimeoutCommand) Run(ctx context.Context, inf cke.Infrastructure, _ string) error {
	return repairDrainBackOff(ctx, inf, c.entry, fmt.Errorf("drain timed out: %s", c.entry.Address))
}

func (c repairDrainTimeoutCommand) Command() cke.Command {
	return cke.Command{
		Name:   "repairDrainTimeoutCommand",
		Target: c.entry.Address,
	}
}

func repairDrainBackOff(ctx context.Context, inf cke.Infrastructure, entry *cke.RepairQueueEntry, err error) error {
	log.Warn("failed to drain node for repair", map[string]interface{}{
		"address":   entry.Address,
		log.FnError: err,
	})
	entry.Status = cke.RepairStatusProcessing
	entry.StepStatus = cke.RepairStepStatusWaiting
	entry.LastTransitionTime = time.Now().Truncate(time.Second).UTC()
	entry.DrainBackOffCount++
	entry.DrainBackOffExpire = entry.LastTransitionTime.Add(time.Second * time.Duration(drainBackOffBaseSeconds+rand.Int63n(int64(drainBackOffBaseSeconds*entry.DrainBackOffCount))))
	return inf.Storage().UpdateRepairsEntry(ctx, entry)
}
