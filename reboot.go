package cke

import (
	"time"
)

// RebootStatus is status of reboot operation
type RebootStatus string

// Reboot statuses
const (
	RebootStatusQueued    = RebootStatus("queued")
	RebootStatusDraining  = RebootStatus("draining")
	RebootStatusRebooting = RebootStatus("rebooting")
	RebootStatusCancelled = RebootStatus("cancelled")
)

// RebootQueueEntry represents a queue entry of reboot operation
type RebootQueueEntry struct {
	Index              int64        `json:"index,string"`
	Node               string       `json:"node"`
	Status             RebootStatus `json:"status"`
	LastTransitionTime time.Time    `json:"last_transition_time,omitempty"`
	DrainBackOffCount  int          `json:"drain_backoff_count,omitempty"`
	DrainBackOffExpire time.Time    `json:"drain_backoff_expire,omitempty"`
}

// NewRebootQueueEntry creates new `RebootQueueEntry`.
// `Index` will be supplied in registration.
func NewRebootQueueEntry(node string) *RebootQueueEntry {
	return &RebootQueueEntry{
		Node:   node,
		Status: RebootStatusQueued,
	}
}

// ClusterMember returns whether the node in this entry is a cluster member.
func (entry *RebootQueueEntry) ClusterMember(c *Cluster) bool {
	for _, clusterNode := range c.Nodes {
		if entry.Node == clusterNode.Address {
			return true
		}
	}
	return false
}

func DedupRebootQueueEntries(entries []*RebootQueueEntry) []*RebootQueueEntry {
	var ret []*RebootQueueEntry
	nodes := map[string]bool{}
	for _, entry := range entries {
		if !nodes[entry.Node] {
			nodes[entry.Node] = true
			ret = append(ret, entry)
		}
	}
	return ret
}

func CountRebootQueueEntries(entries []*RebootQueueEntry) map[string]int {
	ret := map[string]int{}
	ret[string(RebootStatusQueued)] = 0
	ret[string(RebootStatusDraining)] = 0
	ret[string(RebootStatusRebooting)] = 0
	ret[string(RebootStatusCancelled)] = 0

	for _, entry := range entries {
		ret[string(entry.Status)]++
	}

	return ret
}
