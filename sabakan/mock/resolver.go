package mock

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.

import "time"

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
			Bmc:          &Bmc{"iDRAC", ""},
		},
		Status: &MachineStatus{
			State:     MachineStateUninitialized,
			Timestamp: time.Date(2018, 12, 2, 0, 0, 0, 0, time.UTC).Format(time.RFC3339),
			Duration:  100,
		},
	},
}

type Resolver struct{}

type mockResolver struct{}

func (r mockResolver) Query() QueryResolver {
	return &queryResolver{}
}
