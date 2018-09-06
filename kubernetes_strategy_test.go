package cke

import (
	"reflect"
	"strings"
	"testing"
)

type KubernetesTestConfiguration struct {
	// Cluster
	CpNodes    []string
	NonCpNodes []string

	// Running in ClusterStatus
	Rivers             []string
	APIServers         []string
	ControllerManagers []string
	Schedulers         []string
	Kubelets           []string
	Proxies            []string
}

func (c *KubernetesTestConfiguration) Cluster() *Cluster {
	var nodes = make([]*Node, len(c.CpNodes)+len(c.NonCpNodes))
	for i, n := range c.CpNodes {
		nodes[i] = &Node{Address: n, ControlPlane: true}
	}
	for i, n := range c.NonCpNodes {
		nodes[i+len(c.CpNodes)] = &Node{Address: n}
	}
	return &Cluster{Nodes: nodes}
}

func (c *KubernetesTestConfiguration) ClusterState() *ClusterStatus {
	nodeStatus := make(map[string]*NodeStatus)
	for _, addr := range append(c.CpNodes, c.NonCpNodes...) {
		nodeStatus[addr] = &NodeStatus{}
	}
	for _, addr := range c.Rivers {
		nodeStatus[addr].Rivers.Running = true
	}
	for _, addr := range c.APIServers {
		nodeStatus[addr].APIServer.Running = true
	}
	for _, addr := range c.ControllerManagers {
		nodeStatus[addr].ControllerManager.Running = true
	}
	for _, addr := range c.Schedulers {
		nodeStatus[addr].Scheduler.Running = true
	}
	for _, addr := range c.Kubelets {
		nodeStatus[addr].Kubelet.Running = true
	}
	for _, addr := range c.Proxies {
		nodeStatus[addr].Proxy.Running = true
	}

	return &ClusterStatus{NodeStatuses: nodeStatus}
}

func testKubernetesDecideToDo(t *testing.T) {
	cpNodes := []string{"10.0.0.11", "10.0.0.12", "10.0.0.13"}
	nonCpNodes := []string{"10.0.0.14", "10.0.0.15", "10.0.0.16"}
	allNodes := append(cpNodes, nonCpNodes...)

	cases := []struct {
		Name     string
		Input    KubernetesTestConfiguration
		Commands []Command
	}{
		{
			Name: "Bootstrap Rivers on all nodes",
			Input: KubernetesTestConfiguration{
				CpNodes: cpNodes, NonCpNodes: nonCpNodes,
			},
			Commands: []Command{
				{"image-pull", "rivers", Image("rivers")},
				{"mkdir", "/var/log/rivers", ""},
				{"run-container", strings.Join(allNodes, ","), "rivers"},
			},
		},
		{
			Name: "Bootstrap Control Planes",
			Input: KubernetesTestConfiguration{
				CpNodes: cpNodes, NonCpNodes: nonCpNodes, Rivers: allNodes,
			},
			Commands: []Command{
				{"image-pull", "kube-apiserver", Image("kube-apiserver")},
				{"mkdir", "/var/log/kubernetes/apiserver", ""},
				{"mkdir", "/var/log/kubernetes/controller-manager", ""},
				{"mkdir", "/var/log/kubernetes/scheduler", ""},
				{"make-file", "/etc/kubernetes/controller-manager/kubeconfig", ""},
				{"make-file", "/etc/kubernetes/scheduler/kubeconfig", ""},
				{"issue-apiserver-certificates", "10.0.0.11,10.0.0.12,10.0.0.13", ""},
				{"run-container", "10.0.0.11", "kube-apiserver"},
				{"run-container", "10.0.0.12", "kube-apiserver"},
				{"run-container", "10.0.0.13", "kube-apiserver"},
				{"run-container", "10.0.0.11,10.0.0.12,10.0.0.13", "kube-scheduler"},
				{"run-container", "10.0.0.11,10.0.0.12,10.0.0.13", "kube-controller-manager"},
			},
		},
		{
			Name: "Bootstrap kubernetes workers",
			Input: KubernetesTestConfiguration{
				CpNodes: cpNodes, NonCpNodes: nonCpNodes,
				Rivers: allNodes, APIServers: cpNodes, ControllerManagers: cpNodes, Schedulers: cpNodes,
			},
			Commands: []Command{
				{"image-pull", "kube-proxy", Image("kube-proxy")},
				{"mkdir", "/var/log/kubernetes/kubelet", ""},
				{"mkdir", "/var/log/kubernetes/proxy", ""},
				{"make-file", "/etc/kubernetes/kubelet/kubeconfig", ""},
				{"make-file", "/etc/kubernetes/proxy/kubeconfig", ""},
				{"volume-create", strings.Join(allNodes, ","), "dockershim"},
				{"run-container", strings.Join(allNodes, ","), "kubelet"},
				{"run-container", strings.Join(allNodes, ","), "kube-proxy"},
			},
		},
		{
			Name: "Stop Control Planes",
			Input: KubernetesTestConfiguration{
				CpNodes: cpNodes, NonCpNodes: nonCpNodes,
				Rivers:             allNodes,
				APIServers:         append(cpNodes, "10.0.0.14", "10.0.0.15"),
				ControllerManagers: append(cpNodes, "10.0.0.14", "10.0.0.15"),
				Schedulers:         append(cpNodes, "10.0.0.14", "10.0.0.15"),
			},
			Commands: []Command{
				{"stop-containers", "10.0.0.14,10.0.0.15", "kube-apiserver"},
				{"stop-containers", "10.0.0.14,10.0.0.15", "kube-scheduler"},
				{"stop-containers", "10.0.0.14,10.0.0.15", "kube-controller-manager"},
			},
		},
	}

	for _, c := range cases {
		op := kubernetesDecideToDo(c.Input.Cluster(), c.Input.ClusterState())
		if op == nil {
			t.Fatal("op == nil")
		}
		cmds := opCommands(op)
		if len(c.Commands) != len(cmds) {
			t.Errorf("[%s] commands length mismatch. expected length: %d, actual: %d", c.Name, len(c.Commands), len(cmds))
			continue
		}
		for i, res := range cmds {
			cmd := c.Commands[i]
			if !reflect.DeepEqual(cmd, res) {
				t.Errorf("[%s] %#v != %#v", c.Name, cmd, res)
			}
		}
	}
}

func TestKubernetesStrategy(t *testing.T) {
	t.Run("KubernetesDecideToDo", testKubernetesDecideToDo)
}
