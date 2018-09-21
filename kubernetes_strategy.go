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
	nodes := filterNodes(c.Nodes, func(n *Node) bool {
		return !cs.NodeStatuses[n.Address].Rivers.Running
	})
	if len(nodes) > 0 {
		return RiversBootOp(nodes, cpNodes, c.Options.Rivers)
	}

	// Run kubernetes control planes on control-plane nodes
	nodes = filterNodes(cpNodes, func(n *Node) bool {
		return !cs.NodeStatuses[n.Address].APIServer.Running
	})
	if len(nodes) > 0 {
		return APIServerBootOp(nodes, cpNodes, c.ServiceSubnet, c.Options.APIServer)
	}
	nodes = filterNodes(cpNodes, func(n *Node) bool {
		return !cs.NodeStatuses[n.Address].ControllerManager.Running
	})
	if len(nodes) > 0 {
		return ControllerManagerBootOp(nodes, c.Name, c.ServiceSubnet, c.Options.ControllerManager)
	}
	nodes = filterNodes(cpNodes, func(n *Node) bool {
		return !cs.NodeStatuses[n.Address].Scheduler.Running
	})
	if len(nodes) > 0 {
		return SchedulerBootOp(nodes, c.Name, c.Options.Scheduler)
	}

	// Stop kubernetes control plane containers running on non-control-plane nodes
	nodes = filterNodes(nonCpNodes, func(n *Node) bool {
		return cs.NodeStatuses[n.Address].APIServer.Running
	})
	if len(nodes) > 0 {
		return ContainerStopOp(nodes, kubeAPIServerContainerName)
	}
	nodes = filterNodes(nonCpNodes, func(n *Node) bool {
		return cs.NodeStatuses[n.Address].ControllerManager.Running
	})
	if len(nodes) > 0 {
		return ContainerStopOp(nodes, kubeControllerManagerContainerName)
	}
	nodes = filterNodes(nonCpNodes, func(n *Node) bool {
		return cs.NodeStatuses[n.Address].Scheduler.Running
	})
	if len(nodes) > 0 {
		return ContainerStopOp(nodes, kubeSchedulerContainerName)
	}

	// Run kubelet and kube-proxy on all nodes
	nodes = filterNodes(c.Nodes, func(n *Node) bool {
		return !cs.NodeStatuses[n.Address].Kubelet.Running
	})
	if len(nodes) > 0 {
		return KubeletBootOp(nodes, c.Name, c.PodSubnet, c.Options.Kubelet)
	}
	nodes = filterNodes(c.Nodes, func(n *Node) bool {
		return !cs.NodeStatuses[n.Address].Proxy.Running
	})
	if len(nodes) > 0 {
		return KubeProxyBootOp(nodes, c.Name, c.Options.Proxy)
	}

	// Restart containers running with stale images or configurations.
	op := kubernetesDecideRestart(c, cs)
	if op != nil {
		return op
	}

	// Configure kubernetes
	ks := cs.Kubernetes
	if !ks.RBACRoleExists || !ks.RBACRoleBindingExists {
		return KubeRBACRoleInstallOp(cpNodes[0], ks.RBACRoleExists)
	}

	return nil
}

func kubernetesDecideRestart(c *Cluster, cs *ClusterStatus) Operator {
	var cpNodes []*Node
	var nonCpNodes []*Node
	for _, n := range c.Nodes {
		if n.ControlPlane {
			cpNodes = append(cpNodes, n)
		} else {
			nonCpNodes = append(nonCpNodes, n)
		}
	}

	nodes := filterNodes(c.Nodes, func(n *Node) bool {
		riversStatus := cs.NodeStatuses[n.Address].Rivers
		if !RiversParams(cpNodes).Equal(riversStatus.BuiltInParams) {
			return true
		}
		if !c.Options.Rivers.Equal(riversStatus.ExtraParams) {
			return true
		}
		return false
	})
	if len(nodes) > 0 {
		return RiversRestartOp(nodes, cpNodes, c.Options.Rivers)
	}

	nodes = filterNodes(cpNodes, func(n *Node) bool {
		status := cs.NodeStatuses[n.Address].APIServer
		if !APIServerParams(cpNodes, n.Address, c.ServiceSubnet).Equal(status.BuiltInParams) {
			return true
		}
		if !c.Options.APIServer.Equal(status.ExtraParams) {
			return true
		}
		return false
	})
	if len(nodes) > 0 {
		return APIServerRestartOp(nodes, cpNodes, c.ServiceSubnet, c.Options.APIServer)
	}

	nodes = filterNodes(cpNodes, func(n *Node) bool {
		status := cs.NodeStatuses[n.Address].ControllerManager
		if !ControllerManagerParams(c.Name, c.ServiceSubnet).Equal(status.BuiltInParams) {
			return true
		}
		if !c.Options.ControllerManager.Equal(status.ExtraParams) {
			return true
		}
		return false
	})
	if len(nodes) > 0 {
		return ControllerManagerRestartOp(nodes, c.Name, c.ServiceSubnet, c.Options.ControllerManager)
	}

	nodes = filterNodes(cpNodes, func(n *Node) bool {
		status := cs.NodeStatuses[n.Address].Scheduler
		if !SchedulerParams().Equal(status.BuiltInParams) {
			return true
		}
		if !c.Options.Scheduler.Equal(status.ExtraParams) {
			return true
		}
		return false
	})
	if len(nodes) > 0 {
		return SchedulerRestartOp(nodes, c.Name, c.Options.Scheduler)
	}

	nodes = filterNodes(c.Nodes, func(n *Node) bool {
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
	if len(nodes) > 0 {
		return KubeletRestartOp(nodes, c.Name, c.ServiceSubnet, c.Options.Kubelet)
	}

	nodes = filterNodes(c.Nodes, func(n *Node) bool {
		status := cs.NodeStatuses[n.Address].Proxy
		if !ProxyParams().Equal(status.BuiltInParams) {
			return true
		}
		if !c.Options.Proxy.Equal(status.ExtraParams) {
			return true
		}
		return false
	})
	if len(nodes) > 0 {
		return KubeProxyRestartOp(nodes, c.Name, c.Options.Proxy)
	}

	// TODO check image versions and restart container when image is updated

	return nil
}
