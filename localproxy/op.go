package localproxy

import (
	"context"

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/cke/op/common"
)

type unboundBootOp struct {
	conf []byte

	step int
}

var localNode = &cke.Node{
	Address: "localhost",
}

var ckeNodes = []*cke.Node{localNode}

var _ cke.Operator = &unboundBootOp{}

const unboundContainerName = "cke-unbound"

// Name returns the operation name.
func (o *unboundBootOp) Name() string {
	return "unbound-boot-op"
}

// NextCommand returns the next command or nil if completed.
func (o *unboundBootOp) NextCommand() cke.Commander {
	defer func() { o.step++ }()

	switch o.step {
	case 0:
		return common.ImagePullCommand(ckeNodes, cke.UnboundImage)
	case 1:
		files := common.NewFilesBuilder(ckeNodes)
		files.AddFile(context.Background(), "/etc/unbound/unbound.conf", func(context.Context, *cke.Node) ([]byte, error) {
			return o.conf, nil
		})
		return files
	case 2:
		return common.RunContainerCommand(ckeNodes, unboundContainerName, cke.UnboundImage,
			common.WithOpts([]string{"--cap-add=NET_BIND_SERVICE"}),
			common.WithParams(cke.ServiceParams{
				ExtraArguments: []string{"-c", "/etc/unbound/unbound.conf"},
				ExtraBinds: []cke.Mount{
					{
						Source:      "/etc/unbound",
						Destination: "/etc/unbound",
						ReadOnly:    true,
					},
				},
			}),
		)
	default:
		return nil
	}
}

// Targets returns the ip which will be affected by the operation
func (o *unboundBootOp) Targets() []string {
	return []string{"localhost"}
}

type unboundRestartOp struct {
	conf []byte

	step int
}

var _ cke.Operator = &unboundRestartOp{}

// Name returns the operation name.
func (o *unboundRestartOp) Name() string {
	return "unbound-restart-op"
}

// NextCommand returns the next command or nil if completed.
func (o *unboundRestartOp) NextCommand() cke.Commander {
	defer func() { o.step++ }()

	switch o.step {
	case 0:
		return common.ImagePullCommand(ckeNodes, cke.UnboundImage)
	case 1:
		files := common.NewFilesBuilder(ckeNodes)
		files.AddFile(context.Background(), "/etc/unbound/unbound.conf", func(context.Context, *cke.Node) ([]byte, error) {
			return o.conf, nil
		})
		return files
	case 2:
		return common.RunContainerCommand(ckeNodes, unboundContainerName, cke.UnboundImage,
			common.WithOpts([]string{"--cap-add=NET_BIND_SERVICE"}),
			common.WithParams(cke.ServiceParams{
				ExtraArguments: []string{"-c", "/etc/unbound/unbound.conf"},
				ExtraBinds: []cke.Mount{
					{
						Source:      "/etc/unbound",
						Destination: "/etc/unbound",
						ReadOnly:    true,
					},
				},
			}),
			common.WithRestart(),
		)
	default:
		return nil
	}
}

// Targets returns the ip which will be affected by the operation
func (o *unboundRestartOp) Targets() []string {
	return []string{"localhost"}
}
