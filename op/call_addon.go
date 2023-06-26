package op

import (
	"context"

	"github.com/cybozu-go/cke"
)

type callAddOnOp struct{}

func CallAddOnOp() cke.Operator {
	return &callAddOnOp{}
}

func (o *callAddOnOp) Name() string {
	return "callAddOn"
}

func (o *callAddOnOp) NextCommand() cke.Commander {
	return nil
}

func (o *callAddOnOp) Targets() []string {
	return nil
}

func (o *callAddOnOp) Run(ctx context.Context, inf cke.Infrastructure, _ string) error {
	// This function is never executed
	return nil
}

func (o *callAddOnOp) Command() cke.Command {
	return cke.Command{
		Name:   "callAddOn",
		Target: "",
	}
}
