package cke

import "context"

// Operator is an interface for operations
type Operator interface {
	// Name returns the operation name
	Name() string
	// Run executes the operation
	Run(ctx context.Context) error
}
