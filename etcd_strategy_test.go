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

func opCommands(op Operator) []Command {
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
				"10.0.0.11": {ID: 0, Name: "10.0.0.11"},
				"10.0.1.11": {ID: 11, Name: "10.0.1.11"},
				"10.0.1.12": {ID: 12, Name: "10.0.1.12"},
				"10.0.1.13": {ID: 13, Name: "10.0.1.13"},
				"10.0.1.14": {ID: 14, Name: "10.0.1.14"},
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
				"10.0.0.11": {ID: 0, Name: "10.0.0.11"},
				"10.0.0.12": {ID: 1, Name: "10.0.0.12"},
				"10.0.1.11": {ID: 2, Name: "10.0.1.11"},
				"10.0.1.12": {ID: 3, Name: "10.0.1.12"},
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
				"10.0.0.11": {ID: 0, Name: "10.0.0.11"},
				"10.0.0.12": {ID: 1, Name: "10.0.0.12"},
				"10.0.0.13": {ID: 2, Name: ""},
			},
			MemberHealth: map[string]EtcdNodeHealth{
				"10.0.0.11": EtcdNodeHealthy,
				"10.0.0.12": EtcdNodeHealthy,
				"10.0.0.13": EtcdNodeUnhealthy,
				"10.0.0.14": EtcdNodeUnhealthy,
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
				"10.0.0.11": {ID: 0, Name: "10.0.0.11"},
				"10.0.0.12": {ID: 1, Name: "10.0.0.12"},
			},
			MemberHealth: map[string]EtcdNodeHealth{
				"10.0.0.11": EtcdNodeHealthy,
				"10.0.0.12": EtcdNodeHealthy,
				"10.0.0.13": EtcdNodeUnhealthy,
				"10.0.0.14": EtcdNodeUnhealthy,
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
				"10.0.0.11": {ID: 1, Name: "10.0.0.11"},
				"10.0.0.12": {ID: 2, Name: "10.0.0.12"},
				"10.0.1.11": {ID: 11, Name: "10.0.1.11"},
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
				"10.0.0.11": {ID: 1, Name: "10.0.0.11"},
				"10.0.0.12": {ID: 2, Name: "10.0.0.12"},
				"10.0.0.13": {ID: 3, Name: "10.0.0.13"},
			},
			MemberHealth: map[string]EtcdNodeHealth{
				"10.0.0.11": EtcdNodeHealthy,
				"10.0.0.12": EtcdNodeHealthy,
				"10.0.0.13": EtcdNodeHealthy,
				"10.0.0.14": EtcdNodeUnhealthy,
			},
		},
	}
}

func UnhealthyControlPlane() EtcdTestCluster {
	return EtcdTestCluster{
		Nodes: []*Node{
			{Address: "10.0.0.11", ControlPlane: true},
			{Address: "10.0.0.12", ControlPlane: true},
		},
		NodeStatuses: map[string]*NodeStatus{
			"10.0.0.11": {Etcd: EtcdStatus{ServiceStatus: ServiceStatus{Running: true}, HasData: true}},
			"10.0.0.12": {Etcd: EtcdStatus{ServiceStatus: ServiceStatus{Running: true}, HasData: true}},
		},
		Etcd: EtcdClusterStatus{
			Members: map[string]*etcdserverpb.Member{
				"10.0.0.11": {ID: 1, Name: "10.0.0.11"},
				"10.0.0.12": {ID: 1, Name: "10.0.0.12"},
			},
			MemberHealth: map[string]EtcdNodeHealth{
				"10.0.0.11": EtcdNodeUnhealthy,
				"10.0.0.12": EtcdNodeUnhealthy,
			},
		},
	}
}

