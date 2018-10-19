package server

import (
	"reflect"
	"sort"
	"testing"

	"github.com/coreos/etcd/etcdserver/etcdserverpb"
	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/cke/op"
	corev1 "k8s.io/api/core/v1"
)

const (
	testClusterName   = "test"
	testServiceSubnet = "12.34.56.0/24"
)

type testData struct {
	Cluster *cke.Cluster
	Status  *cke.ClusterStatus
}

func (d testData) ControlPlane() (nodes []*cke.Node) {
	for _, n := range d.Cluster.Nodes {
		if n.ControlPlane {
			nodes = append(nodes, n)
		}
	}
	return nodes
}

func (d testData) NodeStatus(n *cke.Node) *cke.NodeStatus {
	return d.Status.NodeStatuses[n.Address]
}

func newData() testData {
	cluster := &cke.Cluster{
		Name: testClusterName,
		Nodes: []*cke.Node{
			{Address: "10.0.0.11", ControlPlane: true},
			{Address: "10.0.0.12", ControlPlane: true},
			{Address: "10.0.0.13", ControlPlane: true},
			{Address: "10.0.0.14"},
			{Address: "10.0.0.15"},
			{Address: "10.0.0.16"},
		},
		ServiceSubnet: testServiceSubnet,
	}
	status := &cke.ClusterStatus{
		NodeStatuses: map[string]*cke.NodeStatus{
			"10.0.0.11": {Etcd: cke.EtcdStatus{ServiceStatus: cke.ServiceStatus{Running: false}, HasData: false}},
			"10.0.0.12": {Etcd: cke.EtcdStatus{ServiceStatus: cke.ServiceStatus{Running: false}, HasData: false}},
			"10.0.0.13": {Etcd: cke.EtcdStatus{ServiceStatus: cke.ServiceStatus{Running: false}, HasData: false}},
			"10.0.0.14": {Etcd: cke.EtcdStatus{ServiceStatus: cke.ServiceStatus{Running: false}, HasData: false}},
			"10.0.0.15": {Etcd: cke.EtcdStatus{ServiceStatus: cke.ServiceStatus{Running: false}, HasData: false}},
			"10.0.0.16": {Etcd: cke.EtcdStatus{ServiceStatus: cke.ServiceStatus{Running: false}, HasData: false}},
		},
	}

	return testData{cluster, status}
}

func (d testData) with(f func(data testData)) testData {
	f(d)
	return d
}

func (d testData) withRivers() testData {
	for _, v := range d.Status.NodeStatuses {
		v.Rivers.Running = true
		v.Rivers.Image = cke.ToolsImage.Name()
		v.Rivers.BuiltInParams = op.RiversParams(d.ControlPlane())
	}
	return d
}

func (d testData) withStoppedEtcd() testData {
	for _, n := range d.ControlPlane() {
		d.NodeStatus(n).Etcd.HasData = true
	}
	return d
}

func (d testData) withUnhealthyEtcd() testData {
	d.withStoppedEtcd()
	for _, n := range d.ControlPlane() {
		st := &d.NodeStatus(n).Etcd
		st.Running = true
		st.Image = cke.EtcdImage.Name()
		st.BuiltInParams = op.EtcdBuiltInParams(n, nil, "")
	}
	return d
}

func (d testData) withHealthyEtcd() testData {
	d.withUnhealthyEtcd()
	st := &d.Status.Etcd
	st.IsHealthy = true
	st.Members = make(map[string]*etcdserverpb.Member)
	st.InSyncMembers = make(map[string]bool)
	for i, n := range d.ControlPlane() {
		st.Members[n.Address] = &etcdserverpb.Member{
			ID:   uint64(i),
			Name: n.Address,
		}
		st.InSyncMembers[n.Address] = true
	}
	return d
}

func (d testData) withAPIServer(serviceSubnet string) testData {
	for _, n := range d.ControlPlane() {
		st := &d.NodeStatus(n).APIServer
		st.Running = true
		st.IsHealthy = true
		st.Image = cke.HyperkubeImage.Name()
		st.BuiltInParams = op.APIServerParams(d.ControlPlane(), n.Address, serviceSubnet)
	}
	return d
}

