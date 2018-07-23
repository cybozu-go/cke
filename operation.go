package cke

// Operator is the interface for operations
type Operator interface {
	// Name returns the operation name.
	Name() string
	// NextCommand returns the next command or nil if completed.
	NextCommand() Commander
}
