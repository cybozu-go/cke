package sabakan

import (
	"sort"
	"testing"
	"time"

	"github.com/cybozu-go/cke"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	corev1 "k8s.io/api/core/v1"
)

func testMachineToNode(t *testing.T) {
	machine := &Machine{}
	machine.Spec.Serial = "test"
	machine.Spec.Labels = []struct {
		Name  string `json:"name"`
		Value string `json:"value"`
	}{{Name: "foo", Value: "foobar"}}
	machine.Spec.Rack = 0
	machine.Spec.IndexInRack = 1
	machine.Spec.Role = "worker"
	machine.Spec.IPv4 = []string{"10.0.0.1"}
	machine.Spec.RegisterDate = testPast250
	machine.Spec.RetireDate = testBaseTS
	machine.Status.State = StateUnhealthy

	node := &cke.Node{
		ControlPlane: true,
		Labels:       map[string]string{"foo": "bar"},
		Annotations:  map[string]string{"hoge": "fuga"},
		Taints:       []corev1.Taint{{Key: "foo", Effect: corev1.TaintEffectNoSchedule}},
	}
	res1, err := MachineToNode(machine, node)
	if err != nil {
		t.Fatal(err)
	}

	domain := "cke.cybozu.com"
	if res1.Annotations["hoge"] != "fuga" {
		t.Error(`res1.Annotations["hoge"] != "fuga", actual:`, res1.Annotations)
	}
	if res1.Annotations[domain+"/serial"] != machine.Spec.Serial {
		t.Error(`res1.Annotations[domain + "/serial"] != "test", actual:`, res1.Annotations)
	}
	if res1.Annotations[domain+"/register-date"] != testPast250.Format(time.RFC3339) {
		t.Error(`res1.Annotations[domain + "/register-date"] != machine.Spec.RegisterDate.Format(time.RFC3339), actual:`, res1.Annotations)
	}
	if res1.Annotations[domain+"/retire-date"] != testBaseTS.Format(time.RFC3339) {
		t.Error(`res1.Annotations[domain + "/register-date"] != machine.Spec.RegisterDate.Format(time.RFC3339), actual:`, res1.Annotations)
	}
	if res1.Labels["foo"] != "bar" {
		t.Error(`res1.Labels["foo"] != "bar", actual:`, res1.Labels)
	}
	if res1.Labels["sabakan."+domain+"/foo"] != "foobar" {
		t.Error(`res1.Labels["sabakan."+domain+"/foo"] != "foobar"`, res1.Labels)
	}
	if res1.Labels[domain+"/rack"] != "0" {
		t.Error(`res1.Labels["cke.cybozu.com/rack"] != "0", actual:`, res1.Labels)
	}
	if res1.Labels[domain+"/index-in-rack"] != "1" {
		t.Error(`res1.Labels["cke.cybozu.com/index-in-rack"] != "1", actual:`, res1.Labels)
	}
	if res1.Labels[domain+"/role"] != "worker" {
		t.Error(`res1.Labels["cke.cybozu.com/worker"] != "worker", actual:`, res1.Labels)
	}
	if !containsTaint(res1.Taints, corev1.Taint{Key: "foo", Effect: corev1.TaintEffectNoSchedule}) {
		t.Error(`res1.Taints do not have corev1.Taint{Key"foo", Effect: corev1.TaintEffectNoSchedule}, actual:`, res1.Taints)
	}
	if !containsTaint(res1.Taints, corev1.Taint{Key: domain + "/state", Value: "unhealthy", Effect: corev1.TaintEffectNoSchedule}) {
		t.Error(`res1.Taints do not have corev1.Taint{Key: "cke.cybozu.com/state", Value: "unhealthy", Effect: "NoSchedule"}, actual:`, res1.Taints)
	}

	machine.Status.State = StateRetiring
	res2, err := MachineToNode(machine, node)
	if err != nil {
		t.Fatal(err)
	}
	if !containsTaint(res2.Taints, corev1.Taint{Key: domain + "/state", Value: "retiring", Effect: corev1.TaintEffectNoExecute}) {
		t.Error(`res2.Taints do not have corev1.Taint{Key: "cke.cybozu.com/state", Value: "retiring", Effect: "NoExecute"}, actual:`, res2.Taints)
	}
	machine.Status.State = StateRetired
	res3, err := MachineToNode(machine, node)
	if err != nil {
		t.Fatal(err)
	}
	if !containsTaint(res3.Taints, corev1.Taint{Key: domain + "/state", Value: "retired", Effect: corev1.TaintEffectNoExecute}) {
		t.Error(`res3.Taints do not have corev1.Taint{Key: "cke.cybozu.com/state", Value: "retired", Effect: "NoExecute"}, actual:`, res3.Taints)
	}
}