func (d testData) withControllerManager(name, serviceSubnet string) testData {
	for _, n := range d.ControlPlane() {
		st := &d.NodeStatus(n).ControllerManager
		st.Running = true
		st.IsHealthy = true
		st.Image = cke.HyperkubeImage.Name()
		st.BuiltInParams = op.ControllerManagerParams(name, serviceSubnet)
	}
	return d
}

func (d testData) withScheduler() testData {
	for _, n := range d.ControlPlane() {
		st := &d.NodeStatus(n).Scheduler
		st.Running = true
		st.IsHealthy = true
		st.Image = cke.HyperkubeImage.Name()
		st.BuiltInParams = op.SchedulerParams()
	}
	return d
}

func (d testData) withKubelet(domain string, allowSwap bool) testData {
	for _, n := range d.Cluster.Nodes {
		st := &d.NodeStatus(n).Kubelet
		st.Running = true
		st.IsHealthy = true
		st.Image = cke.HyperkubeImage.Name()
		st.BuiltInParams = op.KubeletServiceParams(n)
		st.Domain = domain
		st.AllowSwap = allowSwap
	}
	return d
}

func (d testData) withProxy() testData {
	for _, v := range d.Status.NodeStatuses {
		st := &v.Proxy
		st.Running = true
		st.IsHealthy = true
		st.Image = cke.HyperkubeImage.Name()
		st.BuiltInParams = op.ProxyParams()
	}
	return d
}

func (d testData) withAllServices() testData {
	d.withRivers()
	d.withHealthyEtcd()
	d.withAPIServer(testServiceSubnet)
	d.withControllerManager(testClusterName, testServiceSubnet)
	d.withScheduler()
	d.withKubelet("", false)
	d.withProxy()
	return d
}

func (d testData) withK8sReady() testData {
	d.withAllServices()
	d.Status.Kubernetes.IsReady = true
	return d
}

func (d testData) withRBACResources() testData {
	d.withK8sReady()
	d.Status.Kubernetes.RBACRoleExists = true
	d.Status.Kubernetes.RBACRoleBindingExists = true
	return d
}

func (d testData) withEtcdEndpoints() testData {
	d.withRBACResources()
	d.Status.Kubernetes.EtcdEndpoints = &corev1.Endpoints{
		Subsets: []corev1.EndpointSubset{
			{
				Addresses: []corev1.EndpointAddress{
					{IP: "10.0.0.11"},
					{IP: "10.0.0.12"},
					{IP: "10.0.0.13"},
				},
				Ports: []corev1.EndpointPort{{Port: 2379}},
			},
		},
	}
	return d
}

