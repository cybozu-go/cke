package cke

import (
	"testing"

	"github.com/coreos/etcd/etcdserver/etcdserverpb"
)

func etcdDecideToDoCommands(c *Cluster, cs *ClusterStatus) []Command {
	op := etcdDecideToDo(c, cs)
	if op == nil {
		return nil
	}
	var commands []Command
	for {
		commander := op.NextCommand()
		if commander == nil {
			break
		}
		commands = append(commands, commander.Command())
	}
	return commands
}

func testEtcdDecideToDoBootstrap(t *testing.T) {
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
		result := etcdDecideToDoCommands(cluster, clusterStatus)
		if len(c.Commands) != len(result) {
			t.Fatal("results length mismatch")
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

func testRemoveUnhealthyNonCluster(t *testing.T) {
	cases := []struct {
		Nodes        []*Node
		NodeStatuses map[string]*NodeStatus
		Etcd         EtcdClusterStatus
		Commands     []Command
	}{
		{
			Nodes: []*Node{
				{Address: "10.0.0.11", ControlPlane: true},
				{Address: "10.0.0.12"},
			},
			NodeStatuses: map[string]*NodeStatus{
				"10.0.0.11": {Etcd: EtcdStatus{ServiceStatus: ServiceStatus{Running: true}, HasData: true}},
				"10.0.0.12": {Etcd: EtcdStatus{}},
			},
			Etcd: EtcdClusterStatus{
				Members: map[string]*etcdserverpb.Member{
					"10.0.0.11": &etcdserverpb.Member{ID: 0, Name: "10.0.0.11"},
					"10.0.1.11": &etcdserverpb.Member{ID: 11, Name: "10.0.0.11"},
					"10.0.1.12": &etcdserverpb.Member{ID: 12, Name: "10.0.0.12"},
					"10.0.1.13": &etcdserverpb.Member{ID: 13, Name: "10.0.0.13"},
					"10.0.1.14": &etcdserverpb.Member{ID: 14, Name: "10.0.0.14"},
				},
				MemberHealth: map[string]EtcdNodeHealth{
					"10.0.0.11": EtcdNodeHealthy,
					"10.0.1.11": EtcdNodeHealthy,
					"10.0.1.12": EtcdNodeUnhealthy,
					"10.0.1.13": EtcdNodeHealthy,
					"10.0.1.14": EtcdNodeUnhealthy,
				},
			},
			Commands: []Command{
				Command{Name: "remove-etcd-member", Target: "12,14"},
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
		result := etcdDecideToDoCommands(cluster, clusterStatus)
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

func testRemoveUnhealthyNonControlPlane(t *testing.T) {
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
				{Address: "10.0.1.11"},
				{Address: "10.0.1.12"},
			},
			NodeStatuses: map[string]*NodeStatus{
				"10.0.0.11": {Etcd: EtcdStatus{ServiceStatus: ServiceStatus{Running: true}, HasData: true}},
				"10.0.0.12": {Etcd: EtcdStatus{ServiceStatus: ServiceStatus{Running: true}, HasData: true}},
				"10.0.1.11": {Etcd: EtcdStatus{ServiceStatus: ServiceStatus{Running: true}, HasData: true}},
				"10.0.1.12": {Etcd: EtcdStatus{ServiceStatus: ServiceStatus{Running: true}, HasData: true}},
			},
			Etcd: EtcdClusterStatus{
				Members: map[string]*etcdserverpb.Member{
					"10.0.0.11": &etcdserverpb.Member{ID: 0, Name: "10.0.0.11"},
					"10.0.0.12": &etcdserverpb.Member{ID: 1, Name: "10.0.0.12"},
					"10.0.1.11": &etcdserverpb.Member{ID: 2, Name: "10.0.1.11"},
					"10.0.1.12": &etcdserverpb.Member{ID: 3, Name: "10.0.1.12"},
				},
				MemberHealth: map[string]EtcdNodeHealth{
					"10.0.0.11": EtcdNodeHealthy,
					"10.0.0.12": EtcdNodeHealthy,
					"10.0.1.11": EtcdNodeUnhealthy,
					"10.0.1.12": EtcdNodeUnhealthy,
				},
			},
			Commands: []Command{
				Command{Name: "remove-etcd-member", Target: "2"},
				Command{Name: "wait-etcd-sync", Target: "http://10.0.0.11:2379,http://10.0.0.12:2379"},
				Command{Name: "stop-container", Target: "10.0.1.11"},
				Command{Name: "volume-remove", Target: "10.0.1.11"},

				Command{Name: "remove-etcd-member", Target: "3"},
				Command{Name: "wait-etcd-sync", Target: "http://10.0.0.11:2379,http://10.0.0.12:2379"},
				Command{Name: "stop-container", Target: "10.0.1.12"},
				Command{Name: "volume-remove", Target: "10.0.1.12"},
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
		result := etcdDecideToDoCommands(cluster, clusterStatus)
		if len(c.Commands) != len(result) {
			t.Error("commands length mismatch: ", len(result))
			continue
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

func testStartUnstartedNode(t *testing.T) {
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
			},
			NodeStatuses: map[string]*NodeStatus{
				"10.0.0.11": {Etcd: EtcdStatus{ServiceStatus: ServiceStatus{Running: true}, HasData: true}},
				"10.0.0.12": {Etcd: EtcdStatus{ServiceStatus: ServiceStatus{Running: true}, HasData: true}},
				"10.0.0.13": {},
				"10.0.0.14": {},
			},
			Etcd: EtcdClusterStatus{
				Members: map[string]*etcdserverpb.Member{
					"10.0.0.11": &etcdserverpb.Member{ID: 0, Name: "10.0.0.11"},
					"10.0.0.12": &etcdserverpb.Member{ID: 1, Name: "10.0.0.12"},
					"10.0.0.13": &etcdserverpb.Member{ID: 2, Name: ""},
				},
			},
			Commands: []Command{
				Command{Name: "image-pull", Target: "etcd"},
				Command{Name: "volume-remove", Target: "10.0.0.13"},
				Command{Name: "volume-create", Target: "10.0.0.13"},
				Command{Name: "add-etcd-member", Target: "10.0.0.13"},
				Command{Name: "wait-etcd-sync", Target: "http://10.0.0.13:2379"},
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
		result := etcdDecideToDoCommands(cluster, clusterStatus)
		if len(c.Commands) != len(result) {
			t.Error("commands length mismatch: ", len(result))
			continue
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

func testAddNewControlPlane(t *testing.T) {
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
			},
			NodeStatuses: map[string]*NodeStatus{
				"10.0.0.11": {Etcd: EtcdStatus{ServiceStatus: ServiceStatus{Running: true}, HasData: true}},
				"10.0.0.12": {Etcd: EtcdStatus{ServiceStatus: ServiceStatus{Running: true}, HasData: true}},
				"10.0.0.13": {},
				"10.0.0.14": {},
			},
			Etcd: EtcdClusterStatus{
				Members: map[string]*etcdserverpb.Member{
					"10.0.0.11": &etcdserverpb.Member{ID: 0, Name: "10.0.0.11"},
					"10.0.0.12": &etcdserverpb.Member{ID: 1, Name: "10.0.0.12"},
				},
			},
			Commands: []Command{
				Command{Name: "image-pull", Target: "etcd"},
				Command{Name: "volume-remove", Target: "10.0.0.13"},
				Command{Name: "volume-create", Target: "10.0.0.13"},
				Command{Name: "add-etcd-member", Target: "10.0.0.13"},
				Command{Name: "wait-etcd-sync", Target: "http://10.0.0.13:2379"},
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
		result := etcdDecideToDoCommands(cluster, clusterStatus)
		if len(c.Commands) != len(result) {
			t.Error("commands length mismatch: ", len(result))
			continue
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

func testRemoveHealthyNonCluster(t *testing.T) {
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
			},
			NodeStatuses: map[string]*NodeStatus{
				"10.0.0.11": {Etcd: EtcdStatus{ServiceStatus: ServiceStatus{Running: true}, HasData: true}},
				"10.0.0.12": {Etcd: EtcdStatus{ServiceStatus: ServiceStatus{Running: true}, HasData: true}},
				"10.0.1.11": {Etcd: EtcdStatus{ServiceStatus: ServiceStatus{Running: true}, HasData: true}},
			},
			Etcd: EtcdClusterStatus{
				Members: map[string]*etcdserverpb.Member{
					"10.0.0.11": &etcdserverpb.Member{ID: 1, Name: "10.0.0.11"},
					"10.0.0.12": &etcdserverpb.Member{ID: 2, Name: "10.0.0.12"},
					"10.0.1.11": &etcdserverpb.Member{ID: 11, Name: "10.0.1.11"},
				},
			},
			Commands: []Command{
				Command{Name: "remove-etcd-member", Target: "11"},
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
		result := etcdDecideToDoCommands(cluster, clusterStatus)
		if len(c.Commands) != len(result) {
			t.Error("commands length mismatch: ", len(result))
			continue
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

func testRemoveNonControlPlane(t *testing.T) {
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
				{Address: "10.0.0.13"},
				{Address: "10.0.0.14"},
			},
			NodeStatuses: map[string]*NodeStatus{
				"10.0.0.11": {Etcd: EtcdStatus{ServiceStatus: ServiceStatus{Running: true}, HasData: true}},
				"10.0.0.12": {Etcd: EtcdStatus{ServiceStatus: ServiceStatus{Running: true}, HasData: true}},
				"10.0.0.13": {Etcd: EtcdStatus{ServiceStatus: ServiceStatus{Running: true}, HasData: true}},
				"10.0.0.14": {},
			},
			Etcd: EtcdClusterStatus{
				Members: map[string]*etcdserverpb.Member{
					"10.0.0.11": &etcdserverpb.Member{ID: 1, Name: "10.0.0.11"},
					"10.0.0.12": &etcdserverpb.Member{ID: 2, Name: "10.0.0.12"},
					"10.0.0.13": &etcdserverpb.Member{ID: 3, Name: "10.0.0.13"},
				},
			},
			Commands: []Command{
				Command{Name: "remove-etcd-member", Target: "3"},
				Command{Name: "wait-etcd-sync", Target: "http://10.0.0.11:2379,http://10.0.0.12:2379"},
				Command{Name: "stop-container", Target: "10.0.0.13"},
				Command{Name: "volume-remove", Target: "10.0.0.13"},
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
		result := etcdDecideToDoCommands(cluster, clusterStatus)
		if len(c.Commands) != len(result) {
			t.Error("commands length mismatch: ", len(result))
			continue
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

func testEtcdDecideToDo(t *testing.T) {
	t.Run("BootstrapEtcd", testEtcdDecideToDoBootstrap)
	t.Run("RemoveUnhealthyNonCluster", testRemoveUnhealthyNonCluster)
	t.Run("RemoveUnhealthyNonControlPlane", testRemoveUnhealthyNonControlPlane)
	t.Run("StartUnstartedNode", testStartUnstartedNode)
	t.Run("AddNewControlPlane", testAddNewControlPlane)
	t.Run("RemoveHealthyNonCluster", testRemoveHealthyNonCluster)
	t.Run("RemoveHealthyNonControlPlane", testRemoveNonControlPlane)
}

func TestEtcdStrategy(t *testing.T) {
	t.Run("EtcdDecideToDo", testEtcdDecideToDo)
}
