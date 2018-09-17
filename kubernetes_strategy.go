package cke

func kubernetesDecideToDo(c *Cluster, cs *ClusterStatus) Operator {
	var cpNodes []*Node
	var nonCpNodes []*Node
	for _, n := range c.Nodes {
		if n.ControlPlane {
			cpNodes = append(cpNodes, n)
		} else {
			nonCpNodes = append(nonCpNodes, n)
		}
	}

	// Run Rivers on all nodes
	rivers := filterNodes(c.Nodes, func(n *Node) bool {
		return !cs.NodeStatuses[n.Address].Rivers.Running
	})
	if len(rivers) > 0 {
		return RiversBootOp(rivers, cpNodes, c.Options.Rivers)
	}

	// Run kubernetes control planes on control-plane nodes
	apiservers := filterNodes(cpNodes, func(n *Node) bool {
		return !cs.NodeStatuses[n.Address].APIServer.Running
	})
	controllerManagers := filterNodes(cpNodes, func(n *Node) bool {
		return !cs.NodeStatuses[n.Address].ControllerManager.Running
	})
	schedulers := filterNodes(cpNodes, func(n *Node) bool {
		return !cs.NodeStatuses[n.Address].Scheduler.Running
	})
	if len(apiservers)+len(controllerManagers)+len(schedulers) > 0 {
		return KubeCPBootOp(cpNodes, apiservers, controllerManagers, schedulers, c.Name, c.ServiceSubnet, c.Options)
	}

	// Stop kubernetes control planes on non-control-plane nodes
	apiservers = filterNodes(nonCpNodes, func(n *Node) bool {
		return cs.NodeStatuses[n.Address].APIServer.Running
	})
	controllerManagers = filterNodes(nonCpNodes, func(n *Node) bool {
		return cs.NodeStatuses[n.Address].ControllerManager.Running
	})
	schedulers = filterNodes(nonCpNodes, func(n *Node) bool {
		return cs.NodeStatuses[n.Address].Scheduler.Running
	})
	if len(apiservers)+len(controllerManagers)+len(schedulers) > 0 {
		return KubeCPStopOp(apiservers, controllerManagers, schedulers)
	}

	// Run kubelet and kube-proxy on all nodes
	kubelets := filterNodes(c.Nodes, func(n *Node) bool {
		return !cs.NodeStatuses[n.Address].Kubelet.Running
	})
	proxies := filterNodes(c.Nodes, func(n *Node) bool {
		return !cs.NodeStatuses[n.Address].Proxy.Running
	})
	if len(rivers)+len(kubelets)+len(proxies) > 0 {
		return KubeWorkerBootOp(cpNodes, kubelets, proxies, c.Options)
	}

	// Check diff of command options
	return kubernetesOptionsDecideToDo(c, cs)
}

func kubernetesOptionsDecideToDo(c *Cluster, cs *ClusterStatus) Operator {
	var cpNodes []*Node
	var nonCpNodes []*Node
	for _, n := range c.Nodes {
		if n.ControlPlane {
			cpNodes = append(cpNodes, n)
		} else {
			nonCpNodes = append(nonCpNodes, n)
		}
	}

	// Check diff of options for rivers, apiservers, controller-managers, and schedulers
	rivers := filterNodes(cpNodes, func(n *Node) bool {
		riversStatus := cs.NodeStatuses[n.Address].Rivers
		if !RiversParams(cpNodes).Equal(riversStatus.BuiltInParams) {
			return true
		}
		if !c.Options.Rivers.Equal(riversStatus.ExtraParams) {
			return true
		}
		return false
	})
	apiservers := filterNodes(cpNodes, func(n *Node) bool {
		status := cs.NodeStatuses[n.Address].APIServer
		if !APIServerParams(cpNodes, n.Address, c.ServiceSubnet).Equal(status.BuiltInParams) {
			return true
		}
		if !c.Options.APIServer.Equal(status.ExtraParams) {
			return true
		}
		return false
	})
	controllerManagers := filterNodes(cpNodes, func(n *Node) bool {
		status := cs.NodeStatuses[n.Address].ControllerManager
		if !ControllerManagerParams(c.Name, c.ServiceSubnet).Equal(status.BuiltInParams) {
			return true
		}
		if !c.Options.ControllerManager.Equal(status.ExtraParams) {
			return true
		}
		return false
	})
	schedulers := filterNodes(cpNodes, func(n *Node) bool {
		status := cs.NodeStatuses[n.Address].Scheduler
		if !SchedulerParams().Equal(status.BuiltInParams) {
			return true
		}
		if !c.Options.Scheduler.Equal(status.ExtraParams) {
			return true
		}
		return false
	})
	if len(rivers)+len(apiservers)+len(controllerManagers)+len(schedulers) > 0 {
		return KubeCPRestartOp(cpNodes, rivers, apiservers, controllerManagers, schedulers, c.Name, c.ServiceSubnet, c.Options)
	}

	// Check diff of rivers options for worker nodes
	rivers = filterNodes(nonCpNodes, func(n *Node) bool {
		riversStatus := cs.NodeStatuses[n.Address].Rivers
		if !RiversParams(cpNodes).Equal(riversStatus.BuiltInParams) {
			return true
		}
		if !c.Options.Rivers.Equal(riversStatus.ExtraParams) {
			return true
		}
		return false
	})
	kubelets := filterNodes(c.Nodes, func(n *Node) bool {
		status := cs.NodeStatuses[n.Address].Kubelet
		if !KubeletServiceParams(n).Equal(status.BuiltInParams) {
			return true
		}
		if !c.Options.Kubelet.ServiceParams.Equal(status.ExtraParams) {
			return true
		}
		if c.Options.Kubelet.Domain != status.Domain {
			return true
		}
		if c.Options.Kubelet.AllowSwap != status.AllowSwap {
			return true
		}
		return false
	})
	proxies := filterNodes(c.Nodes, func(n *Node) bool {
		status := cs.NodeStatuses[n.Address].Proxy
		if !ProxyParams().Equal(status.BuiltInParams) {
			return true
		}
		if !c.Options.Proxy.Equal(status.ExtraParams) {
			return true
		}
		return false
	})
	if len(rivers)+len(kubelets)+len(proxies) > 0 {
		return KubeWorkerRestartOp(cpNodes, rivers, kubelets, proxies, c.Options)
	}

	// TODO check image versions and restart container when image is updated

	return nil
}
