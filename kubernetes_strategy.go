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
	target := filterNodes(c.Nodes, func(n *Node) bool {
		return !cs.NodeStatuses[n.Address].Rivers.Running
	})
	if len(target) > 0 {
		return RiversBootOp(target, cpNodes, c.Options.Rivers)
	}

	// Run kube-apiserver on control-plane nodes
	target = filterNodes(cpNodes, func(n *Node) bool {
		return !cs.NodeStatuses[n.Address].APIServer.Running
	})
	if len(target) > 0 {
		return APIServerBootOp(target, cpNodes, c.Options.APIServer, c.ServiceSubnet)
	}

	// Stop kube-apiserver on non-control-plane nodes
	target = filterNodes(nonCpNodes, func(n *Node) bool {
		return cs.NodeStatuses[n.Address].APIServer.Running
	})
	if len(target) > 0 {
		return APIServerStopOp(target)
	}

	// Run kube-controller-manager on control-plane nodes
	target = filterNodes(cpNodes, func(n *Node) bool {
		return !cs.NodeStatuses[n.Address].ControllerManager.Running
	})
	if len(target) > 0 {
		return ControllerManagerBootOp(target, c.Options.ControllerManager, c.ServiceSubnet)
	}

	// Stop kube-controller-manager on non-control-plane nodes
	target = filterNodes(nonCpNodes, func(n *Node) bool {
		return cs.NodeStatuses[n.Address].ControllerManager.Running
	})
	if len(target) > 0 {
		return ControllerManagerStopOp(target)
	}

	// Run kube-scheduler on control-plane nodes
	target = filterNodes(cpNodes, func(n *Node) bool {
		return !cs.NodeStatuses[n.Address].Scheduler.Running
	})
	if len(target) > 0 {
		return SchedulerBootOp(target, c.Options.Scheduler, c.ServiceSubnet)
	}

	// Stop kube-scheduler on non-control-plane nodes
	target = filterNodes(nonCpNodes, func(n *Node) bool {
		return cs.NodeStatuses[n.Address].Scheduler.Running
	})
	if len(target) > 0 {
		return SchedulerStopOp(target)
	}

	// Run kubelet on all nodes
	target = filterNodes(c.Nodes, func(n *Node) bool {
		return !cs.NodeStatuses[n.Address].Kubelet.Running
	})
	if len(target) > 0 {
		return KubeletBootOp(target, c.Options.Kubelet)
	}

	// Run kube-proxy on all nodes
	target = filterNodes(c.Nodes, func(n *Node) bool {
		return !cs.NodeStatuses[n.Address].Proxy.Running
	})
	if len(target) > 0 {
		return ProxyBootOp(target, c.Options.Proxy)
	}

	// Check diff of command options
	return kubernetesOptionsDecideToDo(c, cs)
}

func kubernetesOptionsDecideToDo(c *Cluster, cs *ClusterStatus) Operator {
	cpNodes := controlPlanes(c.Nodes)

	// Check diff of rivers options
	target := filterNodes(c.Nodes, func(n *Node) bool {
		riversStatus := cs.NodeStatuses[n.Address].Rivers
		if !riversParams(cpNodes).Equal(riversStatus.BuiltInParams) {
			return true
		}
		if !c.Options.Rivers.Equal(riversStatus.ExtraParams) {
			return true
		}
		return false
	})
	if len(target) > 0 {
		// Stop just one of targets and go to next iteration, in which
		// the stopped target will be started
		return RiversStopOp([]*Node{target[0]})
	}

	// Check diff of kube-apiserver options
	target = filterNodes(cpNodes, func(n *Node) bool {
		status := cs.NodeStatuses[n.Address].APIServer
		if !apiServerParams(cpNodes, n.Address, c.ServiceSubnet).Equal(status.BuiltInParams) {
			return true
		}
		if !c.Options.APIServer.Equal(status.ExtraParams) {
			return true
		}
		return false
	})
	if len(target) > 0 {
		// Stop just one of targets and go to next iteration, in which
		// the stopped target will be started
		return APIServerStopOp([]*Node{target[0]})
	}

	// Check diff of kube-controller-manager options
	target = filterNodes(cpNodes, func(n *Node) bool {
		status := cs.NodeStatuses[n.Address].ControllerManager
		if !controllerManagerParams().Equal(status.BuiltInParams) {
			return true
		}
		if !c.Options.ControllerManager.Equal(status.ExtraParams) {
			return true
		}
		return false
	})
	if len(target) > 0 {
		// Stop just one of targets and go to next iteration, in which
		// the stopped target will be started
		return ControllerManagerStopOp([]*Node{target[0]})
	}

	// Check diff of kube-scheduler options
	target = filterNodes(cpNodes, func(n *Node) bool {
		status := cs.NodeStatuses[n.Address].Scheduler
		if !schedulerParams().Equal(status.BuiltInParams) {
			return true
		}
		if !c.Options.Scheduler.Equal(status.ExtraParams) {
			return true
		}
		return false
	})
	if len(target) > 0 {
		// Stop just one of targets and go to next iteration, in which
		// the stopped target will be started
		return SchedulerStopOp([]*Node{target[0]})
	}

	// Check diff of kubelet options
	target = filterNodes(c.Nodes, func(n *Node) bool {
		bootOp := KubeletBootOp([]*Node{n}, c.Options.Kubelet).(*kubeletBootOp)
		status := cs.NodeStatuses[n.Address].Kubelet
		if !bootOp.serviceParams(n.Address).Equal(status.BuiltInParams) {
			return true
		}
		if !bootOp.extraParams().Equal(status.ExtraParams) {
			return true
		}
		return false
	})
	if len(target) > 0 {
		// Stop just one of targets and go to next iteration, in which
		// the stopped target will be started
		return KubeletStopOp([]*Node{target[0]})
	}

	return nil
}
