package cke

import (
	"testing"

	"github.com/coreos/etcd/etcdserver/etcdserverpb"
)

type EtcdTestCluster struct {
	Nodes        []*Node
	NodeStatuses map[string]*NodeStatus
	Etcd         EtcdClusterStatus
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
		Etcd: EtcdClusterStatus{
			IsHealthy: false,
		},
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
			IsHealthy: true,
			Members: map[string]*etcdserverpb.Member{
				"10.0.0.11": {ID: 0, Name: "10.0.0.11"},
				"10.0.1.11": {ID: 11, Name: "10.0.1.11"},
				"10.0.1.12": {ID: 12, Name: "10.0.1.12"},
				"10.0.1.13": {ID: 13, Name: "10.0.1.13"},
				"10.0.1.14": {ID: 14, Name: "10.0.1.14"},
			},
			InSyncMembers: map[string]bool{
				"10.0.0.11": true,
				"10.0.1.11": true,
				"10.0.1.12": false,
				"10.0.1.13": true,
				"10.0.1.14": false,
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
			IsHealthy: true,
			Members: map[string]*etcdserverpb.Member{
				"10.0.0.11": {ID: 0, Name: "10.0.0.11"},
				"10.0.0.12": {ID: 1, Name: "10.0.0.12"},
				"10.0.1.11": {ID: 2, Name: "10.0.1.11"},
				"10.0.1.12": {ID: 3, Name: "10.0.1.12"},
			},
			InSyncMembers: map[string]bool{
				"10.0.0.11": true,
				"10.0.0.12": true,
				"10.0.1.11": false,
				"10.0.1.12": false,
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
			IsHealthy: true,
			Members: map[string]*etcdserverpb.Member{
				"10.0.0.11": {ID: 0, Name: "10.0.0.11"},
				"10.0.0.12": {ID: 1, Name: "10.0.0.12"},
				"10.0.0.13": {ID: 2, Name: ""},
			},
			InSyncMembers: map[string]bool{
				"10.0.0.11": true,
				"10.0.0.12": true,
				"10.0.0.13": false,
				"10.0.0.14": false,
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
			IsHealthy: true,
			Members: map[string]*etcdserverpb.Member{
				"10.0.0.11": {ID: 0, Name: "10.0.0.11"},
				"10.0.0.12": {ID: 1, Name: "10.0.0.12"},
			},
			InSyncMembers: map[string]bool{
				"10.0.0.11": true,
				"10.0.0.12": true,
				"10.0.0.13": false,
				"10.0.0.14": false,
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
			IsHealthy: true,
			Members: map[string]*etcdserverpb.Member{
				"10.0.0.11": {ID: 1, Name: "10.0.0.11"},
				"10.0.0.12": {ID: 2, Name: "10.0.0.12"},
				"10.0.1.11": {ID: 11, Name: "10.0.1.11"},
			},
			InSyncMembers: map[string]bool{
				"10.0.0.11": true,
				"10.0.0.12": true,
				"10.0.1.11": true,
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
			IsHealthy: true,
			Members: map[string]*etcdserverpb.Member{
				"10.0.0.11": {ID: 1, Name: "10.0.0.11"},
				"10.0.0.12": {ID: 2, Name: "10.0.0.12"},
				"10.0.0.13": {ID: 3, Name: "10.0.0.13"},
			},
			InSyncMembers: map[string]bool{
				"10.0.0.11": true,
				"10.0.0.12": true,
				"10.0.0.13": true,
				"10.0.0.14": true,
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
			IsHealthy:     false,
			Members:       map[string]*etcdserverpb.Member{},
			InSyncMembers: map[string]bool{},
		},
	}
}

func OutdatedImageControlPlane() EtcdTestCluster {
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
			IsHealthy: true,
			Members: map[string]*etcdserverpb.Member{
				"10.0.0.11": {ID: 1, Name: "10.0.0.11"},
				"10.0.0.12": {ID: 2, Name: "10.0.0.12"},
				"10.0.0.13": {ID: 3, Name: "10.0.0.13"},
			},
			InSyncMembers: map[string]bool{
				"10.0.0.11": true,
				"10.0.0.12": true,
				"10.0.0.13": true,
			},
		},
	}
}

func OutdatedParamsControlPlane() EtcdTestCluster {
	oldParams := ServiceParams{ExtraArguments: []string{"--experimental-enable-v2v3"}}
	return EtcdTestCluster{
		Nodes: []*Node{
			{Address: "10.0.0.11", ControlPlane: true},
			{Address: "10.0.0.12", ControlPlane: true},
			{Address: "10.0.0.13", ControlPlane: true},
		},
		NodeStatuses: map[string]*NodeStatus{
			"10.0.0.11": {Etcd: EtcdStatus{ServiceStatus: ServiceStatus{ExtraParams: oldParams, Image: EtcdImage.Name(), Running: true}, HasData: true}},
			"10.0.0.12": {Etcd: EtcdStatus{ServiceStatus: ServiceStatus{ExtraParams: oldParams, Image: EtcdImage.Name(), Running: true}, HasData: true}},
			"10.0.0.13": {Etcd: EtcdStatus{ServiceStatus: ServiceStatus{ExtraParams: oldParams, Image: EtcdImage.Name(), Running: true}, HasData: true}},
		},
		Etcd: EtcdClusterStatus{
			IsHealthy: true,
			Members: map[string]*etcdserverpb.Member{
				"10.0.0.11": {ID: 1, Name: "10.0.0.11"},
				"10.0.0.12": {ID: 2, Name: "10.0.0.12"},
				"10.0.0.13": {ID: 3, Name: "10.0.0.13"},
			},
			InSyncMembers: map[string]bool{
				"10.0.0.11": true,
				"10.0.0.12": true,
				"10.0.0.13": true,
			},
		},
	}
}

func testEtcdDecideToDo(t *testing.T) {
	cases := []struct {
		Name             string
		Input            EtcdTestCluster
		ExpectedOperator string
	}{
		{
			Name:             "Bootstrap",
			Input:            Clean3Nodes(),
			ExpectedOperator: "etcd-bootstrap",
		},
		{
			Name:             "RemoveUnhealthyNonCluster",
			Input:            UnhealthyNonCluster(),
			ExpectedOperator: "etcd-remove-member",
		},
		{
			Name:             "RemoveUnhealthyNonControlPlane",
			Input:            UnhealthyNonControlPlane(),
			ExpectedOperator: "etcd-destroy-member",
		},
		{
			Name:             "StartUnstartedMember",
			Input:            UnstartedMembers(),
			ExpectedOperator: "etcd-add-member",
		},
		{
			Name:             "AddNewMember",
			Input:            NewlyControlPlane(),
			ExpectedOperator: "etcd-add-member",
		},
		{
			Name:             "RemoveHealthyNonCluster",
			Input:            HealthyNonCluster(),
			ExpectedOperator: "etcd-remove-member",
		},
		{
			Name:             "RemoveHealthyNonControlPlane",
			Input:            HealthyNonControlPlane(),
			ExpectedOperator: "etcd-destroy-member",
		},
		{
			Name:             "WaitUnhealthyControlPlane",
			Input:            UnhealthyControlPlane(),
			ExpectedOperator: "etcd-wait-cluster",
		},
		{
			Name:             "UpdateOutdatedImage",
			Input:            OutdatedImageControlPlane(),
			ExpectedOperator: "etcd-update-version",
		},
		{
			Name:             "UpdateOutdatedParams",
			Input:            OutdatedParamsControlPlane(),
			ExpectedOperator: "etcd-restart",
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
		if op.Name() != c.ExpectedOperator {
			t.Errorf("[%s] operator name mismatch: %s != %s", c.Name, op.Name(), c.ExpectedOperator)
		}
	}
}

func TestEtcdStrategy(t *testing.T) {
	t.Run("EtcdDecideToDo", testEtcdDecideToDo)
}
