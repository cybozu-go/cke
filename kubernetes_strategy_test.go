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

	// Declared command-line arguments in Cluster or CKE
	RiverArgs             []string
	APIServerArgs         []string
	ControllerManagerArgs []string
	SchedulerArgs         []string
	KubeletArgs           []string
	ProxyArgs             []string

	// Running in ClusterStatus
	Rivers             []string
	APIServers         []string
	ControllerManagers []string
	Schedulers         []string
	Kubelets           []string
	Proxies            []string

	// Current command-line arguments on the containers
	CurrentRiverArgs             []string
	CurrentAPIServerArgs         []string
	CurrentControllerManagerArgs []string
	CurrentSchedulerArgs         []string
	CurrentKubeletArgs           []string
	CurrentProxyArgs             []string
}

func (c *KubernetesTestConfiguration) Cluster() *Cluster {
	var nodes = make([]*Node, len(c.CpNodes)+len(c.NonCpNodes))
	for i, n := range c.CpNodes {
		nodes[i] = &Node{Address: n, ControlPlane: true}
	}
	for i, n := range c.NonCpNodes {
		nodes[i+len(c.CpNodes)] = &Node{Address: n}
	}
	var options Options
	options.Rivers.ExtraArguments = c.RiverArgs
	options.APIServer.ExtraArguments = c.APIServerArgs
	options.ControllerManager.ExtraArguments = c.ControllerManagerArgs
	options.Scheduler.ExtraArguments = c.SchedulerArgs
	options.Kubelet.ExtraArguments = c.KubeletArgs
	options.Proxy.ExtraArguments = c.ProxyArgs
	return &Cluster{Nodes: nodes, Options: options, ServiceSubnet: "10.20.30.40/31"}
}

