package mock

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// THIS CODE IS A STARTING POINT ONLY. IT WILL NOT BE UPDATED WITH SCHEMA CHANGES.

var machines = map[string]*Machine{
	"m1": {
		Spec: &MachineSpec{
			Serial: "m1",
			Labels: []*Label{
				{"label1", "value1"},
				{"label2", "value2"},
			},
			Rack:         1,
			IndexInRack:  3,
			Role:         "boot",
			Ipv4:         []string{"10.0.1.3"},
			RegisterDate: time.Date(2018, 12, 2, 0, 0, 0, 0, time.UTC).Format(time.RFC3339),
			RetireDate:   time.Date(2019, 1, 2, 0, 0, 0, 0, time.UTC).Format(time.RFC3339),
			Bmc:          &Bmc{"iDRAC", "172.168.1.3"},
		},
		Status: &MachineStatus{
			State:     MachineStateHealthy,
			Timestamp: time.Date(2018, 12, 2, 0, 0, 0, 0, time.UTC).Format(time.RFC3339),
			Duration:  100,
		},
	},
	"m2": {
		Spec: &MachineSpec{
			Serial: "m2",
			Labels: []*Label{
				{"label2", "value2"},
			},
			Rack:         0,
			IndexInRack:  4,
			Role:         "worker",
			Ipv4:         []string{"10.0.0.4"},
			RegisterDate: time.Date(2018, 12, 2, 0, 0, 0, 0, time.UTC).Format(time.RFC3339),
			RetireDate:   time.Date(2019, 12, 2, 0, 0, 0, 0, time.UTC).Format(time.RFC3339),
			Bmc:          &Bmc{"iDRAC", "172.168.0.4"},
		},
		Status: &MachineStatus{
			State:     MachineStateRetired,
			Timestamp: time.Date(2018, 12, 2, 0, 0, 0, 0, time.UTC).Format(time.RFC3339),
			Duration:  100,
		},
	},
	"m3": {
		Spec: &MachineSpec{
			Serial: "m3",
			Labels: []*Label{
				{"label3", "value3"},
			},
			RegisterDate: time.Date(2018, 12, 2, 0, 0, 0, 0, time.UTC).Format(time.RFC3339),
			RetireDate:   time.Date(2019, 1, 2, 0, 0, 0, 0, time.UTC).Format(time.RFC3339),
		},
		Status: &MachineStatus{
			State:     MachineStateUninitialized,
			Timestamp: time.Date(2018, 12, 2, 0, 0, 0, 0, time.UTC).Format(time.RFC3339),
			Duration:  100,
		},
	},
}

type mockResolver struct{}

func (r mockResolver) Query() QueryResolver {
	return queryResolver{}
}

type queryResolver struct{}

func (r queryResolver) Machine(ctx context.Context, serial string) (*Machine, error) {
	if m, ok := machines[serial]; ok {
		return m, nil
	}
	return nil, errors.New("not found")
}
func (r queryResolver) SearchMachines(ctx context.Context, having *MachineParams, notHaving *MachineParams) ([]*Machine, error) {
	if having == nil {
		if notHaving == nil {
			return []*Machine{machines["m1"], machines["m2"], machines["m3"]}, nil
		}
		return []*Machine{machines["m1"], machines["m3"]}, nil
	}

	if len(having.Labels) == 1 {
		if having.Labels[0].Name != "foo" {
			return nil, errors.New("wrong label name: " + having.Labels[0].Name)
		}
		if having.Labels[0].Value != "bar" {
			return nil, errors.New("wrong label value: " + having.Labels[0].Value)
		}
		return []*Machine{machines["m3"]}, nil
	}

	if len(having.Racks) == 1 {
		if having.Racks[0] != 1 {
			return nil, fmt.Errorf("wrong rack number: %d", having.Racks[0])
		}
		return []*Machine{machines["m1"]}, nil
	}

	if len(having.Roles) == 1 {
		if having.Roles[0] != "worker" {
			return nil, errors.New("wrong role: " + having.Roles[0])
		}
		return []*Machine{machines["m2"]}, nil
	}

	if len(having.States) == 1 {
		if string(having.States[0]) == "bad" {
			return nil, errors.New("bad state value")
		}
		if having.States[0] != MachineStateUninitialized {
			return nil, errors.New("unexpected state: " + string(having.States[0]))
		}
		return []*Machine{machines["m3"]}, nil
	}

	if having.MinDaysBeforeRetire != nil {
		if *having.MinDaysBeforeRetire != 90 {
			return nil, fmt.Errorf("unexpected days: %d", *having.MinDaysBeforeRetire)
		}
		return []*Machine{machines["m2"]}, nil
	}

	return nil, nil
}