func containsTaint(taints []corev1.Taint, target corev1.Taint) bool {
	for _, taint := range taints {
		if cmp.Equal(taint, target) {
			return true
		}
	}
	return false
}

func newTestMachineWithIP(rack int, retireDate time.Time, state State, ip, role string) Machine {
	m := Machine{}
	m.Spec.IPv4 = []string{ip}
	m.Spec.Rack = rack
	m.Spec.Role = role
	m.Spec.RetireDate = retireDate
	m.Status.State = state
	m.Status.Duration = DefaultWaitRetiringSeconds * 2
	return m
}

func testNewGenerator(t *testing.T) {
	tmpl := &cke.Cluster{
		Name: "test",
		Nodes: []*cke.Node{
			{
				ControlPlane: true,
				Labels:       map[string]string{"foo": "bar"},
				Annotations:  map[string]string{"hoge": "fuga"},
				Taints:       []corev1.Taint{{Key: "foo", Effect: corev1.TaintEffectNoSchedule}},
			},
			{
				ControlPlane: false,
				Labels: map[string]string{
					"foo":                   "aaa",
					"cke.cybozu.com/role":   "cs",
					"cke.cybozu.com/weight": "3",
				},
				Annotations: map[string]string{"hoge": "bbb"},
				Taints:      []corev1.Taint{{Key: "foo", Effect: corev1.TaintEffectNoExecute}},
			},
		},
	}
	cluster := &cke.Cluster{
		Name: "test",
		Nodes: []*cke.Node{
			{
				Address:      "10.0.0.1",
				ControlPlane: true,
				Labels: map[string]string{
					"foo":                 "bar",
					"cke.cybozu.com/role": "cs",
				},
				Annotations: map[string]string{"hoge": "fuga"},
				Taints:      []corev1.Taint{{Key: "foo", Effect: corev1.TaintEffectNoSchedule}},
			},
			{
				Address:      "10.0.0.3",
				ControlPlane: false,
				Labels: map[string]string{
					"foo":                 "aaa",
					"cke.cybozu.com/role": "ss",
				},
				Annotations: map[string]string{"hoge": "bbb"},
				Taints:      []corev1.Taint{{Key: "foo", Effect: corev1.TaintEffectNoExecute}},
			},
			{
				Address:      "10.0.0.2",
				ControlPlane: true,
				Labels: map[string]string{
					"foo":                 "bar",
					"cke.cybozu.com/role": "cs",
				},
				Annotations: map[string]string{"hoge": "fuga"},
				Taints:      []corev1.Taint{{Key: "foo", Effect: corev1.TaintEffectNoSchedule}},
			},
			{
				Address:      "10.0.0.100",
				ControlPlane: false,
				Labels:       map[string]string{"foo": "aaa"},
				Annotations:  map[string]string{"hoge": "bbb"},
				Taints:       []corev1.Taint{{Key: "foo", Effect: corev1.TaintEffectNoExecute}},
			},
		},
	}
	machines := []Machine{
		newTestMachineWithIP(0, testFuture250, StateHealthy, "10.0.0.1", "cs"),
		newTestMachineWithIP(1, testFuture250, StateRetired, "10.0.0.2", "cs"),
		newTestMachineWithIP(0, testFuture250, StateUnhealthy, "10.0.0.3", "ss"),
		newTestMachineWithIP(1, testFuture250, StateHealthy, "10.0.0.4", "ss"),
	}
	type args struct {
		current  *cke.Cluster
		template *cke.Cluster
		cstr     *cke.Constraints
		machines []Machine
	}
	tests := []struct {
		name string
		args args
		want *Generator
	}{
		{
			"NoCluster",
			args{nil, tmpl, cke.DefaultConstraints(), machines},
			&Generator{
				machineMap: map[string]*Machine{
					"10.0.0.1": &machines[0],
					"10.0.0.2": &machines[1],
					"10.0.0.3": &machines[2],
					"10.0.0.4": &machines[3],
				},
				unusedMachines: []*Machine{&machines[0], &machines[3]},
				cpTmpl:         nodeTemplate{tmpl.Nodes[0], "", 1.0},
				workerTmpls: []nodeTemplate{
					{tmpl.Nodes[1], "cs", 3.0},
				},
			},
		},
		{
			"Cluster",
			args{cluster, tmpl, cke.DefaultConstraints(), machines},
			&Generator{
				controlPlanes: []*cke.Node{
					cluster.Nodes[0],
					cluster.Nodes[2],
				},
				healthyCPs: 1,
				cpRacks:    map[int]int{0: 1, 1: 1},
				workers: []*cke.Node{
					cluster.Nodes[1],
					cluster.Nodes[3],
				},
				workersByRole:  map[string]int{"ss": 1, "": 1},
				healthyWorkers: 0,
				workerRacks:    map[int]int{0: 1},
				machineMap: map[string]*Machine{
					"10.0.0.1": &machines[0],
					"10.0.0.2": &machines[1],
					"10.0.0.3": &machines[2],
					"10.0.0.4": &machines[3],
				},
				unusedMachines: []*Machine{&machines[3]},
				cpTmpl:         nodeTemplate{tmpl.Nodes[0], "", 1.0},
				workerTmpls: []nodeTemplate{
					{tmpl.Nodes[1], "cs", 3.0},
				},
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := NewGenerator(tt.args.current, tt.args.template, tt.args.cstr, tt.args.machines, testBaseTS)
			if !cmp.Equal(got.controlPlanes, tt.want.controlPlanes, cmpopts.IgnoreUnexported(cke.Node{}), cmpopts.EquateEmpty()) {
				t.Errorf("!cmp.Equal(got.controlPlanes, tt.want.controlPlanes), actual: %v, want %v", got.controlPlanes, tt.want.controlPlanes)
			}
			if got.healthyCPs != tt.want.healthyCPs {
				t.Errorf("!cmp.Equal(got.healthyCPs, tt.want.healthyCPs), actual: %v, want %v", got.healthyCPs, tt.want.healthyCPs)
			}
			if !cmp.Equal(got.cpRacks, tt.want.cpRacks, cmpopts.EquateEmpty()) {
				t.Errorf("!cmp.Equal(got.cpRacks, tt.want.cpRacks), actual: %v, want %v", got.cpRacks, tt.want.cpRacks)
			}
			if !cmp.Equal(got.workers, tt.want.workers, cmpopts.IgnoreUnexported(cke.Node{}), cmpopts.EquateEmpty()) {
				t.Errorf("!cmp.Equal(got.workers, tt.want.workers), actual: %v, want %v", got.workers, tt.want.workers)
			}
			if !cmp.Equal(got.workersByRole, tt.want.workersByRole, cmpopts.IgnoreUnexported(cke.Node{}), cmpopts.EquateEmpty()) {
				t.Error("!cmp.Equal(got.workersByRole, tt.want.workersByRole)", cmp.Diff(got.workersByRole, tt.want.workersByRole, cmpopts.IgnoreUnexported(cke.Node{}), cmpopts.EquateEmpty()))
			}
			if got.healthyWorkers != tt.want.healthyWorkers {
				t.Errorf("!cmp.Equal(got.healthyWorkers, tt.want.healthyWorkers), actual: %v, want %v", got.healthyWorkers, tt.want.healthyWorkers)
			}
			if !cmp.Equal(got.workerRacks, tt.want.workerRacks, cmpopts.EquateEmpty()) {
				t.Errorf("!cmp.Equal(got.workerRacks, tt.want.workerRacks), actual: %v, want %v", got.workerRacks, tt.want.workerRacks)
			}
			if !cmp.Equal(got.machineMap, tt.want.machineMap, cmpopts.EquateEmpty()) {
				t.Errorf("!cmp.Equal(got.machineMap, tt.want.machineMap), actual: %v, want %v", got.machineMap, tt.want.machineMap)
			}
			// the order in the unusedMachines is not stable.
			var unusedGot, unusedExpected []string
			for _, m := range got.unusedMachines {
				unusedGot = append(unusedGot, m.Spec.IPv4[0])
			}
			for _, m := range tt.want.unusedMachines {
				unusedExpected = append(unusedExpected, m.Spec.IPv4[0])
			}
			sort.Strings(unusedGot)
			sort.Strings(unusedExpected)
			if !cmp.Equal(unusedGot, unusedExpected) {
				t.Errorf("!cmp.Equal(unusedGot, unusedExpected), actual: %v, want %v", unusedGot, unusedExpected)
			}
			if !cmp.Equal(got.cpTmpl, tt.want.cpTmpl, cmpopts.IgnoreUnexported(cke.Node{})) {
				t.Errorf("!cmp.Equal(got.cpTmpl, tt.want.cpTmpl), actual: %v, want %v", got.cpTmpl, tt.want.cpTmpl)
			}
			if !cmp.Equal(got.workerTmpls, tt.want.workerTmpls, cmpopts.IgnoreUnexported(cke.Node{})) {
				t.Errorf("!cmp.Equal(got.workerTmpl, tt.want.workerTmpl), actual: %v, want %v", got.workerTmpls, tt.want.workerTmpls)
			}
		})
	}
}

