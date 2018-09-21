package cke

import (
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

	// for kubelet
	CurrentKubeletDomain    string
	CurrentKubeletAllowSwap bool

	// for RBAC
	CurrentRBACRoleExists        bool
	CurrentRBACRoleBindingExists bool
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
	return &Cluster{Name: "test", Nodes: nodes, Options: options, ServiceSubnet: "10.20.30.40/31"}
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
			ControllerManager: KubeComponentStatus{ServiceStatus{BuiltInParams: ControllerManagerParams("test", "10.20.30.40/31"), ExtraParams: ServiceParams{ExtraArguments: c.CurrentControllerManagerArgs}}, false},
			Scheduler:         KubeComponentStatus{ServiceStatus{BuiltInParams: SchedulerParams(), ExtraParams: ServiceParams{ExtraArguments: c.CurrentSchedulerArgs}}, false},
			Proxy:             KubeComponentStatus{ServiceStatus{BuiltInParams: ProxyParams(), ExtraParams: ServiceParams{ExtraArguments: c.CurrentProxyArgs}}, false},
			Kubelet: KubeletStatus{
				ServiceStatus{
					BuiltInParams: KubeletServiceParams(&Node{Address: addr}),
					ExtraParams:   ServiceParams{ExtraArguments: c.CurrentKubeletArgs},
				},
				false,
				c.CurrentKubeletDomain,
				c.CurrentKubeletAllowSwap,
			},
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

	k8sStatus := KubernetesClusterStatus{
		RBACRoleExists:        c.CurrentRBACRoleExists,
		RBACRoleBindingExists: c.CurrentRBACRoleBindingExists,
	}

	return &ClusterStatus{NodeStatuses: nodeStatus, Kubernetes: k8sStatus}
}

