package cke

import "testing"

func testEtcdDecideToDo(t *testing.T) {
	cases := []struct {
		Nodes        []*Node
		NodeStatuses map[string]*NodeStatus
		Etcd         EtcdClusterStatus
		Commands     []Command
	}{
		{
			Nodes: []*Node{
				{Address: "10.0.0.11", ControlPlane: true},
				{Address: "10.0.0.12", ControlPlane: true},
				{Address: "10.0.0.13", ControlPlane: true},
				{Address: "10.0.0.14"},
				{Address: "10.0.0.15"},
				{Address: "10.0.0.16"},
			},
			NodeStatuses: map[string]*NodeStatus{
				"10.0.0.11": {Etcd: EtcdStatus{ServiceStatus: ServiceStatus{Running: false}, HasData: false}},
				"10.0.0.12": {Etcd: EtcdStatus{ServiceStatus: ServiceStatus{Running: false}, HasData: false}},
				"10.0.0.13": {Etcd: EtcdStatus{ServiceStatus: ServiceStatus{Running: false}, HasData: false}},
				"10.0.0.14": {Etcd: EtcdStatus{ServiceStatus: ServiceStatus{Running: false}, HasData: false}},
				"10.0.0.15": {Etcd: EtcdStatus{ServiceStatus: ServiceStatus{Running: false}, HasData: false}},
				"10.0.0.16": {Etcd: EtcdStatus{ServiceStatus: ServiceStatus{Running: false}, HasData: false}},
			},
			Etcd: EtcdClusterStatus{},
			Commands: []Command{
				Command{Name: "image-pull", Target: "etcd"},
				Command{Name: "volume-create", Target: "10.0.0.11,10.0.0.12,10.0.0.13"},
				Command{Name: "run-container", Target: "10.0.0.11"},
				Command{Name: "run-container", Target: "10.0.0.12"},
				Command{Name: "run-container", Target: "10.0.0.13"},
			},
		},
	}

	for _, c := range cases {
		cluster := &Cluster{
			Nodes: c.Nodes,
		}
		clusterStatus := &ClusterStatus{
			NodeStatuses: c.NodeStatuses,
			Etcd:         c.Etcd,
		}
		op := etcdDecideToDo(cluster, clusterStatus)
		if op == nil {
			t.Fatal("operator is nil")
		}
		var result []Command
		for {
			commander := op.NextCommand()
			if commander == nil {
				break
			}
			result = append(result, commander.Command())
		}
		if len(c.Commands) != len(result) {
			t.Fatal("commands length mismatch")
		}
		for i, res := range result {
			com := c.Commands[i]
			if com.Name != res.Name {
				t.Error("command name mismatch:", com.Name, res.Name)
			}
			if com.Target != res.Target {
				t.Error("command target mismatch:", com.Target, res.Target)
			}
		}
	}
}

func TestEtcdStrategy(t *testing.T) {
	t.Run("EtcdDecideToDo", testEtcdDecideToDo)
}
