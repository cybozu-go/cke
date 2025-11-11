package common

import (
	"context"
	"time"

	"github.com/cybozu-go/cke"
)

type waitCommand struct {
	duration time.Duration
}

// WaitCommand returns a Commander to wait for the specified time.
func WaitCommand(duration time.Duration) cke.Commander {
	return waitCommand{duration: duration}
}

func (c waitCommand) Run(ctx context.Context, _ cke.Infrastructure, _ string) error {
	select {
	case <-ctx.Done():
	case <-time.After(c.duration):
	}
	return nil
}

func (c waitCommand) Command() cke.Command {
	return cke.Command{
		Name: "wait",
	}
}