func TestDecideOps(t *testing.T) {
	t.Parallel()

	cases := []struct {
		Name        string
		Input       testData
		ExpectedOps []string
	}{
		{
			Name:        "BootRivers",
			Input:       newData(),
			ExpectedOps: []string{"rivers-bootstrap"},
		},
		{
			Name:        "BootRivers2",
			Input:       newData().withHealthyEtcd(),
			ExpectedOps: []string{"rivers-bootstrap"},
		},
		{
			Name: "RestartRivers",
			Input: newData().withRivers().with(func(d testData) {
				d.NodeStatus(d.ControlPlane()[0]).Rivers.Image = ""
			}),
			ExpectedOps: []string{"rivers-restart"},
		},
		{
			Name: "RestartRivers2",
			Input: newData().withRivers().with(func(d testData) {
				d.NodeStatus(d.ControlPlane()[0]).Rivers.BuiltInParams.ExtraArguments = nil
			}),
			ExpectedOps: []string{"rivers-restart"},
		},
		{
			Name: "RestartRivers3",
			Input: newData().withRivers().with(func(d testData) {
				d.NodeStatus(d.ControlPlane()[0]).Rivers.ExtraParams.ExtraArguments = []string{"foo"}
			}),
			ExpectedOps: []string{"rivers-restart"},
		},
		{
			Name: "StartRestartRivers",
			Input: newData().withRivers().with(func(d testData) {
				d.NodeStatus(d.ControlPlane()[0]).Rivers.Image = ""
				d.NodeStatus(d.ControlPlane()[1]).Rivers.Running = false
			}),
			ExpectedOps: []string{"rivers-bootstrap", "rivers-restart"},
		},
		{
			Name:        "EtcdBootstrap",
			Input:       newData().withRivers(),
			ExpectedOps: []string{"etcd-bootstrap"},
		},
		{
			Name:        "EtcdStart",
			Input:       newData().withRivers().withStoppedEtcd(),
			ExpectedOps: []string{"etcd-start"},
		},
		{
			Name:        "WaitEtcd",
			Input:       newData().withRivers().withUnhealthyEtcd(),
			ExpectedOps: []string{"etcd-wait-cluster"},
		},
		{
			Name:  "BootK8s",
			Input: newData().withHealthyEtcd().withRivers(),
			ExpectedOps: []string{
				"kube-apiserver-bootstrap",
				"kube-controller-manager-bootstrap",
				"kube-proxy-bootstrap",
				"kube-scheduler-bootstrap",
				"kubelet-bootstrap",
			},
		},
		{
			Name:  "BootK8s2",
			Input: newData().withHealthyEtcd().withRivers().withAPIServer(testServiceSubnet),
			ExpectedOps: []string{
				"kube-controller-manager-bootstrap",
				"kube-proxy-bootstrap",
				"kube-scheduler-bootstrap",
				"kubelet-bootstrap",
			},
		},
		{
			Name:  "RestartAPIServer",
			Input: newData().withAllServices().withAPIServer("11.22.33.0/24"),
			ExpectedOps: []string{
				"kube-apiserver-restart",
			},
		},
		{
			Name: "RestartAPIServer2",
			Input: newData().withAllServices().with(func(d testData) {
				d.NodeStatus(d.ControlPlane()[0]).APIServer.Image = ""
			}),
			ExpectedOps: []string{
				"kube-apiserver-restart",
			},
		},
		{
			Name: "RestartAPIServer3",
			Input: newData().withAllServices().with(func(d testData) {
				d.NodeStatus(d.ControlPlane()[0]).APIServer.ExtraParams.ExtraArguments = []string{"foo"}
			}),
			ExpectedOps: []string{
				"kube-apiserver-restart",
			},
		},
		{
			Name:  "RestartControllerManager",
			Input: newData().withAllServices().withControllerManager("another", testServiceSubnet),
			ExpectedOps: []string{
				"kube-controller-manager-restart",
			},
		},
		{
			Name: "RestartControllerManager2",
			Input: newData().withAllServices().with(func(d testData) {
				d.NodeStatus(d.ControlPlane()[0]).ControllerManager.Image = ""
			}),
			ExpectedOps: []string{
				"kube-controller-manager-restart",
			},
		},
		{
			Name: "RestartControllerManager3",
			Input: newData().withAllServices().with(func(d testData) {
				d.NodeStatus(d.ControlPlane()[0]).ControllerManager.ExtraParams.ExtraArguments = []string{"foo"}
			}),
			ExpectedOps: []string{
				"kube-controller-manager-restart",
			},
		},
		{
			Name: "RestartScheduler",
			Input: newData().withAllServices().with(func(d testData) {
				d.NodeStatus(d.ControlPlane()[0]).Scheduler.BuiltInParams.ExtraArguments = []string{"foo"}
			}),
			ExpectedOps: []string{
				"kube-scheduler-restart",
			},
		},
		{
			Name: "RestartScheduler2",
			Input: newData().withAllServices().with(func(d testData) {
				d.NodeStatus(d.ControlPlane()[0]).Scheduler.Image = ""
			}),
			ExpectedOps: []string{
				"kube-scheduler-restart",
			},
		},
		{
			Name: "RestartScheduler3",
			Input: newData().withAllServices().with(func(d testData) {
				d.NodeStatus(d.ControlPlane()[0]).Scheduler.ExtraParams.ExtraArguments = []string{"foo"}
			}),
			ExpectedOps: []string{
				"kube-scheduler-restart",
			},
		},
		{
			Name:  "RestartKubelet",
			Input: newData().withAllServices().withKubelet("foo.local", false),
			ExpectedOps: []string{
				"kubelet-restart",
			},
		},
		{
			Name:  "RestartKubelet2",
			Input: newData().withAllServices().withKubelet("", true),
			ExpectedOps: []string{
				"kubelet-restart",
			},
		},
		{
			Name: "RestartKubelet3",
			Input: newData().withAllServices().with(func(d testData) {
				d.NodeStatus(d.Cluster.Nodes[0]).Kubelet.Image = ""
			}),
			ExpectedOps: []string{
				"kubelet-restart",
			},
		},
		{
			Name: "RestartKubelet4",
			Input: newData().withAllServices().with(func(d testData) {
				d.NodeStatus(d.Cluster.Nodes[0]).Kubelet.ExtraParams.ExtraArguments = []string{"foo"}
			}),
			ExpectedOps: []string{
				"kubelet-restart",
			},
		},
		{
			Name: "RestartProxy",
			Input: newData().withAllServices().with(func(d testData) {
				d.NodeStatus(d.Cluster.Nodes[0]).Proxy.BuiltInParams.ExtraArguments = []string{"foo"}
			}),
			ExpectedOps: []string{
				"kube-proxy-restart",
			},
		},
		{
			Name: "RestartProxy2",
			Input: newData().withAllServices().with(func(d testData) {
				d.NodeStatus(d.Cluster.Nodes[0]).Proxy.Image = ""
			}),
			ExpectedOps: []string{
				"kube-proxy-restart",
			},
		},
		{
			Name: "RestartProxy3",
			Input: newData().withAllServices().with(func(d testData) {
				d.NodeStatus(d.Cluster.Nodes[0]).Proxy.ExtraParams.ExtraArguments = []string{"foo"}
			}),
			ExpectedOps: []string{
				"kube-proxy-restart",
			},
		},
		{
			Name:        "WaiKube",
			Input:       newData().withAllServices(),
			ExpectedOps: []string{"wait-kubernetes"},
		},
		{
			Name:        "RBAC",
			Input:       newData().withK8sReady(),
			ExpectedOps: []string{"create-etcd-endpoints", "install-rbac-role"},
		},
		{
			Name:        "EtcdEndpointsCreate",
			Input:       newData().withRBACResources(),
			ExpectedOps: []string{"create-etcd-endpoints"},
		},
		{
			Name: "EtcdEndpointsUpdate1",
			Input: newData().withEtcdEndpoints().with(func(d testData) {
				d.Status.Kubernetes.EtcdEndpoints.Subsets = []corev1.EndpointSubset{}
			}),
			ExpectedOps: []string{"update-etcd-endpoints"},
		},
		{
			Name: "EtcdEndpointsUpdate2",
			Input: newData().withEtcdEndpoints().with(func(d testData) {
				d.Status.Kubernetes.EtcdEndpoints.Subsets[0].Ports = []corev1.EndpointPort{}
			}),
			ExpectedOps: []string{"update-etcd-endpoints"},
		},
		{
			Name: "EtcdEndpointsUpdate1",
			Input: newData().withEtcdEndpoints().with(func(d testData) {
				d.Status.Kubernetes.EtcdEndpoints.Subsets[0].Addresses = []corev1.EndpointAddress{}
			}),
			ExpectedOps: []string{"update-etcd-endpoints"},
		},
		{
			Name:        "AllGreen",
			Input:       newData().withEtcdEndpoints(),
			ExpectedOps: nil,
		},
		{
			Name: "EtcdRemoveNonClusterMember",
			Input: newData().withAllServices().with(func(d testData) {
				d.Status.Etcd.Members["10.0.0.100"] = &etcdserverpb.Member{Name: "10.0.0.100", ID: 3}
			}),
			ExpectedOps: []string{"etcd-remove-member"},
		},
		{
			Name: "EtcdDestroyNonCPMember",
			Input: newData().withAllServices().with(func(d testData) {
				d.Status.Etcd.Members["10.0.0.14"] = &etcdserverpb.Member{Name: "10.0.0.14", ID: 3}
			}),
			ExpectedOps: []string{"etcd-destroy-member"},
		},
		{
			Name: "EtcdReAdd",
			Input: newData().withAllServices().with(func(d testData) {
				d.Status.Etcd.Members["10.0.0.13"].Name = ""
				d.Status.Etcd.Members["10.0.0.13"].ID = 0
			}),
			ExpectedOps: []string{"etcd-add-member"},
		},
		{
			Name: "EtcdIsNotGood",
			Input: newData().withEtcdEndpoints().with(func(d testData) {
				// a node is to be added
				delete(d.Status.Etcd.Members, "10.0.0.13")
				delete(d.Status.Etcd.InSyncMembers, "10.0.0.13")
				// but the cluster is not good enough
				delete(d.Status.Etcd.InSyncMembers, "10.0.0.12")
			}),
			ExpectedOps: nil,
		},
		{
			Name: "EtcdAdd",
			Input: newData().withAllServices().with(func(d testData) {
				delete(d.Status.Etcd.Members, "10.0.0.13")
				delete(d.Status.Etcd.InSyncMembers, "10.0.0.13")
			}),
			ExpectedOps: []string{"etcd-add-member"},
		},
		{
			Name: "EtcdRemoveHealthyNonClusterMember",
			Input: newData().withAllServices().with(func(d testData) {
				d.Status.Etcd.Members["10.0.0.100"] = &etcdserverpb.Member{Name: "10.0.0.100", ID: 3}
				d.Status.Etcd.InSyncMembers["10.0.0.100"] = true
			}),
			ExpectedOps: []string{"etcd-remove-member"},
		},
		{
			Name: "EtcdDestroyHealthyNonCPMember",
			Input: newData().withAllServices().with(func(d testData) {
				d.Status.Etcd.Members["10.0.0.14"] = &etcdserverpb.Member{Name: "10.0.0.14", ID: 14}
				d.Status.Etcd.InSyncMembers["10.0.0.14"] = true
			}),
			ExpectedOps: []string{"etcd-destroy-member"},
		},
		{
			Name: "EtcdRestart",
			Input: newData().withAllServices().with(func(d testData) {
				d.NodeStatus(d.ControlPlane()[0]).Etcd.Image = ""
			}),
			ExpectedOps: []string{"etcd-restart"},
		},
		{
			Name: "Clean",
			Input: newData().withEtcdEndpoints().with(func(d testData) {
				st := d.Status.NodeStatuses["10.0.0.14"]
				st.Etcd.Running = true
				st.Etcd.HasData = true
				st.APIServer.Running = true
				st.ControllerManager.Running = true
				st.Scheduler.Running = true
			}),
			ExpectedOps: []string{
				"stop-etcd",
				"stop-kube-apiserver",
				"stop-kube-controller-manager",
				"stop-kube-scheduler",
			},
		},
	}

	for _, c := range cases {
		ops := DecideOps(c.Input.Cluster, c.Input.Status)
		if len(ops) == 0 && len(c.ExpectedOps) == 0 {
			continue
		}
		opNames := make([]string, len(ops))
		for i, op := range ops {
			opNames[i] = op.Name()
		}
		sort.Strings(opNames)
		if !reflect.DeepEqual(opNames, c.ExpectedOps) {
			t.Errorf("[%s] op names mismatch: %s != %s", c.Name, opNames, c.ExpectedOps)
		}
	OUT:
		for _, op := range ops {
			for i := 0; i < 100; i++ {
				commander := op.NextCommand()
				if commander == nil {
					continue OUT
				}
			}
			t.Fatalf("[%s] Operator.NextCommand() never finished: %s", c.Name, op.Name())
		}
	}
}
