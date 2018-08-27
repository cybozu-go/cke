package cke

import "reflect"

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
		return RiversBootOp(target, cpNodes, cs.Agents, c.Options.Rivers)
	}

	// Run kube-apiserver on control-plane nodes
	target = filterNodes(cpNodes, func(n *Node) bool {
		return !cs.NodeStatuses[n.Address].APIServer.Running
	})
	if len(target) > 0 {
		return APIServerBootOp(target, cpNodes, cs.Agents, c.Options.APIServer, c.ServiceSubnet)
	}

	// Stop kube-apiserver on non-control-plane nodes
	target = filterNodes(nonCpNodes, func(n *Node) bool {
		return cs.NodeStatuses[n.Address].APIServer.Running
	})
	if len(target) > 0 {
		return APIServerStopOp(target, cs.Agents)
	}

	// Run kube-controller-manager on control-plane nodes
	target = filterNodes(cpNodes, func(n *Node) bool {
		return !cs.NodeStatuses[n.Address].ControllerManager.Running
	})
	if len(target) > 0 {
		return ControllerManagerBootOp(target, cs.Agents, c.Options.ControllerManager, c.ServiceSubnet)
	}

	// Stop kube-controller-manager on non-control-plane nodes
	target = filterNodes(nonCpNodes, func(n *Node) bool {
		return cs.NodeStatuses[n.Address].ControllerManager.Running
	})
	if len(target) > 0 {
		return ControllerManagerStopOp(target, cs.Agents)
	}

	// Run kube-scheduler on control-plane nodes
	target = filterNodes(cpNodes, func(n *Node) bool {
		return !cs.NodeStatuses[n.Address].Scheduler.Running
	})
	if len(target) > 0 {
		return SchedulerBootOp(target, cs.Agents, c.Options.Scheduler, c.ServiceSubnet)
	}

	// Stop kube-scheduler on non-control-plane nodes
	target = filterNodes(nonCpNodes, func(n *Node) bool {
		return cs.NodeStatuses[n.Address].Scheduler.Running
	})
	if len(target) > 0 {
		return SchedulerStopOp(target, cs.Agents)
	}

	// Run kubelet on all nodes
	target = filterNodes(c.Nodes, func(n *Node) bool {
		return !cs.NodeStatuses[n.Address].Kubelet.Running
	})
	if len(target) > 0 {
		return KubeletBootOp(target, cs.Agents, c.Options.Kubelet)
	}

	// Run kube-proxy on all nodes
	target = filterNodes(c.Nodes, func(n *Node) bool {
		return !cs.NodeStatuses[n.Address].Proxy.Running
	})
	if len(target) > 0 {
		return ProxyBootOp(target, cs.Agents, c.Options.Proxy)
	}

	// Check diff of command options
	return kubernetesOptionsDecideToDo(c, cs)
}

func kubernetesOptionsDecideToDo(c *Cluster, cs *ClusterStatus) Operator {
	cpNodes := controlPlanes(c.Nodes)

	// Check diff of rivers options
	target := filterNodes(c.Nodes, func(n *Node) bool {
		riversStatus := cs.NodeStatuses[n.Address].Rivers
		if !reflect.DeepEqual(riversParams(cpNodes), riversStatus.BuiltInParams) {
			return true
		}
		if !reflect.DeepEqual(c.Options.Rivers, riversStatus.ExtraParams) {
			return true
		}
		return false
	})
	if len(target) > 0 {
		// Stop just one of targets and go to next iteration, in which
		// the stopped target will be started
		return RiversStopOp([]*Node{target[0]}, cs.Agents)
	}

	// Check diff of kube-apiserver options
	target = filterNodes(cpNodes, func(n *Node) bool {
		status := cs.NodeStatuses[n.Address].APIServer
		if !reflect.DeepEqual(apiServerParams(cpNodes, n.Address, c.ServiceSubnet), status.BuiltInParams) {
			return true
		}
		if !reflect.DeepEqual(c.Options.APIServer, status.ExtraParams) {
			return true
		}
		return false
	})
	if len(target) > 0 {
		// Stop just one of targets and go to next iteration, in which
		// the stopped target will be started
		return APIServerStopOp([]*Node{target[0]}, cs.Agents)
	}

	// Check diff of kube-controller-manager options
	target = filterNodes(cpNodes, func(n *Node) bool {
		status := cs.NodeStatuses[n.Address].ControllerManager
		if !reflect.DeepEqual(controllerManagerParams(), status.BuiltInParams) {
			return true
		}
		if !reflect.DeepEqual(c.Options.ControllerManager, status.ExtraParams) {
			return true
		}
		return false
	})
	if len(target) > 0 {
		// Stop just one of targets and go to next iteration, in which
		// the stopped target will be started
		return ControllerManagerStopOp([]*Node{target[0]}, cs.Agents)
	}

	// Check diff of kube-scheduler options
	target = filterNodes(cpNodes, func(n *Node) bool {
		status := cs.NodeStatuses[n.Address].Scheduler
		if !reflect.DeepEqual(schedulerParams(), status.BuiltInParams) {
			return true
		}
		if !reflect.DeepEqual(c.Options.Scheduler, status.ExtraParams) {
			return true
		}
		return false
	})
	if len(target) > 0 {
		// Stop just one of targets and go to next iteration, in which
		// the stopped target will be started
		return SchedulerStopOp([]*Node{target[0]}, cs.Agents)
	}

	// Check diff of kubelet options
	target = filterNodes(c.Nodes, func(n *Node) bool {
		bootOp := KubeletBootOp([]*Node{n}, cs.Agents, c.Options.Kubelet).(*kubeletBootOp)
		status := cs.NodeStatuses[n.Address].Kubelet
		if !reflect.DeepEqual(bootOp.serviceParams(n.Address), status.BuiltInParams) {
			return true
		}
		if !reflect.DeepEqual(bootOp.extraParams(), status.ExtraParams) {
			return true
		}
		return false
	})
	if len(target) > 0 {
		// Stop just one of targets and go to next iteration, in which
		// the stopped target will be started
		return KubeletStopOp([]*Node{target[0]}, cs.Agents)
	}

	return nil
}

func controlPlanes(nodes []*Node) []*Node {
	return filterNodes(nodes, func(n *Node) bool {
		return n.ControlPlane
	})
}

func filterNodes(nodes []*Node, f func(n *Node) bool) []*Node {
	var filtered []*Node
	for _, n := range nodes {
		if f(n) {
			filtered = append(filtered, n)
		}
	}
	return filtered
}
