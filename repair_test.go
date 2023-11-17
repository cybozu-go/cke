package cke

import (
	"slices"
	"testing"
)

func TestRepairQueueEntry(t *testing.T) {
	cluster := &Cluster{
		Nodes: []*Node{
			{
				Address:  "1.1.1.1",
				Hostname: "node1",
			},
		},
		Repair: Repair{
			RepairProcedures: []RepairProcedure{
				{
					MachineTypes: []string{"type1", "type2"},
					RepairOperations: []RepairOperation{
						{
							Operation: "unreachable",
							RepairSteps: []RepairStep{
								{RepairCommand: []string{"type12", "unreachable0"}},
								{RepairCommand: []string{"type12", "unreachable1"}},
							},
							HealthCheckCommand: []string{"check12-unreachable"},
						},
						{
							Operation: "unhealthy",
							RepairSteps: []RepairStep{
								{RepairCommand: []string{"type12", "unhealthy0"}},
							},
							HealthCheckCommand: []string{"check12-unhealthy"},
						},
					},
				},
				{
					MachineTypes: []string{"type3"},
					RepairOperations: []RepairOperation{
						{
							RepairSteps: []RepairStep{
								{RepairCommand: []string{"type3", "unreachable0"}},
							},
							HealthCheckCommand: []string{"check3"},
						},
					},
				},
			},
		},
	}

	// for in-cluster machine
	entry := NewRepairQueueEntry("unreachable", "type2", "1.1.1.1")
	entry.FillNodename(cluster)
	if entry.Nodename != "node1" {
		t.Error("FillNodename() failed to fill Nodename:", entry.Nodename)
	}
	if !entry.IsInCluster() {
		t.Error("IsInCluster() returned false incorrectly")
	}

	// for out-of-cluster machine
	// GetCorrespondingNode should fail for bad address
	entry = NewRepairQueueEntry("unreachable", "type2", "2.2.2.2")
	entry.FillNodename(cluster)
	if entry.Nodename != "" {
		t.Error("FillNodename() filled wrong Nodename:", entry.Nodename)
	}
	if entry.IsInCluster() {
		t.Error("IsInCluster() returned true incorrectly")
	}

	// HaveFinished should return true iff entry has succeeded or failed
	entry = NewRepairQueueEntry("unreachable", "type2", "1.1.1.1")
	for _, testCase := range []struct {
		status   RepairStatus
		finished bool
	}{
		{RepairStatusQueued, false},
		{RepairStatusProcessing, false},
		{RepairStatusSucceeded, true},
		{RepairStatusFailed, true},
	} {
		entry.Status = testCase.status
		if entry.HasFinished() != testCase.finished {
			t.Errorf("HaveFinished() returned %v incorrectly for %q", testCase.finished, testCase.status)
		}
	}

	// GetMatchingRepairOperation should succeed
	entry = NewRepairQueueEntry("unreachable", "type2", "1.1.1.1")
	op, err := entry.GetMatchingRepairOperation(cluster)
	if err != nil {
		t.Fatal("GetMatchingRepairOperation() failed:", err)
	}
	if !slices.Equal(op.HealthCheckCommand, []string{"check12-unreachable"}) {
		t.Error("GetMatchingRepairOperation() returned wrong repair operation:", op)
	}

	// GetMatchingRepairOperation should fail for bad machine type
	entry = NewRepairQueueEntry("unreachable", "type4", "1.1.1.1")
	_, err = entry.GetMatchingRepairOperation(cluster)
	if err != ErrRepairProcedureNotFound {
		t.Error("GetMatchingRepairOperation() returned wrong error:", err)
	}

	// GetMatchingRepairOperation should fail for bad operation
	entry = NewRepairQueueEntry("noop", "type2", "1.1.1.1")
	_, err = entry.GetMatchingRepairOperation(cluster)
	if err != ErrRepairOperationNotFound {
		t.Error("GetMatchingRepairOperation() returned wrong error:", err)
	}

	// GetCurrentRepairStep should succeed
	entry = NewRepairQueueEntry("unreachable", "type2", "1.1.1.1")
	entry.Status = RepairStatusProcessing
	entry.Step = 1
	entry.StepStatus = RepairStepStatusWatching
	step, err := entry.GetCurrentRepairStep(cluster)
	if err != nil {
		t.Fatal("GetCurrentRepairStep() failed:", err)
	}
	if !slices.Equal(step.RepairCommand, []string{"type12", "unreachable1"}) {
		t.Error("GetCurrentRepairStep() returned wrong repair step:", step)
	}

	// GetCurrentRepairStep should fail for end of steps
	entry.Step++
	_, err = entry.GetCurrentRepairStep(cluster)
	if err != ErrRepairStepOutOfRange {
		t.Error("GetCurrentRepairStep() returned wrong error:", err)
	}

	// GetCurrentRepairStep should fail for bad machine type
	entry = NewRepairQueueEntry("unreachable", "type4", "1.1.1.1")
	entry.Status = RepairStatusProcessing
	entry.Step = 1
	entry.StepStatus = RepairStepStatusWatching
	_, err = entry.GetCurrentRepairStep(cluster)
	if err != ErrRepairProcedureNotFound {
		t.Error("GetCurrentRepairStep() returned wrong error:", err)
	}

	// GetCurrentRepairStep should fail for bad operation
	entry = NewRepairQueueEntry("noop", "type2", "1.1.1.1")
	entry.Status = RepairStatusProcessing
	entry.Step = 1
	entry.StepStatus = RepairStepStatusWatching
	_, err = entry.GetCurrentRepairStep(cluster)
	if err != ErrRepairOperationNotFound {
		t.Error("GetCurrentRepairStep() returned wrong error:", err)
	}
}
