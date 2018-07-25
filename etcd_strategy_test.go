package cke

import (
	"strconv"
	"strings"
	"testing"

	"github.com/coreos/etcd/etcdserver/etcdserverpb"
)

type EtcdTestCluster struct {
	Nodes        []*Node
	NodeStatuses map[string]*NodeStatus
	Etcd         EtcdClusterStatus
}

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

func Clean3Nodes() EtcdTestCluster {
	return EtcdTestCluster{
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
	}
}

func UnhealthyNonCluster() EtcdTestCluster {
	return EtcdTestCluster{
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
	}
}

func UnhealthyNonControlPlane() EtcdTestCluster {
	return EtcdTestCluster{
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
	}
}

func UnstartedMembers() EtcdTestCluster {
	return EtcdTestCluster{
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
	}
}

func NewlyControlPlane() EtcdTestCluster {
	return EtcdTestCluster{
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
	}
}

func HealthyNonCluster() EtcdTestCluster {
	return EtcdTestCluster{
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
			MemberHealth: map[string]EtcdNodeHealth{
				"10.0.0.11": EtcdNodeHealthy,
				"10.0.0.12": EtcdNodeHealthy,
				"10.0.1.11": EtcdNodeHealthy,
			},
		},
	}
}

func HealthyNonControlPlane() EtcdTestCluster {
	return EtcdTestCluster{
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
	}
}

func BootstrapCommands(targets ...string) []Command {
	hosts := strings.Join(targets, ",")
	commands := []Command{
		{Name: "image-pull", Target: "etcd"},
		{Name: "volume-create", Target: hosts},
	}
	for _, addr := range targets {
		commands = append(commands, Command{Name: "run-container", Target: addr})
	}
	return commands
}

func AddMemberCommands(addr string) []Command {
	return []Command{
		{Name: "image-pull", Target: "etcd"},
		{Name: "volume-remove", Target: addr},
		{Name: "volume-create", Target: addr},
		{Name: "add-etcd-member", Target: addr},
		{Name: "wait-etcd-sync", Target: "http://" + addr + ":2379"},
	}
}

func RemoveMemberCommands(ids ...uint64) []Command {
	var ss []string
	for _, id := range ids {
		ss = append(ss, strconv.FormatUint(id, 10))
	}
	return []Command{{Name: "remove-etcd-member", Target: strings.Join(ss, ",")}}
}

func DestroyMemberCommands(cps []string, addrs []string, ids map[string]uint64) []Command {
	var endpoints []string
	for _, cp := range cps {
		endpoints = append(endpoints, "http://"+cp+":2379")
	}
	var commands []Command
	for _, addr := range addrs {
		commands = append(commands,
			Command{Name: "remove-etcd-member", Target: strconv.FormatUint(ids[addr], 10)},
			Command{Name: "wait-etcd-sync", Target: strings.Join(endpoints, ",")},
			Command{Name: "stop-container", Target: addr},
			Command{Name: "volume-remove", Target: addr},
		)
	}
	return commands
}

func testEtcdDecideToDo(t *testing.T) {
	cases := []struct {
		Name             string
		Input            EtcdTestCluster
		ExpectedCommands []Command
	}{
		{
			Name:             "Bootstrap",
			Input:            Clean3Nodes(),
			ExpectedCommands: BootstrapCommands("10.0.0.11", "10.0.0.12", "10.0.0.13"),
		},
		{
			Name:             "RemoveUnhealthyNonCluster",
			Input:            UnhealthyNonCluster(),
			ExpectedCommands: RemoveMemberCommands(12, 14),
		},
		{
			Name:  "RemoveUnhealthyNonControlPlane",
			Input: UnhealthyNonControlPlane(),
			ExpectedCommands: DestroyMemberCommands(
				[]string{"10.0.0.11", "10.0.0.12"},
				[]string{"10.0.1.11", "10.0.1.12"},
				map[string]uint64{"10.0.1.11": 2, "10.0.1.12": 3}),
		},
		{
			Name:             "StartUnstartedMember",
			Input:            UnstartedMembers(),
			ExpectedCommands: AddMemberCommands("10.0.0.13"),
		},
		{
			Name:             "AddNewMember",
			Input:            NewlyControlPlane(),
			ExpectedCommands: AddMemberCommands("10.0.0.13"),
		},
		{
			Name:             "RemoveHealthyNonCluster",
			Input:            HealthyNonCluster(),
			ExpectedCommands: RemoveMemberCommands(11),
		},
		{
			Name:  "RemoveHealthyNonControlPlane",
			Input: HealthyNonControlPlane(),
			ExpectedCommands: DestroyMemberCommands(
				[]string{"10.0.0.11", "10.0.0.12"},
				[]string{"10.0.0.13"},
				map[string]uint64{"10.0.0.13": 3}),
		},
	}

	for _, c := range cases {
		cluster := &Cluster{
			Nodes: c.Input.Nodes,
		}
		clusterStatus := &ClusterStatus{
			NodeStatuses: c.Input.NodeStatuses,
			Etcd:         c.Input.Etcd,
		}
		result := etcdDecideToDoCommands(cluster, clusterStatus)
		if len(c.ExpectedCommands) != len(result) {
			t.Errorf("[%s] commands length mismatch: %d", c.Name, len(result))
			continue
		}
		for i, res := range result {
			com := c.ExpectedCommands[i]
			if com.Name != res.Name {
				t.Errorf("[%s] command name mismatch: %s != %s", c.Name, com.Name, res.Name)
			}
			if com.Target != res.Target {
				t.Errorf("[%s] command '%s' target mismatch: %s != %s", c.Name, com.Name, com.Target, res.Target)
			}
		}
	}
}

func TestEtcdStrategy(t *testing.T) {
	t.Run("EtcdDecideToDo", testEtcdDecideToDo)
}
