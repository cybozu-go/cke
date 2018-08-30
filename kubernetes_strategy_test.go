package cke

import (
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
			Name: "Bootstrap Rivers",
			Input: KubernetesTestConfiguration{
				CpNodes: cpNodes, NonCpNodes: nonCpNodes,
			},
			Commands: []Command{
				{"image-pull", "rivers", ""},
				{"mkdir", "/var/log/rivers", ""},
				{"run-container", strings.Join(allNodes, ","), ""},
			},
		},
		{
			Name: "Bootstrap APIServers",
			Input: KubernetesTestConfiguration{
				CpNodes: cpNodes, NonCpNodes: nonCpNodes,
				Rivers: allNodes,
			},
			Commands: []Command{
				{"image-pull", "kube-apiserver", ""},
				{"mkdir", "/var/log/kubernetes/apiserver", ""},
				{"run-container", "10.0.0.11", ""},
				{"run-container", "10.0.0.12", ""},
				{"run-container", "10.0.0.13", ""},
			},
		},
		{
			Name: "Bootstrap ControllerManagers",
			Input: KubernetesTestConfiguration{
				CpNodes: cpNodes, NonCpNodes: nonCpNodes,
				Rivers: allNodes, APIServers: cpNodes,
			},
			Commands: []Command{
				{"make-file", "/etc/kubernetes/controller-manager/kubeconfig", ""},
				{"image-pull", "kube-controller-manager", ""},
				{"mkdir", "/var/log/kubernetes/controller-manager", ""},
				{"run-container", strings.Join(cpNodes, ","), ""},
			},
		},
		{
			Name: "Bootstrap Scheduler",
			Input: KubernetesTestConfiguration{
				CpNodes: cpNodes, NonCpNodes: nonCpNodes,
				Rivers: allNodes, APIServers: cpNodes, ControllerManagers: cpNodes,
			},
			Commands: []Command{
				{"make-file", "/etc/kubernetes/scheduler/kubeconfig", ""},
				{"image-pull", "kube-scheduler", ""},
				{"mkdir", "/var/log/kubernetes/scheduler", ""},
				{"run-container", strings.Join(cpNodes, ","), ""},
			},
		},
		{
			Name: "Bootstrap Kubelet",
			Input: KubernetesTestConfiguration{
				CpNodes: cpNodes, NonCpNodes: nonCpNodes,
				Rivers: allNodes, APIServers: cpNodes, ControllerManagers: cpNodes, Schedulers: cpNodes,
			},
			Commands: []Command{
				{"make-file", "/etc/kubernetes/kubelet/kubeconfig", ""},
				{"image-pull", "kubelet", ""},
				{"image-pull", "pause", ""},
				{"mkdir", "/var/log/kubernetes/kubelet", ""},
				{"volume-create", strings.Join(allNodes, ","), ""},
				{"run-container", strings.Join(allNodes, ","), ""},
			},
		},
		{
			Name: "Bootstrap Proxy",
			Input: KubernetesTestConfiguration{
				CpNodes: cpNodes, NonCpNodes: nonCpNodes,
				Rivers: allNodes, APIServers: cpNodes, ControllerManagers: cpNodes, Schedulers: cpNodes, Kubelets: allNodes,
			},
			Commands: []Command{
				{"make-file", "/etc/kubernetes/proxy/kubeconfig", ""},
				{"image-pull", "kube-proxy", ""},
				{"mkdir", "/var/log/kubernetes/proxy", ""},
				{"run-container", strings.Join(allNodes, ","), ""},
			},
		},
		{
			Name: "Stop APIServers",
			Input: KubernetesTestConfiguration{
				CpNodes: cpNodes, NonCpNodes: nonCpNodes,
				Rivers: allNodes, APIServers: append(cpNodes, "10.0.0.14", "10.0.0.15"),
			},
			Commands: []Command{
				{"stop-container", "10.0.0.14", ""},
				{"stop-container", "10.0.0.15", ""},
			},
		},
		{
			Name: "Stop Controller Managers",
			Input: KubernetesTestConfiguration{
				CpNodes: cpNodes, NonCpNodes: nonCpNodes,
				Rivers: allNodes, APIServers: cpNodes,
				ControllerManagers: append(cpNodes, "10.0.0.14", "10.0.0.15"),
			},
			Commands: []Command{
				{"rm", "/etc/kubernetes/controller-manager/kubeconfig", ""},
				{"stop-container", "10.0.0.14", ""},
				{"stop-container", "10.0.0.15", ""},
			},
		},
		{
			Name: "Stop Schedulers",
			Input: KubernetesTestConfiguration{
				CpNodes: cpNodes, NonCpNodes: nonCpNodes,
				Rivers: allNodes, APIServers: cpNodes, ControllerManagers: cpNodes,
				Schedulers: append(cpNodes, "10.0.0.14", "10.0.0.15"),
			},
			Commands: []Command{
				{"rm", "/etc/kubernetes/scheduler/kubeconfig", ""},
				{"stop-container", "10.0.0.14", ""},
				{"stop-container", "10.0.0.15", ""},
			},
		},
		{
			Name: "Stop the container because its command arguments are different from expected ones",
			Input: KubernetesTestConfiguration{
				CpNodes: cpNodes, NonCpNodes: nonCpNodes,
				Rivers: allNodes, APIServers: cpNodes, ControllerManagers: cpNodes, Schedulers: cpNodes, Kubelets: allNodes, Proxies: allNodes,
			},
			Commands: []Command{
				{"stop-container", "10.0.0.11", ""},
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
			if cmd.Name != res.Name {
				t.Errorf("[%s] command name mismatch: %s != %s", c.Name, cmd.Name, res.Name)
			}
			if cmd.Target != res.Target {
				t.Errorf("[%s] command '%s' target mismatch: %s != %s", c.Name, cmd.Name, cmd.Target, res.Target)
			}
		}
	}
}

func TestKubernetesStrategy(t *testing.T) {
	t.Run("KubernetesDecideToDo", testKubernetesDecideToDo)
}
