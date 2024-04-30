package sabakan

import (
	"testing"

	"github.com/cybozu-go/cke"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestRepairer(t *testing.T) {
	constraints := &cke.Constraints{
		MaximumRepairs: 3,
	}

	machines := []Machine{
		{Spec: MachineSpec{Serial: "0000"}, Status: MachineStatus{State: StateUnhealthy}},
		{Spec: MachineSpec{Serial: "1111", IPv4: []string{"1.1.1.1"}, BMC: BMC{Type: "type1"}}, Status: MachineStatus{State: StateUnhealthy}},
		{Spec: MachineSpec{Serial: "2222", IPv4: []string{"2.2.2.2"}, BMC: BMC{Type: "type2"}}, Status: MachineStatus{State: StateUnhealthy}},
		{Spec: MachineSpec{Serial: "3333", IPv4: []string{"3.3.3.3"}, BMC: BMC{Type: "type3"}}, Status: MachineStatus{State: StateUnreachable}},
		{Spec: MachineSpec{Serial: "4444", IPv4: []string{"4.4.4.4"}, BMC: BMC{Type: "type4"}}, Status: MachineStatus{State: StateUnreachable}},
	}

	entries := []*cke.RepairQueueEntry{
		nil,
		cke.NewRepairQueueEntry("unhealthy", "type1", "1.1.1.1"),
		cke.NewRepairQueueEntry("unhealthy", "type2", "2.2.2.2"),
		cke.NewRepairQueueEntry("unreachable", "type3", "3.3.3.3"),
		cke.NewRepairQueueEntry("unreachable", "type4", "4.4.4.4"),
	}

	tests := []struct {
		name            string
		failedMachines  []Machine
		queuedEntries   []*cke.RepairQueueEntry
		expectedEntries []*cke.RepairQueueEntry
	}{
		{
			name:            "NoFailedMachine",
			failedMachines:  []Machine{},
			queuedEntries:   []*cke.RepairQueueEntry{entries[2]},
			expectedEntries: []*cke.RepairQueueEntry{},
		},
		{
			name:            "OneFailedMachine",
			failedMachines:  []Machine{machines[1]},
			queuedEntries:   []*cke.RepairQueueEntry{entries[2]},
			expectedEntries: []*cke.RepairQueueEntry{entries[1]},
		},
		{
			name:            "IgnoreNoIPAddress",
			failedMachines:  []Machine{machines[0], machines[1]},
			queuedEntries:   []*cke.RepairQueueEntry{entries[2]},
			expectedEntries: []*cke.RepairQueueEntry{entries[1]},
		},
		{
			name:            "IgnoreRecentlyRepaired",
			failedMachines:  []Machine{machines[1], machines[2], machines[3]},
			queuedEntries:   []*cke.RepairQueueEntry{entries[2]},
			expectedEntries: []*cke.RepairQueueEntry{entries[1], entries[3]},
		},
		{
			name:            "IgnoreRecentlyRepairedWithDifferentOperation",
			failedMachines:  []Machine{machines[1], machines[2], machines[3]},
			queuedEntries:   []*cke.RepairQueueEntry{cke.NewRepairQueueEntry("unreachable", "type2", "2.2.2.2")},
			expectedEntries: []*cke.RepairQueueEntry{entries[1], entries[3]},
		},
		{
			name:            "IgnoreTooManyFailedMachines",
			failedMachines:  []Machine{machines[1], machines[2], machines[3]},
			queuedEntries:   []*cke.RepairQueueEntry{entries[2], entries[4]},
			expectedEntries: []*cke.RepairQueueEntry{},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			entries := Repairer(tt.failedMachines, tt.queuedEntries, constraints)
			if !cmp.Equal(entries, tt.expectedEntries, cmpopts.EquateEmpty()) {
				t.Errorf("!cmp.Equal(entries, tt.newEntries), actual: %v, expected: %v", entries, tt.expectedEntries)
			}
		})
	}
}
