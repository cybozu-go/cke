package cke

import (
	"context"
)

// Commander is a single step to proceed an operation
type Commander interface {
	// Run executes the command
	Run(ctx context.Context) error
	// Command returns the command information
	Command() Command
}

// Operator is an interface for operations
type Operator interface {
	// Name returns the operation name
	Name() string
	// NextCommand returns the next command or nil if completed
	NextCommand() Commander
	// NewRecord returns a new record for this operation
	NewRecord() *Record
	// Cleanup clean up garbage of previous failed operations, if any
	Cleanup(ctx context.Context) error
}
