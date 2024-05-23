package sabakan

import (
	"testing"
	"time"

	"github.com/cybozu-go/cke"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestRepairer(t *testing.T) {
	constraints := &cke.Constraints{
		MaximumRepairs:         3,
		RepairRebootingSeconds: 300,
	}

	machines := []Machine{
		{Spec: MachineSpec{Serial: "0000"}, Status: MachineStatus{State: StateUnhealthy}},
		{Spec: MachineSpec{Serial: "1111", IPv4: []string{"1.1.1.1"}, BMC: BMC{Type: "type1"}}, Status: MachineStatus{State: StateUnreachable}},
		{Spec: MachineSpec{Serial: "2222", IPv4: []string{"2.2.2.2"}, BMC: BMC{Type: "type2"}}, Status: MachineStatus{State: StateUnhealthy}},
		{Spec: MachineSpec{Serial: "3333", IPv4: []string{"3.3.3.3"}, BMC: BMC{Type: "type3"}}, Status: MachineStatus{State: StateUnreachable}},
		{Spec: MachineSpec{Serial: "4444", IPv4: []string{"4.4.4.4"}, BMC: BMC{Type: "type4"}}, Status: MachineStatus{State: StateUnreachable}},
	}

	entries := []*cke.RepairQueueEntry{
		nil,
		cke.NewRepairQueueEntry("unreachable", "type1", "1.1.1.1"),
		cke.NewRepairQueueEntry("unhealthy", "type2", "2.2.2.2"),
		cke.NewRepairQueueEntry("unreachable", "type3", "3.3.3.3"),
		cke.NewRepairQueueEntry("unreachable", "type4", "4.4.4.4"),
	}

	now := time.Now().UTC()
	recent := now.Add(-30 * time.Second)
	stale := now.Add(-3000 * time.Second)
	rebootEntries := []*cke.RebootQueueEntry{
		nil,
		{Node: "1.1.1.1", Status: cke.RebootStatusRebooting, LastTransitionTime: recent},
		{Node: "2.2.2.2", Status: cke.RebootStatusRebooting, LastTransitionTime: recent},
		{Node: "3.3.3.3", Status: cke.RebootStatusRebooting, LastTransitionTime: stale},
		{Node: "4.4.4.4", Status: cke.RebootStatusDraining, LastTransitionTime: recent},
	}

	tests := []struct {
		name            string
		failedMachines  []Machine
		queuedEntries   []*cke.RepairQueueEntry
		rebootEntries   []*cke.RebootQueueEntry
		expectedEntries []*cke.RepairQueueEntry
	}{
		{
			name:            "NoFailedMachine",
			failedMachines:  []Machine{},
			queuedEntries:   []*cke.RepairQueueEntry{entries[2]},
			rebootEntries:   nil,
			expectedEntries: []*cke.RepairQueueEntry{},
		},
		{
			name:            "OneFailedMachine",
			failedMachines:  []Machine{machines[1]},
			queuedEntries:   []*cke.RepairQueueEntry{entries[2]},
			rebootEntries:   nil,
			expectedEntries: []*cke.RepairQueueEntry{entries[1]},
		},
		{
			name:            "IgnoreNoIPAddress",
			failedMachines:  []Machine{machines[0], machines[1]},
			queuedEntries:   []*cke.RepairQueueEntry{entries[2]},
			rebootEntries:   nil,
			expectedEntries: []*cke.RepairQueueEntry{entries[1]},
		},
		{
			name:            "IgnoreRecentlyRepaired",
			failedMachines:  []Machine{machines[1], machines[2], machines[3]},
			queuedEntries:   []*cke.RepairQueueEntry{entries[2]},
			rebootEntries:   nil,
			expectedEntries: []*cke.RepairQueueEntry{entries[1], entries[3]},
		},
		{
			name:            "IgnoreRecentlyRepairedWithDifferentOperation",
			failedMachines:  []Machine{machines[1], machines[2], machines[3]},
			queuedEntries:   []*cke.RepairQueueEntry{cke.NewRepairQueueEntry("unreachable", "type2", "2.2.2.2")},
			rebootEntries:   nil,
			expectedEntries: []*cke.RepairQueueEntry{entries[1], entries[3]},
		},
		{
			name:            "IgnoreTooManyFailedMachines",
			failedMachines:  []Machine{machines[1], machines[2], machines[3]},
			queuedEntries:   []*cke.RepairQueueEntry{entries[2], entries[4]},
			rebootEntries:   nil,
			expectedEntries: []*cke.RepairQueueEntry{},
		},
		{
			name:            "IgnoreRebootingUnreachableMachine",
			failedMachines:  []Machine{machines[1]},
			queuedEntries:   []*cke.RepairQueueEntry{},
			rebootEntries:   []*cke.RebootQueueEntry{rebootEntries[1]},
			expectedEntries: []*cke.RepairQueueEntry{},
		},
		{
			name:            "RebootingButUnhealthy",
			failedMachines:  []Machine{machines[2]},
			queuedEntries:   []*cke.RepairQueueEntry{},
			rebootEntries:   []*cke.RebootQueueEntry{rebootEntries[2]},
			expectedEntries: []*cke.RepairQueueEntry{entries[2]},
		},
		{
			name:            "RebootingButStale",
			failedMachines:  []Machine{machines[3]},
			queuedEntries:   []*cke.RepairQueueEntry{},
			rebootEntries:   []*cke.RebootQueueEntry{rebootEntries[3]},
			expectedEntries: []*cke.RepairQueueEntry{entries[3]},
		},
		{
			name:            "QueuedButNotRebooting",
			failedMachines:  []Machine{machines[4]},
			queuedEntries:   []*cke.RepairQueueEntry{},
			rebootEntries:   []*cke.RebootQueueEntry{rebootEntries[4]},
			expectedEntries: []*cke.RepairQueueEntry{entries[4]},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			entries := Repairer(tt.failedMachines, tt.queuedEntries, tt.rebootEntries, constraints, now)
			if !cmp.Equal(entries, tt.expectedEntries, cmpopts.EquateEmpty()) {
				t.Errorf("!cmp.Equal(entries, tt.newEntries), actual: %v, expected: %v", entries, tt.expectedEntries)
			}
		})
	}
}