func testGenerate(t *testing.T) {
	tmpl := &cke.Cluster{
		Name: "test",
		Nodes: []*cke.Node{
			{
				ControlPlane: true,
				Labels:       map[string]string{"foo": "bar"},
				Annotations:  map[string]string{"hoge": "fuga"},
				Taints:       []corev1.Taint{{Key: "foo", Effect: corev1.TaintEffectNoSchedule}},
			},
			{
				ControlPlane: false,
				Labels:       map[string]string{"foo": "aaa"},
				Annotations:  map[string]string{"hoge": "bbb"},
				Taints:       []corev1.Taint{{Key: "foo", Effect: corev1.TaintEffectNoExecute}},
			},
		},
	}

	// generated cluster w/o nodes
	generated := *tmpl
	generated.Nodes = nil

	machines := []Machine{
		newTestMachineWithIP(0, testFuture250, StateHealthy, "10.0.0.1", "cs"),
		newTestMachineWithIP(1, testFuture250, StateRetired, "10.0.0.2", "cs"),
		newTestMachineWithIP(0, testFuture250, StateUnhealthy, "10.0.0.3", "ss"),
		newTestMachineWithIP(1, testFuture250, StateHealthy, "10.0.0.4", "ss"),
	}

	tests := []struct {
		name     string
		template *cke.Cluster
		cstr     *cke.Constraints
		machines []Machine

		expectErr bool
	}{
		{
			"NoMachine",
			tmpl,
			cke.DefaultConstraints(),
			nil,
			true,
		},
		{
			"Success",
			tmpl,
			cke.DefaultConstraints(),
			machines,
			false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			g := NewGenerator(nil, tt.template, tt.cstr, tt.machines, testBaseTS)
			cluster, err := g.Generate()
			if err != nil {
				if !tt.expectErr {
					t.Error(err)
				}
				return
			}
			if tt.expectErr {
				t.Error("expected an error")
				return
			}
			nodes := cluster.Nodes
			cluster.Nodes = nil
			if !cmp.Equal(cluster, &generated) {
				t.Log(cmp.Diff(cluster, &generated))
				t.Errorf("cluster mismatch: actual=%#v, expected=%#v", cluster, &generated)
			}

			var cps, workers int
			for _, n := range nodes {
				if n.ControlPlane {
					cps++
				} else {
					workers++
				}
			}
			if cps != tt.cstr.ControlPlaneCount {
				t.Error(`cps != tt.cstr.ControlPlaneCount`, cps)
			}
			if workers != tt.cstr.MinimumWorkers {
				t.Error(`workers != tt.cstr.MinimumWorkers`, workers)
			}
		})
	}
}

