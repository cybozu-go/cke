package server

import (
	"context"
)

// Integrator defines interface to integrate external addon into CKE server.
type Integrator interface {
	// StartWatch starts watching etcd until the context is canceled.
	//
	// It should send an empty struct to the channel when some event occurs.
	// To avoid blocking, use select when sending.
	//
	// If the integrator does not implement StartWatch, simply return nil.
	StartWatch(context.Context, chan<- struct{}) error

	// Do does something for CKE.  leaderKey is an etcd object key that
	// exists as long as the current process is the leader.
	Do(ctx context.Context, leaderKey string) error
}