func testKubernetesDecideToDo(t *testing.T) {
	cpNodes := []string{"10.0.0.11", "10.0.0.12", "10.0.0.13"}
	nonCpNodes := []string{"10.0.0.14", "10.0.0.15", "10.0.0.16"}
	allNodes := append(cpNodes, nonCpNodes...)

	cases := []struct {
		Name     string
		Input    KubernetesTestConfiguration
		Operator string
	}{
		{
			Name: "Bootstrap Rivers on all nodes",
			Input: KubernetesTestConfiguration{
				CpNodes: cpNodes, NonCpNodes: nonCpNodes,
			},
			Operator: "rivers-bootstrap",
		},
		{
			Name: "Bootstrap kube-apiserver on control planes",
			Input: KubernetesTestConfiguration{
				CpNodes: cpNodes, NonCpNodes: nonCpNodes, Rivers: allNodes,
			},
			Operator: "kube-apiserver-bootstrap",
		},
		{
			Name: "Bootstrap kube-controller-manager on control planes",
			Input: KubernetesTestConfiguration{
				CpNodes: cpNodes, NonCpNodes: nonCpNodes, Rivers: allNodes, APIServers: cpNodes,
			},
			Operator: "kube-controller-manager-bootstrap",
		},
		{
			Name: "Bootstrap kube-scheduler on control planes",
			Input: KubernetesTestConfiguration{
				CpNodes: cpNodes, NonCpNodes: nonCpNodes, Rivers: allNodes, APIServers: cpNodes, ControllerManagers: cpNodes,
			},
			Operator: "kube-scheduler-bootstrap",
		},
		{
			Name: "Bootstrap kubelet",
			Input: KubernetesTestConfiguration{
				CpNodes: cpNodes, NonCpNodes: nonCpNodes,
				Rivers: allNodes, APIServers: cpNodes, ControllerManagers: cpNodes, Schedulers: cpNodes,
			},
			Operator: "kubelet-bootstrap",
		},
		{
			Name: "Bootstrap kube-proxy",
			Input: KubernetesTestConfiguration{
				CpNodes: cpNodes, NonCpNodes: nonCpNodes,
				Rivers: allNodes, APIServers: cpNodes, ControllerManagers: cpNodes, Schedulers: cpNodes, Kubelets: allNodes,
			},
			Operator: "kube-proxy-bootstrap",
		},
		{
			Name: "Stop kube-apiserver",
			Input: KubernetesTestConfiguration{
				CpNodes: cpNodes, NonCpNodes: nonCpNodes,
				Rivers:             allNodes,
				APIServers:         append(cpNodes, "10.0.0.14", "10.0.0.15"),
				ControllerManagers: cpNodes,
				Schedulers:         cpNodes,
			},
			Operator: "container-stop",
		},
		{
			Name: "Stop kube-controller-manager",
			Input: KubernetesTestConfiguration{
				CpNodes: cpNodes, NonCpNodes: nonCpNodes,
				Rivers:             allNodes,
				APIServers:         cpNodes,
				ControllerManagers: append(cpNodes, "10.0.0.14", "10.0.0.15"),
				Schedulers:         cpNodes,
			},
			Operator: "container-stop",
		},
		{
			Name: "Stop kube-scheduler",
			Input: KubernetesTestConfiguration{
				CpNodes: cpNodes, NonCpNodes: nonCpNodes,
				Rivers:             allNodes,
				APIServers:         cpNodes,
				ControllerManagers: cpNodes,
				Schedulers:         append(cpNodes, "10.0.0.14", "10.0.0.15"),
			},
			Operator: "container-stop",
		},
		{
			Name: "Install ClusterRole for RBAC",
			Input: KubernetesTestConfiguration{
				CpNodes: cpNodes, NonCpNodes: nonCpNodes,
				Rivers:             allNodes,
				APIServers:         cpNodes,
				ControllerManagers: cpNodes,
				Schedulers:         cpNodes,
				Kubelets:           allNodes,
				Proxies:            allNodes,
			},
			Operator: "install-rbac-role",
		},
		{
			Name: "Do nothing if the cluster is stable",
			Input: KubernetesTestConfiguration{
				CpNodes: cpNodes, NonCpNodes: nonCpNodes,
				Rivers:                       allNodes,
				APIServers:                   cpNodes,
				ControllerManagers:           cpNodes,
				Schedulers:                   cpNodes,
				Kubelets:                     allNodes,
				Proxies:                      allNodes,
				CurrentRBACRoleExists:        true,
				CurrentRBACRoleBindingExists: true,
			},
			Operator: "",
		},
		{
			Name: "Restart kube-apiserver when params are updated",
			Input: KubernetesTestConfiguration{
				CpNodes: cpNodes, NonCpNodes: nonCpNodes,
				Rivers:             allNodes,
				APIServers:         cpNodes,
				ControllerManagers: cpNodes,
				Schedulers:         cpNodes,
				Kubelets:           allNodes,
				Proxies:            allNodes,
				APIServerArgs:      []string{"--apiserver-count=999"},
			},
			Operator: "kube-apiserver-restart",
		},
		{
			Name: "Restart kube-controller-manager when params are updated",
			Input: KubernetesTestConfiguration{
				CpNodes: cpNodes, NonCpNodes: nonCpNodes,
				Rivers:                allNodes,
				APIServers:            cpNodes,
				ControllerManagers:    cpNodes,
				Schedulers:            cpNodes,
				Kubelets:              allNodes,
				Proxies:               allNodes,
				ControllerManagerArgs: []string{"--contention-profiling=0.99"},
			},
			Operator: "kube-controller-manager-restart",
		},
		{
			Name: "Restart kube-scheduler when params are updated",
			Input: KubernetesTestConfiguration{
				CpNodes: cpNodes, NonCpNodes: nonCpNodes,
				Rivers:             allNodes,
				APIServers:         cpNodes,
				ControllerManagers: cpNodes,
				Schedulers:         cpNodes,
				Kubelets:           allNodes,
				Proxies:            allNodes,
				SchedulerArgs:      []string{"--leader-elect-retry-period duration=2s"},
			},
			Operator: "kube-scheduler-restart",
		},
		{
			Name: "Restart kubelet when params are updated",
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
			Operator: "kubelet-restart",
		},
		{
			Name: "Restart kube-proxy when params are updated",
			Input: KubernetesTestConfiguration{
				CpNodes: cpNodes, NonCpNodes: nonCpNodes,
				Rivers:             allNodes,
				APIServers:         cpNodes,
				ControllerManagers: cpNodes,
				Schedulers:         cpNodes,
				Kubelets:           allNodes,
				Proxies:            allNodes,
				ProxyArgs:          []string{"--healthz-port=11223"},
			},
			Operator: "kube-proxy-restart",
		},
	}

	for _, c := range cases {
		op := kubernetesDecideToDo(c.Input.Cluster(), c.Input.ClusterState())
		if op == nil && len(c.Operator) == 0 {
			continue
		} else if op == nil {
			t.Fatal("op == nil")
		}
		if op.Name() != c.Operator {
			t.Errorf("[%s] operator name mismatch: %s != %s", c.Name, op.Name(), c.Operator)
		}
	}
}

func TestKubernetesStrategy(t *testing.T) {
	t.Run("KubernetesDecideToDo", testKubernetesDecideToDo)
}
