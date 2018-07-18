package cke

import (
	"context"
)

// Operator is the interface for operations
type Operator interface {
	// Name returns the operation name.
	Name() string
	// NextCommand returns the next command or nil if completed.
	NextCommand() Commander
	// Cleanup clean up garbage of previous failed operations, if any.
	Cleanup(ctx context.Context) error
}