func OutdatedControlPlane() EtcdTestCluster {
	oldEtcd := "quay.io/cybozu/etcd:3.2.18-2"
	return EtcdTestCluster{
		Nodes: []*Node{
			{Address: "10.0.0.11", ControlPlane: true},
			{Address: "10.0.0.12", ControlPlane: true},
			{Address: "10.0.0.13", ControlPlane: true},
		},
		NodeStatuses: map[string]*NodeStatus{
			"10.0.0.11": {Etcd: EtcdStatus{ServiceStatus: ServiceStatus{Running: true, Image: oldEtcd}, HasData: true}},
			"10.0.0.12": {Etcd: EtcdStatus{ServiceStatus: ServiceStatus{Running: true, Image: oldEtcd}, HasData: true}},
			"10.0.0.13": {Etcd: EtcdStatus{ServiceStatus: ServiceStatus{Running: true, Image: oldEtcd}, HasData: true}},
		},
		Etcd: EtcdClusterStatus{
			Members: map[string]*etcdserverpb.Member{
				"10.0.0.11": {ID: 1, Name: "10.0.0.11"},
				"10.0.0.12": {ID: 2, Name: "10.0.0.12"},
				"10.0.0.13": {ID: 3, Name: "10.0.0.13"},
			},
			MemberHealth: map[string]EtcdNodeHealth{
				"10.0.0.11": EtcdNodeHealthy,
				"10.0.0.12": EtcdNodeHealthy,
				"10.0.0.13": EtcdNodeHealthy,
			},
		},
	}
}

func NotInMemberControlPlane() EtcdTestCluster {
	return EtcdTestCluster{
		Nodes: []*Node{
			{Address: "10.0.0.11", ControlPlane: true},
			{Address: "10.0.0.12", ControlPlane: true},
		},
		NodeStatuses: map[string]*NodeStatus{
			"10.0.0.11": {Etcd: EtcdStatus{ServiceStatus: ServiceStatus{Running: true}, HasData: true}},
			"10.0.0.12": {Etcd: EtcdStatus{ServiceStatus: ServiceStatus{Running: true}, HasData: true}},
		},
		Etcd: EtcdClusterStatus{
			Members:      map[string]*etcdserverpb.Member{},
			MemberHealth: map[string]EtcdNodeHealth{},
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
	var endpoints []string
	for _, target := range targets {
		endpoints = append(endpoints, "http://"+target+":2379")
	}
	commands = append(commands, Command{Name: "wait-etcd-sync", Target: strings.Join(endpoints, ",")})
	return commands
}

func AddMemberCommands(addr string) []Command {
	return []Command{
		{Name: "image-pull", Target: "etcd"},
		{Name: "stop-container", Target: addr},
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

func DestroyMemberCommands(cps []string, addrs []string, ids []uint64) []Command {
	var endpoints []string
	for _, cp := range cps {
		endpoints = append(endpoints, "http://"+cp+":2379")
	}
	var commands []Command
	for i, addr := range addrs {
		commands = append(commands,
			Command{Name: "remove-etcd-member", Target: strconv.FormatUint(ids[i], 10)},
			Command{Name: "wait-etcd-sync", Target: strings.Join(endpoints, ",")},
			Command{Name: "stop-container", Target: addr},
			Command{Name: "volume-remove", Target: addr},
		)
	}
	return commands
}

func UpdateMemberCommands(cps []string) []Command {
	var commands []Command
	for _, cp := range cps {
		commands = append(commands,
			Command{Name: "wait-etcd-sync", Target: "http://" + cp + ":2379"},
			Command{Name: "image-pull", Target: "etcd"},
			Command{Name: "stop-container", Target: cp},
			Command{Name: "run-container", Target: cp},
		)
	}
	return commands
}

func WaitMemberCommands(cps []string) []Command {
	var endpoints []string
	for _, cp := range cps {
		endpoints = append(endpoints, "http://"+cp+":2379")
	}
	return []Command{
		{Name: "wait-etcd-sync", Target: strings.Join(endpoints, ",")},
	}
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
				[]uint64{2, 3}),
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
				[]uint64{3}),
		},
		{
			Name:             "WaitUnhealthyControlPlane",
			Input:            UnhealthyControlPlane(),
			ExpectedCommands: WaitMemberCommands([]string{"10.0.0.11", "10.0.0.12"}),
		},
		{
			Name:             "UpdateOutdatedControlPlane",
			Input:            OutdatedControlPlane(),
			ExpectedCommands: UpdateMemberCommands([]string{"10.0.0.11", "10.0.0.12", "10.0.0.13"}),
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

		op := etcdDecideToDo(cluster, clusterStatus)
		if op == nil {
			t.Fatal("op == nil")
		}
		cmds := opCommands(op)
		if len(c.ExpectedCommands) != len(cmds) {
			t.Errorf("[%s] commands length mismatch: %d", c.Name, len(cmds))
			continue
		}
		for i, res := range cmds {
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
