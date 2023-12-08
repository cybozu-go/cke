package cke

import (
	"time"
)

// RebootQueueState is state of reboot queue
type RebootQueueState string

// RebootQueue states
const (
	// Reboot queue is enabled (should be set by ckecli)
	RebootQueueStateEnabled = RebootQueueState("enabled")
	// Reboot queue is requested to stop (should be set by ckecli)
	RebootQueueStateStopping = RebootQueueState("stopping")
	// Reboot queue is disabled (should be set by CKE)
	RebootQueueStateDisabled = RebootQueueState("disabled")
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

var rebootStatuses = []RebootStatus{RebootStatusQueued, RebootStatusDraining, RebootStatusRebooting, RebootStatusCancelled}

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
	for _, status := range rebootStatuses {
		// initialize explicitly to provide list of possible statuses
		ret[string(status)] = 0
	}

	for _, entry := range entries {
		ret[string(entry.Status)]++
	}

	return ret
}

func BuildNodeRebootStatus(nodes []*Node, entries []*RebootQueueEntry) map[string]map[string]bool {
	ret := make(map[string]map[string]bool)
	addr2name := make(map[string]string)

	for _, node := range nodes {
		name := node.Nodename()
		ret[name] = make(map[string]bool)
		for _, status := range rebootStatuses {
			// initialize explicitly to provide list of possible statuses
			ret[name][string(status)] = false
		}
		addr2name[node.Address] = name
	}

	for _, entry := range entries {
		name, ok := addr2name[entry.Node]
		if !ok {
			// removed from K8s cluster after queued
			continue
		}
		ret[name][string(entry.Status)] = true
	}

	return ret
}
