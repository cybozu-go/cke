package op

import (
	"context"

	"github.com/cybozu-go/cke"
)

type nopOp struct{}

func NopOp() cke.Operator {
	return &nopOp{}
}

func (o *nopOp) Name() string {
	return "nop"
}

func (o *nopOp) NextCommand() cke.Commander {
	return nil
}

func (o *nopOp) Targets() []string {
	return nil
}

func (o *nopOp) Run(ctx context.Context, inf cke.Infrastructure, _ string) error {
	// This function is never executed
	return nil
}

func (o *nopOp) Command() cke.Command {
	return cke.Command{
		Name:   "nop",
		Target: "",
	}
}
