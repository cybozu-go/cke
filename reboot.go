package cke

// RebootStatus is status of reboot operation
type RebootStatus string

// Reboot statuses
const (
	RebootStatusQueued    = RebootStatus("queued")
	RebootStatusRebooting = RebootStatus("rebooting")
	RebootStatusCancelled = RebootStatus("cancelled")
)

// RebootQueueEntry represents a queue entry of reboot operation
type RebootQueueEntry struct {
	Index  int64        `json:"index,string"`
	Nodes  []string     `json:"nodes"`
	Status RebootStatus `json:"status"`
}

// NewRebootQueueEntry creates new `RebootQueueEntry`.
// `Index` will be supplied in registration.
func NewRebootQueueEntry(nodes []string) *RebootQueueEntry {
	return &RebootQueueEntry{
		Nodes:  nodes,
		Status: RebootStatusQueued,
	}
}
