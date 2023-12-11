package cke

import (
	"errors"
	"slices"
	"time"
)

// RepairStatus is the status of a repair operation
type RepairStatus string

const (
	RepairStatusQueued     = RepairStatus("queued")
	RepairStatusProcessing = RepairStatus("processing")
	RepairStatusSucceeded  = RepairStatus("succeeded")
	RepairStatusFailed     = RepairStatus("failed")
)

var repairStatuses = []RepairStatus{RepairStatusQueued, RepairStatusProcessing, RepairStatusSucceeded, RepairStatusFailed}

// RepairStepStatus is the status of the current step in a repair operation
type RepairStepStatus string

const (
	RepairStepStatusWaiting  = RepairStepStatus("waiting")
	RepairStepStatusDraining = RepairStepStatus("draining")
	RepairStepStatusWatching = RepairStepStatus("watching")
)

// RepairQueueEntry represents a queue entry of a repair operation
type RepairQueueEntry struct {
	Index              int64            `json:"index,string"`
	Address            string           `json:"address"`
	Nodename           string           `json:"nodename"`
	MachineType        string           `json:"machine_type"`
	Operation          string           `json:"operation"`
	Status             RepairStatus     `json:"status"`
	Step               int              `json:"step"`
	StepStatus         RepairStepStatus `json:"step_status"`
	Deleted            bool             `json:"deleted"`
	LastTransitionTime time.Time        `json:"last_transition_time,omitempty"`
	DrainBackOffCount  int              `json:"drain_backoff_count,omitempty"`
	DrainBackOffExpire time.Time        `json:"drain_backoff_expire,omitempty"`
}

var (
	ErrRepairProcedureNotFound = errors.New("repair procedure not found for repair queue entry")
	ErrRepairOperationNotFound = errors.New("repair operation not found for repair queue entry")
	ErrRepairStepOutOfRange    = errors.New("repair step of repair queue entry is out of range")
)

func NewRepairQueueEntry(operation, machineType, address string) *RepairQueueEntry {
	return &RepairQueueEntry{
		Operation:   operation,
		MachineType: machineType,
		Address:     address,
		Status:      RepairStatusQueued,
		StepStatus:  RepairStepStatusWaiting,
	}
}

func (entry *RepairQueueEntry) FillNodename(cluster *Cluster) {
	for _, node := range cluster.Nodes {
		if node.Address == entry.Address {
			entry.Nodename = node.Nodename()
			return
		}
	}
	entry.Nodename = ""
}

func (entry *RepairQueueEntry) IsInCluster() bool {
	return entry.Nodename != ""
}

func (entry *RepairQueueEntry) HasFinished() bool {
	return entry.Status == RepairStatusSucceeded || entry.Status == RepairStatusFailed
}

func (entry *RepairQueueEntry) getMatchingRepairProcedure(cluster *Cluster) (*RepairProcedure, error) {
	for i, proc := range cluster.Repair.RepairProcedures {
		if slices.Contains(proc.MachineTypes, entry.MachineType) {
			return &cluster.Repair.RepairProcedures[i], nil
		}
	}
	return nil, ErrRepairProcedureNotFound
}

func (entry *RepairQueueEntry) GetMatchingRepairOperation(cluster *Cluster) (*RepairOperation, error) {
	proc, err := entry.getMatchingRepairProcedure(cluster)
	if err != nil {
		return nil, err
	}
	for i, op := range proc.RepairOperations {
		if op.Operation == entry.Operation {
			return &proc.RepairOperations[i], nil
		}
	}
	return nil, ErrRepairOperationNotFound
}

func (entry *RepairQueueEntry) GetCurrentRepairStep(cluster *Cluster) (*RepairStep, error) {
	op, err := entry.GetMatchingRepairOperation(cluster)
	if err != nil {
		return nil, err
	}
	if entry.Step >= len(op.RepairSteps) {
		return nil, ErrRepairStepOutOfRange
	}
	return &op.RepairSteps[entry.Step], nil
}

func CountRepairQueueEntries(entries []*RepairQueueEntry) map[string]int {
	ret := make(map[string]int)
	for _, status := range repairStatuses {
		// initialize explicitly to provide list of possible statuses
		ret[string(status)] = 0
	}

	for _, entry := range entries {
		ret[string(entry.Status)]++
	}

	return ret
}

func BuildMachineRepairStatus(nodes []*Node, entries []*RepairQueueEntry) map[string]map[string]bool {
	ret := make(map[string]map[string]bool)

	// (keys of ret) == union of (addresses of nodes) and (addresses of entries)
	for _, node := range nodes {
		ret[node.Address] = make(map[string]bool)
	}
	for _, entry := range entries {
		if _, ok := ret[entry.Address]; ok {
			continue
		}
		ret[entry.Address] = make(map[string]bool)
	}

	for address := range ret {
		for _, status := range repairStatuses {
			// initialize explicitly to provide list of possible statuses
			ret[address][string(status)] = false
		}
	}

	for _, entry := range entries {
		ret[entry.Address][string(entry.Status)] = true
	}

	return ret

}