func testRegenerate(t *testing.T) {
	machines := []Machine{
		newTestMachineWithIP(0, testFuture250, StateHealthy, "10.0.0.1", "cs"),
		newTestMachineWithIP(1, testFuture250, StateRetired, "10.0.0.2", "cs"),
		newTestMachineWithIP(0, testFuture250, StateUnhealthy, "10.0.0.3", "ss"),
		newTestMachineWithIP(1, testFuture250, StateHealthy, "10.0.0.4", "ss"),
	}

	cluster := &cke.Cluster{
		Name: "old",
		Nodes: []*cke.Node{
			{
				Address:      "10.0.0.4",
				ControlPlane: true,
			},
			{
				Address: "10.0.0.1",
			},
			{
				Address: "10.0.0.3",
			},
		},
	}

	tmpl := &cke.Cluster{
		Name: "new",
		Nodes: []*cke.Node{
			{
				ControlPlane: true,
			},
			{
				ControlPlane: false,
			},
		},
	}

	// generated cluster w/o nodes
	generated := *tmpl
	generated.Nodes = nil

	g := NewGenerator(cluster, tmpl, cke.DefaultConstraints(), machines, testBaseTS)
	regenerated, err := g.Regenerate()
	if err != nil {
		t.Fatal(err)
	}

	nodes := regenerated.Nodes
	regenerated.Nodes = nil
	if !cmp.Equal(regenerated, &generated) {
		t.Log(cmp.Diff(cluster, &generated))
		t.Errorf("cluster mismatch: actual=%#v, expected=%#v", regenerated, &generated)
	}

	if len(nodes) != 3 {
		t.Fatal(`len(nodes) != 3`, nodes)
	}
	if nodes[0].Address != "10.0.0.4" {
		t.Error(`nodes[0].Address != "10.0.0.4"`, nodes[0].Address)
	}
	if !nodes[0].ControlPlane {
		t.Error(`!nodes[0].ControlPlane`)
	}
	if nodes[1].Address != "10.0.0.1" {
		t.Error(`nodes[1].Address != "10.0.0.1"`, nodes[1].Address)
	}
	if nodes[1].ControlPlane {
		t.Error(`nodes[1].ControlPlane`)
	}
	if nodes[2].Address != "10.0.0.3" {
		t.Error(`nodes[2].Address != "10.0.0.3"`, nodes[2].Address)
	}
	if nodes[2].ControlPlane {
		t.Error(`nodes[2].ControlPlane`)
	}
}

