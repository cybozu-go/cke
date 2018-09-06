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
			Name: "Bootstrap Control Planes",
			Input: KubernetesTestConfiguration{
				CpNodes: cpNodes, NonCpNodes: nonCpNodes, Rivers: allNodes,
			},
			Commands: []Command{
				{"image-pull", "kube-apiserver", Image("kube-apiserver")},
				{"mkdir", "/var/log/kubernetes/apiserver", ""},
				{"mkdir", "/var/log/kubernetes/controller-manager", ""},
				{"mkdir", "/var/log/kubernetes/scheduler", ""},
				{"issue-apiserver-certificates", "10.0.0.11,10.0.0.12,10.0.0.13", ""},
				{"run-container", "10.0.0.11", "kube-apiserver"},
				{"run-container", "10.0.0.12", "kube-apiserver"},
				{"run-container", "10.0.0.13", "kube-apiserver"},
				{"run-container", "10.0.0.11,10.0.0.12,10.0.0.13", "kube-scheduler"},
				{"run-container", "10.0.0.11,10.0.0.12,10.0.0.13", "kube-controller-manager"},
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
				{"image-pull", "kubelet", Image("kubelet")},
				{"image-pull", "pause", Image("pause")},
				{"mkdir", "/var/log/kubernetes/kubelet", ""},
				{"volume-create", strings.Join(allNodes, ","), "dockershim"},
				{"run-container", strings.Join(allNodes, ","), "kubelet"},
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
				{"image-pull", "kube-proxy", Image("kube-proxy")},
				{"mkdir", "/var/log/kubernetes/proxy", ""},
				{"run-container", strings.Join(allNodes, ","), "kube-proxy"},
			},
		},
		{
			Name: "Stop Controll Planes",
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
		{
			Name: "Stop rivers because its command arguments are different from expected ones",
			Input: KubernetesTestConfiguration{
				CpNodes: cpNodes, NonCpNodes: nonCpNodes,
				Rivers: allNodes, APIServers: cpNodes, ControllerManagers: cpNodes, Schedulers: cpNodes, Kubelets: allNodes, Proxies: allNodes,
			},
			Commands: []Command{
				{"kill-container", "10.0.0.11", "rivers"},
				{"kill-container", "10.0.0.12", "rivers"},
				{"kill-container", "10.0.0.13", "rivers"},
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
			if cmd.Detail != res.Detail {
				t.Errorf("[%s] command '%s' detail mismatch: %s != %s", c.Name, cmd.Name, cmd.Detail, res.Detail)
			}
		}
	}
}

func TestKubernetesStrategy(t *testing.T) {
	t.Run("KubernetesDecideToDo", testKubernetesDecideToDo)
}
