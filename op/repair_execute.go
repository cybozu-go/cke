package op

import (
	"context"
	"strings"
	"time"

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/log"
	"github.com/cybozu-go/well"
)

type repairExecuteOp struct {
	finished bool

	entry *cke.RepairQueueEntry
	step  *cke.RepairStep
}

func RepairExecuteOp(entry *cke.RepairQueueEntry, step *cke.RepairStep) cke.Operator {
	return &repairExecuteOp{
		entry: entry,
		step:  step,
	}
}

func (o *repairExecuteOp) Name() string {
	return "repair-execute"
}

func (o *repairExecuteOp) NextCommand() cke.Commander {
	if o.finished {
		return nil
	}
	o.finished = true

	return repairExecuteCommand{
		entry:          o.entry,
		command:        o.step.RepairCommand,
		timeoutSeconds: o.step.CommandTimeoutSeconds,
		retries:        o.step.CommandRetries,
		interval:       o.step.CommandInterval,
	}
}

func (o *repairExecuteOp) Targets() []string {
	return []string{o.entry.Address}
}

type repairExecuteCommand struct {
	entry          *cke.RepairQueueEntry
	command        []string
	timeoutSeconds *int
	retries        *int
	interval       *int
}

func (c repairExecuteCommand) Run(ctx context.Context, inf cke.Infrastructure, _ string) error {
	c.entry.Status = cke.RepairStatusProcessing
	c.entry.StepStatus = cke.RepairStepStatusWatching
	c.entry.LastTransitionTime = time.Now().Truncate(time.Second).UTC()
	err := inf.Storage().UpdateRepairsEntry(ctx, c.entry)
	if err != nil {
		return err
	}

	attempts := 1
	if c.retries != nil {
		attempts = *c.retries + 1
	}
RETRY:
	for i := 0; i < attempts; i++ {
		err := func() error {
			ctx := ctx
			timeout := cke.DefaultRepairCommandTimeoutSeconds
			if c.timeoutSeconds != nil {
				timeout = *c.timeoutSeconds
			}
			if timeout != 0 {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, time.Second*time.Duration(timeout))
				defer cancel()
			}

			args := append(c.command[1:], c.entry.Address)
			command := well.CommandContext(ctx, c.command[0], args...)
			return command.Run()
		}()
		if err == nil {
			return nil
		}

		log.Warn("failed on executing repair command", map[string]interface{}{
			log.FnError: err,
			"address":   c.entry.Address,
			"command":   strings.Join(c.command, " "),
			"attempts":  i,
		})
		if c.interval != nil && *c.interval != 0 {
			select {
			case <-time.After(time.Second * time.Duration(*c.interval)):
			case <-ctx.Done():
				break RETRY
			}
		}
	}

	// The failure of a repair command should not be considered as a serious error of CKE.
	log.Warn("given up repairing machine", map[string]interface{}{
		"address": c.entry.Address,
		"command": strings.Join(c.command, " "),
	})
	return repairFinish(ctx, inf, c.entry, false)
}

func (c repairExecuteCommand) Command() cke.Command {
	return cke.Command{
		Name:   "repairExecuteCommand",
		Target: c.entry.Address,
	}
}