func testUpdate(t *testing.T) {
	machines := []Machine{
		newTestMachineWithIP(0, testFuture500, StateHealthy, "10.0.0.1", "cs"),     // [0]
		newTestMachineWithIP(0, testFuture250, StateRetiring, "10.0.0.2", "cs"),    // [1]
		newTestMachineWithIP(1, testFuture250, StateUnhealthy, "10.0.0.3", "cs"),   // [2]
		newTestMachineWithIP(1, testFuture250, StateUnreachable, "10.0.0.4", "cs"), // [3]
		newTestMachineWithIP(2, testFuture250, StateUnreachable, "10.0.0.5", "cs"), // [4]
		newTestMachineWithIP(2, testFuture500, StateHealthy, "10.0.0.6", "cs"),     // [5]
		newTestMachineWithIP(3, testFuture1000, StateHealthy, "10.0.0.7", "cs"),    // [6]
		newTestMachineWithIP(3, testFuture500, StateHealthy, "10.0.0.8", "cs"),     // [7]
		newTestMachineWithIP(4, testFuture500, StateUpdating, "10.0.0.9", "cs"),    // [8]
	}
	cps := []*cke.Node{
		{Address: "10.0.0.1", ControlPlane: true},    // [0]
		{Address: "10.0.0.2", ControlPlane: true},    // [1]
		{Address: "10.0.0.3", ControlPlane: true},    // [2]
		{Address: "10.0.0.4", ControlPlane: true},    // [3]
		{Address: "10.0.0.5", ControlPlane: true},    // [4]
		{Address: "10.0.0.6", ControlPlane: true},    // [5]
		{Address: "10.0.0.7", ControlPlane: true},    // [6]
		{Address: "10.0.0.8", ControlPlane: true},    // [7]
		{Address: "10.0.0.9", ControlPlane: true},    // [8]
		{Address: "10.100.0.10", ControlPlane: true}, // [9]  non-existent
		{Address: "10.100.0.11", ControlPlane: true}, // [10] non-existent
	}
	workers := []*cke.Node{
		{Address: "10.0.0.1"}, // [0]
		{Address: "10.0.0.2", // [1]
			Taints: []corev1.Taint{
				{
					Key:    "cke.cybozu.com/state",
					Value:  "retiring",
					Effect: corev1.TaintEffectNoExecute,
				},
			}},
		{Address: "10.0.0.3"},    // [2]
		{Address: "10.0.0.4"},    // [3]
		{Address: "10.0.0.5"},    // [4]
		{Address: "10.0.0.6"},    // [5]
		{Address: "10.0.0.7"},    // [6]
		{Address: "10.0.0.8"},    // [7]
		{Address: "10.0.0.9"},    // [8]
		{Address: "10.100.0.10"}, // [9]  non-existent
		{Address: "10.100.0.11"}, // [10] non-existent
	}

	tmpl := &cke.Cluster{
		Name: "tmpl",
		Nodes: []*cke.Node{
			{
				ControlPlane: true,
			},
			{
				ControlPlane: false,
			},
		},
	}

	tests := []struct {
		name     string
		current  *cke.Cluster
		cstr     *cke.Constraints
		machines []Machine

		expectErr error
		expected  *cke.Cluster
	}{
		{
			"RemoveNonExistent",
			&cke.Cluster{
				Nodes: []*cke.Node{cps[0], cps[1], cps[9], workers[2], workers[3], workers[10]},
			},
			&cke.Constraints{
				ControlPlaneCount: 3,
				MinimumWorkers:    1,
			},
			machines,

			nil,
			&cke.Cluster{
				Nodes: []*cke.Node{cps[0], cps[1], cps[6], workers[2], workers[3]},
			},
		},
		{
			"TooManyNonExistent",
			&cke.Cluster{
				Nodes: []*cke.Node{cps[0], cps[3], cps[4], workers[5]},
			},
			&cke.Constraints{
				ControlPlaneCount: 3,
				MinimumWorkers:    1,
			},
			[]Machine{machines[3], machines[5]},

			errTooManyNonExistent,
			nil,
		},
		{
			"NotAvailable",
			&cke.Cluster{
				Nodes: []*cke.Node{cps[0], cps[3], cps[5], workers[6]},
			},
			&cke.Constraints{
				ControlPlaneCount: 3,
				MinimumWorkers:    1,
			},
			[]Machine{machines[0], machines[3], machines[6]},

			errNotAvailable,
			nil,
		},
		{
			"IncreaseCP",
			&cke.Cluster{
				Nodes: []*cke.Node{cps[0], cps[5], workers[6]},
			},
			&cke.Constraints{
				ControlPlaneCount: 3,
				MinimumWorkers:    1,
			},
			machines,

			nil,
			&cke.Cluster{
				Nodes: []*cke.Node{cps[0], cps[5], cps[7], workers[6]},
			},
		},
		{
			"DecreaseCPDemote",
			&cke.Cluster{
				Nodes: []*cke.Node{cps[0], cps[6], cps[7], workers[5]},
			},
			&cke.Constraints{
				ControlPlaneCount: 2,
				MinimumWorkers:    1,
			},
			machines,

			nil,
			&cke.Cluster{
				Nodes: []*cke.Node{cps[0], cps[6], workers[5], workers[7]},
			},
		},
		{
			"DecreaseCPRemove",
			&cke.Cluster{
				Nodes: []*cke.Node{cps[0], cps[6], cps[7], workers[5]},
			},
			&cke.Constraints{
				ControlPlaneCount: 2,
				MinimumWorkers:    1,
				MaximumWorkers:    1,
			},
			machines,

			nil,
			&cke.Cluster{
				Nodes: []*cke.Node{cps[0], cps[6], workers[5]},
			},
		},
		{
			"ReplaceCPDemoteAdd",
			&cke.Cluster{
				Nodes: []*cke.Node{cps[0], cps[2], cps[7], workers[5]},
			},
			&cke.Constraints{
				ControlPlaneCount: 3,
				MinimumWorkers:    1,
			},
			machines,

			nil,
			&cke.Cluster{
				Nodes: []*cke.Node{cps[0], cps[6], cps[7], workers[2], workers[5]},
			},
		},
		{
			"ReplaceCPPromoteDemote",
			&cke.Cluster{
				Nodes: []*cke.Node{cps[2], cps[5], cps[6], workers[7], workers[0]},
			},
			&cke.Constraints{
				ControlPlaneCount: 3,
				MinimumWorkers:    2,
				MaximumWorkers:    2,
			},
			machines,

			nil,
			&cke.Cluster{
				Nodes: []*cke.Node{cps[5], cps[6], cps[0], workers[7], workers[2]},
			},
		},
		{
			"ReplaceCPDrop",
			&cke.Cluster{
				Nodes: []*cke.Node{cps[0], cps[2], cps[7], workers[1]},
			},
			&cke.Constraints{
				ControlPlaneCount: 3,
				MinimumWorkers:    1,
				MaximumWorkers:    1,
			},
			machines,

			nil,
			&cke.Cluster{
				Nodes: []*cke.Node{cps[0], cps[6], cps[7], workers[1]},
			},
		},
		{
			"ReplaceCPFail",
			&cke.Cluster{
				Nodes: []*cke.Node{cps[0], cps[2], cps[7], workers[1]},
			},
			&cke.Constraints{
				ControlPlaneCount: 3,
				MinimumWorkers:    1,
				MaximumWorkers:    1,
			},
			[]Machine{machines[0], machines[1], machines[2], machines[7]},

			errNotAvailable,
			nil,
		},
		{
			"IncreaseWorkerUnlimited",
			&cke.Cluster{
				Nodes: []*cke.Node{cps[0], cps[5], workers[1]},
			},
			&cke.Constraints{
				ControlPlaneCount: 2,
				MinimumWorkers:    2,
			},
			machines,

			nil,
			&cke.Cluster{
				Nodes: []*cke.Node{cps[0], cps[5], workers[1], workers[6], workers[7]},
			},
		},
		{
			"IncreaseWorkerLimited",
			&cke.Cluster{
				Nodes: []*cke.Node{cps[0], cps[5], workers[1]},
			},
			&cke.Constraints{
				ControlPlaneCount: 2,
				MinimumWorkers:    2,
				MaximumWorkers:    2,
			},
			machines,

			nil,
			&cke.Cluster{
				Nodes: []*cke.Node{cps[0], cps[5], workers[1], workers[6]},
			},
		},
		{
			"DecreaseWorkerRemove",
			&cke.Cluster{
				Nodes: []*cke.Node{cps[0], cps[5], workers[1], workers[7]},
			},
			&cke.Constraints{
				ControlPlaneCount: 2,
				MinimumWorkers:    1,
			},
			machines,

			nil,
			&cke.Cluster{
				Nodes: []*cke.Node{cps[0], cps[5], workers[7]},
			},
		},
		{
			"DecreaseWorkerReplace",
			&cke.Cluster{
				Nodes: []*cke.Node{cps[0], cps[5], workers[1], workers[7]},
			},
			&cke.Constraints{
				ControlPlaneCount: 2,
				MinimumWorkers:    2,
				MaximumWorkers:    2,
			},
			machines,

			nil,
			&cke.Cluster{
				Nodes: []*cke.Node{cps[0], cps[5], workers[6], workers[7]},
			},
		},
		{
			"NotDecreaseWorker",
			&cke.Cluster{
				Nodes: []*cke.Node{cps[0], cps[5], workers[1], workers[7]},
			},
			&cke.Constraints{
				ControlPlaneCount: 2,
				MinimumWorkers:    2,
				MaximumWorkers:    2,
			},
			[]Machine{machines[0], machines[1], machines[5], machines[7]},

			nil,
			nil,
		},
		{
			"Taint",
			&cke.Cluster{
				Nodes: []*cke.Node{cps[5], workers[2], workers[7], {
					Address:      "10.0.0.1",
					ControlPlane: true,
					Taints: []corev1.Taint{{
						Key:    "cke.cybozu.com/state",
						Value:  "retiring",
						Effect: corev1.TaintEffectNoExecute,
					}},
				}},
			},
			&cke.Constraints{
				ControlPlaneCount: 2,
				MinimumWorkers:    2,
				MaximumWorkers:    2,
			},
			[]Machine{machines[0], machines[2], machines[5], machines[7]},

			nil,
			&cke.Cluster{
				Nodes: []*cke.Node{cps[0], cps[5], workers[7], {
					Address: "10.0.0.3",
					Taints: []corev1.Taint{{
						Key:    "cke.cybozu.com/state",
						Value:  "retiring",
						Effect: corev1.TaintEffectNoExecute,
					}},
				}},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			g := NewGenerator(tt.current, tmpl, tt.cstr, tt.machines, testBaseTS)
			got, err := g.Update()

			if err != tt.expectErr {
				if err != nil {
					t.Error("unexpected error:", err)
				} else {
					t.Error("expected an error")
				}
				return
			}
			if err != nil {
				return
			}

			if tt.expected == nil {
				if got != nil {
					t.Error("expected nop, but get changed", got)
				}
				return
			}

			gotCPs := make(map[string]bool)
			gotWorkers := make(map[string]bool)
			expectedCPs := make(map[string]bool)
			expectedWorkers := make(map[string]bool)
			for _, n := range got.Nodes {
				if n.ControlPlane {
					gotCPs[n.Address] = true
				} else {
					gotWorkers[n.Address] = true
				}
			}
			for _, n := range tt.expected.Nodes {
				if n.ControlPlane {
					expectedCPs[n.Address] = true
				} else {
					expectedWorkers[n.Address] = true
				}
			}

			if !cmp.Equal(gotCPs, expectedCPs) {
				t.Error(`!cmp.Equal(gotCPs, expectedCPs)`, gotCPs, expectedCPs)
				t.Log(cmp.Diff(gotCPs, expectedCPs))
			}
			if !cmp.Equal(gotWorkers, expectedWorkers) {
				t.Error(`!cmp.Equal(gotWorkers, expectedWorkers)`, gotWorkers, expectedWorkers)
				t.Log(cmp.Diff(gotWorkers, expectedWorkers))
			}
		})
	}
}

