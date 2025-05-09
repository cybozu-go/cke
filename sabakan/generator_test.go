package sabakan

import (
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/cke/op"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	res1 := MachineToNode(machine, node)

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
	if res1.Labels["topology.kubernetes.io/zone"] != "rack0" {
		t.Error(`res1.Labels["topology.kubernetes.io/zone"] != "rack0", actual:`, res1.Labels)
	}
	if res1.Labels["node-role.kubernetes.io/worker"] != "true" {
		t.Error(`res1.Lables["node-role.kubernetes.io/worker"] != "true", actual:`, res1.Labels)
	}
	if res1.Labels["node-role.kubernetes.io/master"] != "true" {
		t.Error(`res1.Lables["node-role.kubernetes.io/master"] != "true", actual:`, res1.Labels)
	}
	if res1.Labels["node-role.kubernetes.io/control-plane"] != "true" {
		t.Error(`res1.Lables["node-role.kubernetes.io/control-plane"] != "true", actual:`, res1.Labels)
	}
	if res1.Labels[domain+"/register-month"] != testPast250.Format("2006-01") {
		t.Error(`res1.Labels["cke.cybozu.com/register-month"] != machine.Spec.RegisterDate.Format("2006-01"), actual:`, res1.Labels)
	}
	if res1.Labels[domain+"/retire-month"] != testBaseTS.Format("2006-01") {
		t.Error(`res1.Labels["cke.cybozu.com/register-month"] != machine.Spec.RetireDate.Format("2006-01"), actual:`, res1.Labels)
	}
	if !containsTaint(res1.Taints, corev1.Taint{Key: "foo", Effect: corev1.TaintEffectNoSchedule}) {
		t.Error(`res1.Taints do not have corev1.Taint{Key"foo", Effect: corev1.TaintEffectNoSchedule}, actual:`, res1.Taints)
	}

	machine.Status.State = StateUnreachable
	res2 := MachineToNode(machine, node)
	if !containsTaint(res2.Taints, corev1.Taint{Key: domain + "/state", Value: "unreachable", Effect: corev1.TaintEffectNoSchedule}) {
		t.Error(`res2.Taints do not have corev1.Taint{Key: "cke.cybozu.com/state", Value: "unreachable", Effect: "NoSchedule"}, actual:`, res2.Taints)
	}
	machine.Status.State = StateRetiring
	res3 := MachineToNode(machine, node)
	if !containsTaint(res3.Taints, corev1.Taint{Key: domain + "/state", Value: "retiring", Effect: corev1.TaintEffectNoExecute}) {
		t.Error(`res3.Taints do not have corev1.Taint{Key: "cke.cybozu.com/state", Value: "retiring", Effect: "NoExecute"}, actual:`, res3.Taints)
	}
	machine.Status.State = StateRetired
	res4 := MachineToNode(machine, node)
	if !containsTaint(res4.Taints, corev1.Taint{Key: domain + "/state", Value: "retired", Effect: corev1.TaintEffectNoExecute}) {
		t.Error(`res4.Taints do not have corev1.Taint{Key: "cke.cybozu.com/state", Value: "retired", Effect: "NoExecute"}, actual:`, res4.Taints)
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
	m.Status.Duration = DefaultWaitRetiredSeconds * 2
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
					"foo":                 "aaa",
					"cke.cybozu.com/role": "cs",
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
	k8sNode := corev1.Node{}
	k8sNode.Name = "10.0.0.1"
	k8sNode.Spec.Taints = []corev1.Taint{{Key: "key", Value: "value", Effect: corev1.TaintEffectNoExecute}}
	clusterStatus := &cke.ClusterStatus{Kubernetes: cke.KubernetesClusterStatus{Nodes: []corev1.Node{k8sNode}}}
	type args struct {
		current       *cke.Cluster
		template      *cke.Cluster
		cstr          *cke.Constraints
		machines      []Machine
		clusterStatus *cke.ClusterStatus
	}
	tests := []struct {
		name string
		args args
		want *Generator
	}{
		{
			"NoCluster",
			args{nil, tmpl, cke.DefaultConstraints(), machines, nil},
			&Generator{
				machineMap: map[string]*Machine{
					"10.0.0.1": &machines[0],
					"10.0.0.2": &machines[1],
					"10.0.0.3": &machines[2],
					"10.0.0.4": &machines[3],
				},
				k8sNodeMap: map[string]*corev1.Node{},
				cpTmpl:     nodeTemplate{tmpl.Nodes[0], ""},
				workerTmpls: []nodeTemplate{
					{tmpl.Nodes[1], "cs"},
				},
			},
		},
		{
			"Cluster",
			args{cluster, tmpl, cke.DefaultConstraints(), machines, clusterStatus},
			&Generator{
				machineMap: map[string]*Machine{
					"10.0.0.1": &machines[0],
					"10.0.0.2": &machines[1],
					"10.0.0.3": &machines[2],
					"10.0.0.4": &machines[3],
				},
				k8sNodeMap: map[string]*corev1.Node{"10.0.0.1": &k8sNode},
				cpTmpl:     nodeTemplate{tmpl.Nodes[0], ""},
				workerTmpls: []nodeTemplate{
					{tmpl.Nodes[1], "cs"},
				},
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := NewGenerator(tt.args.template, tt.args.cstr, tt.args.machines, tt.args.clusterStatus, testBaseTS)
			if !cmp.Equal(got.machineMap, tt.want.machineMap, cmpopts.EquateEmpty()) {
				t.Errorf("!cmp.Equal(got.machineMap, tt.want.machineMap), actual: %v, want %v", got.machineMap, tt.want.machineMap)
			}
			if !cmp.Equal(got.k8sNodeMap, tt.want.k8sNodeMap, cmpopts.EquateEmpty()) {
				t.Errorf("!cmp.Equal(got.k8sNodeMap, tt.want.k8sNodeMap), actual: %v, want %v", got.k8sNodeMap, tt.want.k8sNodeMap)
			}
			// the order in the nextUnused is not stable.
			var unusedGot, unusedExpected []string
			for _, m := range got.nextUnused {
				unusedGot = append(unusedGot, m.Spec.IPv4[0])
			}
			for _, m := range tt.want.nextUnused {
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
		Name:          "test",
		ServiceSubnet: "10.0.0.0/14",
		Nodes: []*cke.Node{
			{
				User:         "cybozu",
				ControlPlane: true,
				Labels:       map[string]string{"foo": "bar"},
				Annotations:  map[string]string{"hoge": "fuga"},
				Taints:       []corev1.Taint{{Key: "foo", Effect: corev1.TaintEffectNoSchedule}},
			},
			{
				User:         "cybozu",
				ControlPlane: false,
				Labels:       map[string]string{"foo": "aaa"},
				Annotations:  map[string]string{"hoge": "bbb"},
				Taints:       []corev1.Taint{{Key: "foo", Effect: corev1.TaintEffectNoExecute}},
			},
		},
		Options: cke.Options{
			Kubelet: cke.KubeletParams{
				CRIEndpoint: "/var/run/k8s-containerd.sock",
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

			g := NewGenerator(tt.template, tt.cstr, tt.machines, nil, testBaseTS)
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

	k8sNode := corev1.Node{}
	k8sNode.Name = "10.0.0.4"
	k8sNode.Spec.Taints = []corev1.Taint{{Key: "regenerate_will_not_care_about_taints", Value: "even_for_control_plane_node", Effect: corev1.TaintEffectNoExecute}}
	clusterStatus := &cke.ClusterStatus{Kubernetes: cke.KubernetesClusterStatus{Nodes: []corev1.Node{k8sNode}}}

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
		Name:          "new",
		ServiceSubnet: "10.0.0.0/14",
		Nodes: []*cke.Node{
			{
				User:         "cybozu",
				ControlPlane: true,
			},
			{
				User:         "cybozu",
				ControlPlane: false,
			},
		},
		Options: cke.Options{
			Kubelet: cke.KubeletParams{
				CRIEndpoint: "/var/run/k8s-containerd.sock",
			},
		},
	}

	// generated cluster w/o nodes
	generated := *tmpl
	generated.Nodes = nil

	g := NewGenerator(tmpl, cke.DefaultConstraints(), machines, clusterStatus, testBaseTS)
	regenerated, err := g.Regenerate(cluster)
	if err != nil {
		t.Fatal(err)
	}

	nodes := regenerated.Nodes
	regenerated.Nodes = nil
	if !cmp.Equal(regenerated, &generated) {
		t.Log(cmp.Diff(regenerated, &generated))
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
		newTestMachineWithIP(4, testFuture500, StateRetired, "10.0.0.10", "cs"),    // [9]
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
		{Address: "10.100.0.10", ControlPlane: true}, // [9]
		{Address: "10.100.0.11", ControlPlane: true}, // [10] non-existent
		{Address: "10.100.0.12", ControlPlane: true}, // [11] non-existent
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
		{Address: "10.0.0.3"}, // [2]
		{Address: "10.0.0.4"}, // [3]
		{Address: "10.0.0.5"}, // [4]
		{Address: "10.0.0.6"}, // [5]
		{Address: "10.0.0.7"}, // [6]
		{Address: "10.0.0.8"}, // [7]
		{Address: "10.0.0.9"}, // [8]
		{Address: "10.0.0.10", // [9]
			Taints: []corev1.Taint{
				{
					Key:    "cke.cybozu.com/state",
					Value:  "retired",
					Effect: corev1.TaintEffectNoExecute,
				},
			}},
		{Address: "10.100.0.11"}, // [10] non-existent
		{Address: "10.100.0.12"}, // [11] non-existent
	}

	var k8sUntaintedNodes, k8sSystemTaintedNodes, k8sUserTaintedNodes []corev1.Node
	for _, m := range machines {
		n := corev1.Node{}
		n.Name = m.Spec.IPv4[0]
		k8sUntaintedNodes = append(k8sUntaintedNodes, n)
		n.Spec.Taints = []corev1.Taint{
			{Key: corev1.TaintNodeNotReady, Effect: corev1.TaintEffectNoSchedule},
			{Key: op.CKETaintMaster, Effect: corev1.TaintEffectNoSchedule},
			{Key: "foo.cybozu.com/transient", Effect: corev1.TaintEffectNoSchedule},
		}
		k8sSystemTaintedNodes = append(k8sSystemTaintedNodes, n)
		n.Spec.Taints = []corev1.Taint{{Key: "foo", Value: "bar", Effect: corev1.TaintEffectNoSchedule}}
		k8sUserTaintedNodes = append(k8sUserTaintedNodes, n)
	}

	tmpl := &cke.Cluster{
		Name:          "tmpl",
		ServiceSubnet: "10.0.0.0/14",
		Nodes: []*cke.Node{
			{
				User:         "cybozu",
				ControlPlane: true,
			},
			{
				User:         "cybozu",
				ControlPlane: false,
			},
		},
		CPTolerations: []string{"foo.cybozu.com/transient"},
		Options: cke.Options{
			Kubelet: cke.KubeletParams{
				CRIEndpoint: "/var/run/k8s-containerd.sock",
			},
		},
		Sabakan: cke.Sabakan{
			SpareNodeTaintKey: "node.cybozu.io/spare",
		},
	}

	tests := []struct {
		name          string
		current       *cke.Cluster
		cstr          *cke.Constraints
		machines      []Machine
		clusterStatus *cke.ClusterStatus

		expectErr error
		expected  *cke.Cluster
	}{
		{
			"RemoveNonExistent",
			&cke.Cluster{
				Nodes: []*cke.Node{cps[0], cps[1], cps[10], workers[2], workers[3], workers[11]},
			},
			&cke.Constraints{
				ControlPlaneCount: 3,
			},
			machines,
			nil,

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
			},
			[]Machine{machines[3], machines[5]},
			nil,

			errTooManyNonExistent,
			nil,
		},
		{
			"NotAvailable",
			&cke.Cluster{
				Nodes: []*cke.Node{cps[0], cps[3], cps[5], workers[4]},
			},
			&cke.Constraints{
				ControlPlaneCount: 3,
			},
			[]Machine{machines[0], machines[3], machines[4]},
			nil,

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
			},
			machines,
			nil,

			nil,
			&cke.Cluster{
				Nodes: []*cke.Node{cps[0], cps[5], cps[7], workers[6]},
			},
		},
		{
			"IncreaseCPIgnoringLongLifeButUserTaintedNode",
			&cke.Cluster{
				Nodes: []*cke.Node{cps[0], cps[5], workers[8]},
			},
			&cke.Constraints{
				ControlPlaneCount: 3,
			},
			machines,
			&cke.ClusterStatus{
				Kubernetes: cke.KubernetesClusterStatus{
					Nodes: []corev1.Node{k8sUserTaintedNodes[6], k8sSystemTaintedNodes[7]},
				},
			},

			nil,
			&cke.Cluster{
				Nodes: []*cke.Node{cps[0], cps[5], cps[7], workers[8]},
			},
		},
		{
			"DecreaseCPRemoveAllCPsHealthy",
			&cke.Cluster{
				Nodes: []*cke.Node{cps[0], cps[6], cps[7], workers[5]},
			},
			&cke.Constraints{
				ControlPlaneCount: 2,
			},
			machines,
			nil,

			nil,
			&cke.Cluster{
				Nodes: []*cke.Node{cps[0], cps[6], workers[5], workers[7]},
			},
		},
		{
			"DecreaseCPRemoveUnhealthyCP",
			&cke.Cluster{
				Nodes: []*cke.Node{cps[2], cps[6], cps[7], workers[5]},
			},
			&cke.Constraints{
				ControlPlaneCount: 2,
			},
			machines,
			nil,

			nil,
			&cke.Cluster{
				Nodes: []*cke.Node{cps[6], cps[7], workers[5], workers[2]},
			},
		},
		{
			"ReplaceCPDemoteAdd",
			&cke.Cluster{
				Nodes: []*cke.Node{cps[0], cps[2], cps[7], workers[5]},
			},
			&cke.Constraints{
				ControlPlaneCount: 3,
			},
			machines,
			nil,

			nil,
			&cke.Cluster{
				Nodes: []*cke.Node{cps[0], cps[5], cps[7], workers[2]},
			},
		},
		{
			"ReplaceCPSpareAdd",
			&cke.Cluster{
				Nodes: []*cke.Node{cps[0], cps[2], cps[7], workers[5], workers[6]},
			},
			&cke.Constraints{
				ControlPlaneCount: 3,
			},
			machines,
			&cke.ClusterStatus{
				Kubernetes: cke.KubernetesClusterStatus{
					Nodes: []corev1.Node{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: workers[6].Address,
							},
							Spec: corev1.NodeSpec{
								Taints: []corev1.Taint{
									{
										Key:    "node.cybozu.io/spare",
										Value:  "true",
										Effect: corev1.TaintEffectNoSchedule,
									},
								},
							},
						},
					},
				},
			},
			nil,
			&cke.Cluster{
				Nodes: []*cke.Node{cps[0], cps[6], cps[7], workers[2], workers[5]},
			},
		},
		{
			"ReplaceCPDemoteTaintedAddUntainted",
			&cke.Cluster{
				Nodes: []*cke.Node{cps[0], cps[5], cps[7], workers[2]},
			},
			&cke.Constraints{
				ControlPlaneCount: 3,
			},
			machines,
			&cke.ClusterStatus{
				Kubernetes: cke.KubernetesClusterStatus{
					Nodes: []corev1.Node{k8sUserTaintedNodes[5], k8sUntaintedNodes[6]},
				},
			},

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
			},
			machines,
			nil,

			nil,
			&cke.Cluster{
				Nodes: []*cke.Node{cps[5], cps[6], cps[0], workers[7], workers[2]},
			},
		},
		{
			"ReplaceCPFail",
			&cke.Cluster{
				Nodes: []*cke.Node{cps[0], cps[2], cps[7], workers[1]},
			},
			&cke.Constraints{
				ControlPlaneCount: 3,
			},
			[]Machine{machines[0], machines[1], machines[2], machines[7]},
			nil,

			errNotAvailable,
			nil,
		},
		{
			"IncreaseWorker",
			&cke.Cluster{
				Nodes: []*cke.Node{cps[0], cps[5], workers[1]},
			},
			&cke.Constraints{
				ControlPlaneCount: 2,
			},
			machines,
			nil,

			nil,
			&cke.Cluster{
				Nodes: []*cke.Node{cps[0], cps[5], workers[1], workers[6], workers[7]},
			},
		},
		{
			"DecreaseWorkerRemove",
			&cke.Cluster{
				Nodes: []*cke.Node{cps[0], cps[5], workers[9], workers[7], workers[6]},
			},
			&cke.Constraints{
				ControlPlaneCount: 2,
			},
			machines,
			nil,

			nil,
			&cke.Cluster{
				Nodes: []*cke.Node{cps[0], cps[5], workers[7], workers[6]},
			},
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
			},
			[]Machine{machines[0], machines[2], machines[5], machines[7]},
			nil,

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
		{
			"UpdateNotRequired",
			&cke.Cluster{
				Nodes: []*cke.Node{cps[0], cps[5], workers[1], workers[7], workers[9]},
			},
			&cke.Constraints{
				ControlPlaneCount:  2,
				MinimumWorkersRate: 90,
			},
			[]Machine{machines[0], machines[1], machines[5], machines[7], machines[9]},
			nil,

			nil,
			nil,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			g := NewGenerator(tmpl, tt.cstr, tt.machines, tt.clusterStatus, testBaseTS)
			got, err := g.Update(tt.current)

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

func testRegenerateAfterUpdate(t *testing.T) {
	machines := []Machine{
		newTestMachineWithIP(0, testFuture250, StateHealthy, "10.0.0.1", "cs"),
		newTestMachineWithIP(1, testFuture250, StateRetired, "10.0.0.2", "cs"),
		newTestMachineWithIP(0, testFuture250, StateUnreachable, "10.0.0.3", "cs"),
		newTestMachineWithIP(1, testFuture250, StateHealthy, "10.0.0.4", "cs"),
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
		},
	}

	tmpl := &cke.Cluster{
		Name:          "new",
		ServiceSubnet: "10.0.0.0/14",
		Nodes: []*cke.Node{
			{
				User:         "cybozu",
				ControlPlane: true,
			},
			{
				User:         "cybozu",
				ControlPlane: false,
			},
		},
		Options: cke.Options{
			Kubelet: cke.KubeletParams{
				CRIEndpoint: "/var/run/k8s-containerd.sock",
			},
		},
	}

	g := NewGenerator(tmpl, cke.DefaultConstraints(), machines, nil, testBaseTS)

	got, err := g.Update(cluster)
	if err != nil {
		t.Error("unexpected error:", err)
		return
	}
	if got != nil {
		t.Error("unexpected update:", got)
		return
	}

	got, err = g.Regenerate(cluster)
	if err != nil {
		t.Error("unexpected error:", err)
		return
	}

	nodes := got.Nodes
	if len(nodes) != 2 {
		t.Fatal(`len(nodes) != 2`, nodes)
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
}

func TestCountMachinesByRack(t *testing.T) {
	{
		// control plane
		g := &Generator{nextControlPlanes: []*Machine{}}
		racks := []int{0, 0, 1}
		for _, r := range racks {
			m := &Machine{}
			m.Spec.Rack = r
			g.nextControlPlanes = append(g.nextControlPlanes, m)
		}

		bin := g.countMachinesByRack(true, "")
		if bin[0] != 2 || bin[1] != 1 {
			t.Errorf(
				"rack0: expect 2 actual %d, rack1: expect 1 actual %d",
				bin[0],
				bin[1],
			)
		}
	}

	{
		// empty controlplane
		g := &Generator{nextControlPlanes: []*Machine{}}
		bin := g.countMachinesByRack(true, "")
		if len(bin) != 0 {
			t.Errorf("len(bin): expect 0 actual %d", len(bin))
		}
	}

	{
		// worker
		g := &Generator{nextWorkers: []*Machine{}}
		racks := []int{0, 0, 1}
		roles := []string{"cs", "ss", "cs"}
		for i, r := range racks {
			m := &Machine{}
			m.Spec.Rack = r
			m.Spec.Role = roles[i]
			g.nextWorkers = append(g.nextWorkers, m)
		}

		bin := g.countMachinesByRack(false, "cs")
		if bin[0] != 1 || bin[1] != 1 {
			t.Errorf(
				"rack0: expect 2 actual %d rack1: expect 1 actual %d",
				bin[0],
				bin[1],
			)
		}
		bin = g.countMachinesByRack(false, "ss")
		if bin[0] != 1 || bin[1] != 0 {
			t.Errorf(
				"rack0: expect 2 actual %d rack1: expect 1 actual %d",
				bin[0],
				bin[1],
			)
		}
		bin = g.countMachinesByRack(false, "")
		if bin[0] != 2 || bin[1] != 1 {
			t.Errorf(
				"rack0: expect 2 actual %d rack1: expect 1 actual %d",
				bin[0],
				bin[1],
			)
		}
	}

	{
		// empty worker
		g := &Generator{nextWorkers: []*Machine{}}
		bin := g.countMachinesByRack(false, "")
		if len(bin) != 0 {
			t.Errorf("len(bin): expect 0 actual %d", len(bin))
		}
	}
}

func testRackDistribution(t *testing.T) {
	baseTemplate := &cke.Cluster{
		Name:          "test",
		ServiceSubnet: "10.0.0.0/14",
		Nodes: []*cke.Node{
			{
				User:         "cybozu",
				ControlPlane: true,
				Labels: map[string]string{
					"cke.cybozu.com/role": "cs",
				},
			},
			{
				User:         "cybozu",
				ControlPlane: false,
				Labels: map[string]string{
					"cke.cybozu.com/role": "cs",
				},
			},
			{
				User:         "cybozu",
				ControlPlane: false,
				Labels: map[string]string{
					"cke.cybozu.com/role": "ss",
				},
			},
		},
		Options: cke.Options{
			Kubelet: cke.KubeletParams{
				CRIEndpoint: "/var/run/k8s-containerd.sock",
			},
		},
	}
	baseConstraints := &cke.Constraints{
		ControlPlaneCount: 3,
	}

	t.Run("GenerateInitialCluster", func(t *testing.T) {
		rackIDs := []int{0, 1, 2}
		var machines []Machine
		for i := range rackIDs {
			machines = append(machines, createRack(i, "standard")...)
		}

		g := NewGenerator(baseTemplate, baseConstraints, machines, nil, testBaseTS)
		cluster, err := g.Generate()
		if err != nil {
			t.Fatal(err)
		}

		expect := map[string]map[string]int{
			"0": {"cs": 17, "ss": 10},
			"1": {"cs": 17, "ss": 10},
			"2": {"cs": 17, "ss": 10},
		}
		actual := countRolesByRack(cluster.Nodes)
		if !cmp.Equal(actual, expect) {
			t.Errorf("expect=%v, actual=%v", expect, actual)
		}
	})

	t.Run("MultipleRackTypes", func(t *testing.T) {
		rackIDs := []int{0, 1, 2, 3}
		var machines []Machine
		for i := range rackIDs {
			if i == 3 {
				machines = append(machines, createRack(i, "storage")...)
				continue
			}
			machines = append(machines, createRack(i, "standard")...)
		}

		g := NewGenerator(baseTemplate, baseConstraints, machines, nil, testBaseTS)
		cluster, err := g.Generate()
		if err != nil {
			t.Fatal(err)
		}

		expect := map[string]map[string]int{
			"0": {"cs": 17, "ss": 10},
			"1": {"cs": 17, "ss": 10},
			"2": {"cs": 17, "ss": 10},
			"3": {"ss": 19},
		}
		actual := countRolesByRack(cluster.Nodes)
		if !cmp.Equal(actual, expect) {
			t.Errorf("expect=%v, actual=%v", expect, actual)
		}
	})

	t.Run("ReachMaxMachinesInRack", func(t *testing.T) {
		rackIDs := []int{0, 1, 2}
		var machines []Machine
		for i := range rackIDs {
			if i == 2 {
				machines = append(machines, createRack(i, "storage")...)
				continue
			}
			machines = append(machines, createRack(i, "standard")...)
		}

		constraints := &cke.Constraints{
			ControlPlaneCount: 3,
		}

		withoutCSTemplate := &cke.Cluster{
			Name:          "test",
			ServiceSubnet: "10.0.0.0/14",
			Nodes:         []*cke.Node{},
			Options: cke.Options{
				Kubelet: cke.KubeletParams{
					CRIEndpoint: "/var/run/k8s-containerd.sock",
				},
			},
		}
		for i, n := range baseTemplate.Nodes {
			if i == 1 {
				continue
			}
			withoutCSTemplate.Nodes = append(withoutCSTemplate.Nodes, n)
		}

		g := NewGenerator(withoutCSTemplate, constraints, machines, nil, testBaseTS)
		cluster, err := g.Generate()
		if err != nil {
			t.Fatal(err)
		}

		expect := map[string]map[string]int{
			"0": {"ss": 10},
			"1": {"ss": 10},
			"2": {"ss": 19},
		}
		actual := countRolesByRack(cluster.Nodes)
		if !cmp.Equal(actual, expect) {
			t.Errorf("expect=%v, actual=%v", expect, actual)
		}
	})

	t.Run("IncreaseRack", func(t *testing.T) {
		rackIDs := []int{0, 1, 2}
		var machines []Machine
		for i := range rackIDs {
			machines = append(machines, createRack(i, "standard")...)
		}

		g := NewGenerator(baseTemplate, baseConstraints, machines, nil, testBaseTS)
		cluster, err := g.Generate()
		if err != nil {
			t.Fatal(err)
		}

		machines = append(machines, createRack(3, "storage")...)
		constraints := &cke.Constraints{
			ControlPlaneCount: 3,
		}
		g = NewGenerator(baseTemplate, constraints, machines, nil, testBaseTS)
		cluster, err = g.Update(cluster)
		if err != nil {
			t.Fatal(err)
		}

		expect := map[string]map[string]int{
			"0": {"cs": 17, "ss": 10},
			"1": {"cs": 17, "ss": 10},
			"2": {"cs": 17, "ss": 10},
			"3": {"ss": 19},
		}
		actual := countRolesByRack(cluster.Nodes)
		if !cmp.Equal(actual, expect) {
			t.Errorf("expect=%v, actual=%v", expect, actual)
		}
	})

	t.Run("DisappearRack", func(t *testing.T) {
		rackIDs := []int{0, 1, 2, 3}
		var machines []Machine
		for i := range rackIDs {
			machines = append(machines, createRack(i, "standard")...)
		}

		g := NewGenerator(baseTemplate, baseConstraints, machines, nil, testBaseTS)
		cluster, err := g.Generate()
		if err != nil {
			t.Fatal(err)
		}

		machines = machines[28:] // Disappear rack0
		g = NewGenerator(baseTemplate, baseConstraints, machines, nil, testBaseTS)
		cluster, err = g.Update(cluster)
		if err != nil {
			t.Fatal(err)
		}

		expect := map[string]map[string]int{
			"1": {"cs": 17, "ss": 10},
			"2": {"cs": 17, "ss": 10},
			"3": {"cs": 17, "ss": 10},
		}
		actual := countRolesByRack(cluster.Nodes)
		if !cmp.Equal(actual, expect) {
			t.Errorf("expect=%v, actual=%v", expect, actual)
		}
	})

	t.Run("OneAllUnhealthyRack", func(t *testing.T) {
		rackIDs := []int{0, 1, 2, 3}
		var machines []Machine
		for i := range rackIDs {
			machines = append(machines, createRack(i, "standard")...)
		}

		for i := 0; i < 28; i++ {
			machines[i].Status.State = StateUnhealthy
		}
		g := NewGenerator(baseTemplate, baseConstraints, machines, nil, testBaseTS)
		cluster, err := g.Generate()
		if err != nil {
			t.Fatal(err)
		}

		expect := map[string]map[string]int{
			"1": {"cs": 17, "ss": 10},
			"2": {"cs": 17, "ss": 10},
			"3": {"cs": 17, "ss": 10},
		}
		actual := countRolesByRack(cluster.Nodes)
		if !cmp.Equal(actual, expect) {
			t.Errorf("expect=%v, actual=%v", expect, actual)
		}
	})
}

func createRack(rack int, rackType string) []Machine {
	var numCS, numSS int
	switch rackType {
	case "standard":
		numCS = 18
		numSS = 10
	case "storage":
		numCS = 0
		numSS = 19
	default:
		return nil
	}

	var machines []Machine
	for i := 0; i < numCS; i++ {
		addr := fmt.Sprintf("10.0.%d.%d", rack, i+1)
		machines = append(machines, newTestMachineWithIP(rack, testBaseTS, StateHealthy, addr, "cs"))
	}
	for i := numCS; i < numCS+numSS; i++ {
		addr := fmt.Sprintf("10.0.%d.%d", rack, i+1)
		machines = append(machines, newTestMachineWithIP(rack, testBaseTS, StateHealthy, addr, "ss"))
	}

	return machines
}

func countRolesByRack(nodes []*cke.Node) map[string]map[string]int {
	count := make(map[string]map[string]int)
	for _, n := range nodes {
		if n.ControlPlane {
			continue
		}
		if _, ok := count[n.Labels["cke.cybozu.com/rack"]]; !ok {
			count[n.Labels["cke.cybozu.com/rack"]] = make(map[string]int)
		}
		count[n.Labels["cke.cybozu.com/rack"]][n.Labels["cke.cybozu.com/role"]]++
	}

	return count
}

func TestGenerator(t *testing.T) {
	t.Run("MachineToNode", testMachineToNode)
	t.Run("New", testNewGenerator)
	t.Run("Generate", testGenerate)
	t.Run("Regenerate", testRegenerate)
	t.Run("Update", testUpdate)
	t.Run("RegenerateAfterUpdate", testRegenerateAfterUpdate)
	t.Run("RackDistribution", testRackDistribution)
}
