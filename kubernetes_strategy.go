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
	if len(apiservers) > 0 {
		return APIServerBootOp(cpNodes, apiservers, c.ServiceSubnet, c.Options.APIServer)
	}
	controllerManagers := filterNodes(cpNodes, func(n *Node) bool {
		return !cs.NodeStatuses[n.Address].ControllerManager.Running
	})
	if len(controllerManagers) > 0 {
		return ControllerManagerBootOp(controllerManagers, c.Name, c.ServiceSubnet, c.Options.ControllerManager)
	}
	schedulers := filterNodes(cpNodes, func(n *Node) bool {
		return !cs.NodeStatuses[n.Address].Scheduler.Running
	})
	if len(schedulers) > 0 {
		return SchedulerBootOp(schedulers, c.Name, c.Options.Scheduler)
	}

	// Stop kubernetes control plane containers running on non-control-plane nodes
	apiservers = filterNodes(nonCpNodes, func(n *Node) bool {
		return cs.NodeStatuses[n.Address].APIServer.Running
	})
	if len(apiservers) > 0 {
		return ContainerStopOp(apiservers, kubeAPIServerContainerName)
	}
	controllerManagers = filterNodes(nonCpNodes, func(n *Node) bool {
		return cs.NodeStatuses[n.Address].ControllerManager.Running
	})
	if len(controllerManagers) > 0 {
		return ContainerStopOp(apiservers, kubeControllerManagerContainerName)
	}
	schedulers = filterNodes(nonCpNodes, func(n *Node) bool {
		return cs.NodeStatuses[n.Address].Scheduler.Running
	})
	if len(schedulers) > 0 {
		return ContainerStopOp(apiservers, kubeSchedulerContainerName)
	}

	// Run kubelet and kube-proxy on all nodes
	kubelets := filterNodes(c.Nodes, func(n *Node) bool {
		return !cs.NodeStatuses[n.Address].Kubelet.Running
	})
	if len(kubelets) > 0 {
		return KubeletBootOp(kubelets, c.Name, c.PodSubnet, c.Options.Kubelet)
	}
	proxies := filterNodes(c.Nodes, func(n *Node) bool {
		return !cs.NodeStatuses[n.Address].Proxy.Running
	})
	if len(proxies) > 0 {
		return KubeProxyBootOp(proxies, c.Options.Proxy)
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

	rivers := filterNodes(c.Nodes, func(n *Node) bool {
		riversStatus := cs.NodeStatuses[n.Address].Rivers
		if !RiversParams(cpNodes).Equal(riversStatus.BuiltInParams) {
			return true
		}
		if !c.Options.Rivers.Equal(riversStatus.ExtraParams) {
			return true
		}
		return false
	})
	if len(rivers) > 0 {
		return RiversRestartOp(cpNodes, rivers, c.Options.Rivers)
	}

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
	if len(apiservers) > 0 {
		return APIServerRestartOp(cpNodes, apiservers, c.ServiceSubnet, c.Options.APIServer)
	}

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
	if len(controllerManagers) > 0 {
		return ControllerManagerRestartOp(cpNodes, controllerManagers, c.Name, c.ServiceSubnet, c.Options.ControllerManager)
	}

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
	if len(schedulers) > 0 {
		return SchedulerRestartOp(cpNodes, schedulers, c.Name, c.Options.Scheduler)
	}

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
	if len(kubelets) > 0 {
		return KubeletRestartOp(cpNodes, kubelets, c.Name, c.ServiceSubnet, c.Options.Kubelet)
	}

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
	if len(proxies) > 0 {
		return KubeProxyRestartOp(cpNodes, proxies, c.Options.Proxy)
	}

	// TODO check image versions and restart container when image is updated

	return nil
}