func testWeighted(t *testing.T) {
	addresses := []string{
		"10.0.0.1",
		"10.0.0.2",
		"10.0.0.3",
		"10.0.0.4",
		"10.0.0.5",
		"10.0.0.6",
		"10.0.0.7",
	}
	machines := []Machine{
		newTestMachineWithIP(0, testFuture500, StateHealthy, addresses[0], "cs"),
		newTestMachineWithIP(0, testFuture250, StateHealthy, addresses[1], "ss"),
		newTestMachineWithIP(1, testFuture250, StateHealthy, addresses[2], "cs"),
		newTestMachineWithIP(1, testFuture250, StateHealthy, addresses[3], "ss"),
		newTestMachineWithIP(2, testFuture250, StateHealthy, addresses[4], "cs"),
		newTestMachineWithIP(2, testFuture250, StateHealthy, addresses[5], "ss"),
		newTestMachineWithIP(3, testFuture1000, StateHealthy, addresses[6], "new"),
	}

	cases := []struct {
		name        string
		tmplCP      *cke.Node
		tmplWorkers []*cke.Node
		cstr        *cke.Constraints

		expectCPs         []string
		expectWorkerRoles map[string]int
	}{
		{
			"blank",
			&cke.Node{},
			[]*cke.Node{{}},
			cke.DefaultConstraints(),

			[]string{addresses[6]},
			map[string]int{"cs": 1},
		},
		{
			"cp-role",
			&cke.Node{
				Labels: map[string]string{
					"cke.cybozu.com/role": "cs",
				},
			},
			[]*cke.Node{{}},
			cke.DefaultConstraints(),

			[]string{addresses[0]},
			map[string]int{"new": 1},
		},
		{
			"worker-role",
			&cke.Node{},
			[]*cke.Node{{
				Labels: map[string]string{
					"cke.cybozu.com/role": "ss",
				},
			}},
			cke.DefaultConstraints(),

			[]string{addresses[6]},
			map[string]int{"ss": 1},
		},
		{
			"weight1",
			&cke.Node{},
			[]*cke.Node{
				{
					Labels: map[string]string{
						"cke.cybozu.com/role":   "cs",
						"cke.cybozu.com/weight": "3",
					},
				},
				{
					Labels: map[string]string{
						"cke.cybozu.com/role": "ss",
					},
				},
			},
			&cke.Constraints{
				ControlPlaneCount: 1,
				MinimumWorkers:    2,
			},

			[]string{addresses[6]},
			map[string]int{"cs": 1, "ss": 1},
		},
		{
			"weight2",
			&cke.Node{},
			[]*cke.Node{
				{
					Labels: map[string]string{
						"cke.cybozu.com/role":   "cs",
						"cke.cybozu.com/weight": "3",
					},
				},
				{
					Labels: map[string]string{
						"cke.cybozu.com/role": "ss",
					},
				},
			},
			&cke.Constraints{
				ControlPlaneCount: 1,
				MinimumWorkers:    4,
			},

			[]string{addresses[6]},
			map[string]int{"cs": 3, "ss": 1},
		},
		{
			"weight3",
			&cke.Node{},
			[]*cke.Node{
				{
					Labels: map[string]string{
						"cke.cybozu.com/role":   "cs",
						"cke.cybozu.com/weight": "0.1",
					},
				},
				{
					Labels: map[string]string{
						"cke.cybozu.com/role":   "ss",
						"cke.cybozu.com/weight": "0.3",
					},
				},
			},
			&cke.Constraints{
				ControlPlaneCount: 1,
				MinimumWorkers:    4,
			},

			[]string{addresses[6]},
			map[string]int{"cs": 1, "ss": 3},
		},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tt.tmplCP.ControlPlane = true
			tt.tmplCP.User = "cybozu"
			for _, w := range tt.tmplWorkers {
				w.User = "cybozu"
			}
			tmpl := &cke.Cluster{
				Name:  "test",
				Nodes: append(tt.tmplWorkers, tt.tmplCP),
			}

			g := NewGenerator(nil, tmpl, tt.cstr, machines, testBaseTS)
			cluster, err := g.Generate()
			if err != nil {
				t.Fatal(err)
			}

			var cps []string
			workerRoles := make(map[string]int)
			for _, n := range cluster.Nodes {
				if n.ControlPlane {
					cps = append(cps, n.Address)
					continue
				}

				role := n.Labels["cke.cybozu.com/role"]
				workerRoles[role]++
			}

			if !cmp.Equal(tt.expectCPs, cps) {
				t.Error("unexpected CPs", cmp.Diff(tt.expectCPs, cps))
			}
			if !cmp.Equal(tt.expectWorkerRoles, workerRoles) {
				t.Error("unexpected workers", cmp.Diff(tt.expectWorkerRoles, workerRoles))
			}
		})
	}
}

func TestGenerator(t *testing.T) {
	t.Run("MachineToNode", testMachineToNode)
	t.Run("New", testNewGenerator)
	t.Run("Generate", testGenerate)
	t.Run("Regenerate", testRegenerate)
	t.Run("Update", testUpdate)
	t.Run("Weighted", testWeighted)
}
