package server

import "time"

// Config is the configuration for cke-server.
type Config struct {
	// Interval is the interval of the main loop.
	Interval time.Duration
	// CertsGCInterval is the interval of the certificate garbage collection.
	CertsGCInterval time.Duration
	// MaxConcurrentUpdates is the maximum number of concurrent updates.
	MaxConcurrentUpdates int
	// DrainRetryTimes is the number of retries for drain.
	DrainRetryTimes int
	// DrainRetryInterval is the interval of retries for drain.
	DrainRetryInterval time.Duration
}
