package cke

import "errors"

// Constraints is a set of conditions that a cluster must satisfy
type Constraints struct {
	ControlPlaneCount        int `json:"control-plane-count"`
	MinimumWorkersRate       int `json:"minimum-workers-rate"`
	RebootMaximumUnreachable int `json:"maximum-unreachable-nodes-for-reboot"`
	MaximumRepairs           int `json:"maximum-repair-queue-entries"`
	RepairRebootingSeconds   int `json:"wait-seconds-to-repair-rebooting"`
}

// Check checks the cluster satisfies the constraints
func (c *Constraints) Check(cluster *Cluster) error {
	cpCount := 0

	for _, n := range cluster.Nodes {
		if n.ControlPlane {
			cpCount++
		}
	}

	if cpCount != c.ControlPlaneCount {
		return errors.New("number of control planes is not equal to the constraint")
	}

	return nil
}

// DefaultConstraints returns the default constraints
func DefaultConstraints() *Constraints {
	return &Constraints{
		ControlPlaneCount:        1,
		MinimumWorkersRate:       80,
		RebootMaximumUnreachable: 0,
		MaximumRepairs:           0,
		RepairRebootingSeconds:   0,
	}
}