func (c *KubernetesTestConfiguration) ClusterState() *ClusterStatus {

	var cps = make([]*Node, len(c.CpNodes))
	for i, n := range c.CpNodes {
		cps[i] = &Node{Address: n, ControlPlane: true}
	}

	nodeStatus := make(map[string]*NodeStatus)
	for _, addr := range append(c.CpNodes, c.NonCpNodes...) {
		nodeStatus[addr] = &NodeStatus{
			Rivers:            ServiceStatus{BuiltInParams: RiversParams(cps), ExtraParams: ServiceParams{ExtraArguments: c.CurrentRiverArgs}},
			APIServer:         KubeComponentStatus{ServiceStatus{BuiltInParams: APIServerParams(cps, addr, "10.20.30.40/31"), ExtraParams: ServiceParams{ExtraArguments: c.CurrentAPIServerArgs}}, false},
			ControllerManager: KubeComponentStatus{ServiceStatus{BuiltInParams: ControllerManagerParams(), ExtraParams: ServiceParams{ExtraArguments: c.CurrentControllerManagerArgs}}, false},
			Scheduler:         KubeComponentStatus{ServiceStatus{BuiltInParams: SchedulerParams(), ExtraParams: ServiceParams{ExtraArguments: c.CurrentSchedulerArgs}}, false},
			Proxy:             KubeComponentStatus{ServiceStatus{BuiltInParams: ProxyParams(), ExtraParams: ServiceParams{ExtraArguments: c.CurrentProxyArgs}}, false},
			Kubelet:           KubeComponentStatus{ServiceStatus{BuiltInParams: KubeletServiceParams(&Node{Address: addr}), ExtraParams: ServiceParams{ExtraArguments: c.CurrentKubeletArgs}}, false},
		}
	}
	for _, addr := range c.Rivers {
		nodeStatus[addr].Rivers.Running = true
	}
	for _, addr := range c.APIServers {
		nodeStatus[addr].APIServer.IsHealthy = true
		nodeStatus[addr].APIServer.Running = true
	}
	for _, addr := range c.ControllerManagers {
		nodeStatus[addr].ControllerManager.IsHealthy = true
		nodeStatus[addr].ControllerManager.Running = true
	}
	for _, addr := range c.Schedulers {
		nodeStatus[addr].Scheduler.IsHealthy = true
		nodeStatus[addr].Scheduler.Running = true
	}
	for _, addr := range c.Kubelets {
		nodeStatus[addr].Kubelet.IsHealthy = true
		nodeStatus[addr].Kubelet.Running = true
	}
	for _, addr := range c.Proxies {
		nodeStatus[addr].Proxy.IsHealthy = true
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
				{"image-pull", "hyperkube", Image("hyperkube")},
				{"mkdir", "/var/log/kubernetes/apiserver", ""},
				{"mkdir", "/var/log/kubernetes/controller-manager", ""},
				{"mkdir", "/var/log/kubernetes/scheduler", ""},
				{"issue-apiserver-certificates", "10.0.0.11,10.0.0.12,10.0.0.13", ""},
				{"setup-apiserver-certificates", "10.0.0.11,10.0.0.12,10.0.0.13", ""},
				{"make-controller-manager-kubeconfig", "10.0.0.11,10.0.0.12,10.0.0.13", ""},
				{"make-scheduler-kubeconfig", "10.0.0.11,10.0.0.12,10.0.0.13", ""},
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
				{"image-pull", "hyperkube", Image("hyperkube")},
				{"mkdir", "/var/log/kubernetes/kubelet", ""},
				{"mkdir", "/var/log/kubernetes/proxy", ""},
				{"make-kubelet-kubeconfig", strings.Join(allNodes, ","), ""},
				{"make-proxy-kubeconfig", strings.Join(allNodes, ","), ""},
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
		{
			Name: "Do notions if the cluster is stable",
			Input: KubernetesTestConfiguration{
				CpNodes: cpNodes, NonCpNodes: nonCpNodes,
				Rivers:             allNodes,
				APIServers:         cpNodes,
				ControllerManagers: cpNodes,
				Schedulers:         cpNodes,
				Kubelets:           allNodes,
				Proxies:            allNodes,
			},
			Commands: []Command{},
		},
		{
			Name: "Restart kubernetes control planes when params are updated",
			Input: KubernetesTestConfiguration{
				CpNodes: cpNodes, NonCpNodes: nonCpNodes,
				Rivers:                allNodes,
				APIServers:            cpNodes,
				ControllerManagers:    cpNodes,
				Schedulers:            cpNodes,
				Kubelets:              allNodes,
				Proxies:               allNodes,
				APIServerArgs:         []string{"--apiserver-count=999"},
				ControllerManagerArgs: []string{"--contention-profiling=0.99"},
				SchedulerArgs:         []string{"--leader-elect-retry-period duration=2s"},
			},
			Commands: func() []Command {
				cmds := []Command{
					{"image-pull", "rivers", Image("rivers")},
					{"image-pull", "hyperkube", Image("hyperkube")},
				}
				for _, n := range cpNodes {
					cmds = append(cmds,
						Command{Name: "stop-containers", Target: n, Detail: "kube-apiserver"},
						Command{Name: "run-container", Target: n, Detail: "kube-apiserver"})
				}
				for _, n := range cpNodes {
					cmds = append(cmds,
						Command{Name: "stop-containers", Target: n, Detail: "kube-controller-manager"},
						Command{Name: "run-container", Target: n, Detail: "kube-controller-manager"})
				}
				for _, n := range cpNodes {
					cmds = append(cmds,
						Command{Name: "stop-containers", Target: n, Detail: "kube-scheduler"},
						Command{Name: "run-container", Target: n, Detail: "kube-scheduler"})
				}
				return cmds
			}(),
		},
		{
			Name: "Restart kubernetes workers when params are updated",
			Input: KubernetesTestConfiguration{
				CpNodes: cpNodes, NonCpNodes: nonCpNodes,
				Rivers:             allNodes,
				APIServers:         cpNodes,
				ControllerManagers: cpNodes,
				Schedulers:         cpNodes,
				Kubelets:           allNodes,
				Proxies:            allNodes,
				KubeletArgs:        []string{"--cpu-cfs-quota=true"},
			},
			Commands: []Command{
				{"image-pull", "hyperkube", Image("hyperkube")},
				{Name: "make-kubelet-kubeconfig", Target: strings.Join(allNodes, ","), Detail: ""},
				{Name: "stop-containers", Target: strings.Join(allNodes, ","), Detail: "kubelet"},
				{Name: "run-container", Target: strings.Join(allNodes, ","), Detail: "kubelet"},
			},
		},
	}

	for _, c := range cases {
		op := kubernetesDecideToDo(c.Input.Cluster(), c.Input.ClusterState())
		if op == nil && len(c.Commands) == 0 {
			continue
		} else if op == nil {
			t.Fatal("op == nil")
		}
		cmds := opCommands(op)
		if len(c.Commands) != len(cmds) {
			t.Errorf("[%s](%s) commands length mismatch. expected length: %d, actual: %d", c.Name, op.Name(), len(c.Commands), len(cmds))
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
