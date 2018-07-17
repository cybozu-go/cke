package cke

import "testing"

func testConstraintsCheck(t *testing.T) {
	nodes := []*Node{
		{ControlPlane: true},
		{ControlPlane: true},
		{ControlPlane: false},
		{ControlPlane: false},
	}

	tests := []struct {
		name        string
		constraints Constraints
		cluster     Cluster
		wantErr     bool
	}{
		{
			name:        "valid case",
			constraints: Constraints{ControlPlaneCount: 2, MaximumWorkers: 2, MinimumWorkers: 1},
			cluster:     Cluster{Nodes: nodes[:]},
			wantErr:     false,
		},
		{
			name:        "maximum 0 means unlimited",
			constraints: Constraints{ControlPlaneCount: 2, MaximumWorkers: 0, MinimumWorkers: 1},
			cluster:     Cluster{Nodes: nodes[:]},
			wantErr:     false,
		},
		{
			name:        "control plane not equal",
			constraints: Constraints{ControlPlaneCount: 1, MaximumWorkers: 2, MinimumWorkers: 1},
			cluster:     Cluster{Nodes: nodes[:]},
			wantErr:     true,
		},
		{
			name:        "too many workers",
			constraints: Constraints{ControlPlaneCount: 2, MaximumWorkers: 1, MinimumWorkers: 1},
			cluster:     Cluster{Nodes: nodes[:]},
			wantErr:     true,
		},
		{
			name:        "too few workers",
			constraints: Constraints{ControlPlaneCount: 2, MaximumWorkers: 6, MinimumWorkers: 2},
			cluster:     Cluster{Nodes: nodes[:2]},
			wantErr:     true,
		},
	}
	for _, tt := range tests {
		c := tt.constraints
		t.Run(tt.name, func(t *testing.T) {
			if err := c.Check(&tt.cluster); (err != nil) != tt.wantErr {
				t.Errorf("Constraints.Check() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConstraints(t *testing.T) {
	t.Run("Check", testConstraintsCheck)
}
