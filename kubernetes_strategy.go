package cke

func kubernetesDecideToDo(c *Cluster, cs *ClusterStatus) Operator {
	var cpNodes []*Node
	for _, n := range c.Nodes {
		if n.ControlPlane {
			cpNodes = append(cpNodes, n)
		}
	}

	// (1) Run Rivers
	var target []*Node
	for _, n := range cpNodes {
		if !cs.NodeStatuses[n.Address].Rivers.Running {
			target = append(target, n)
		}
	}
	if len(target) > 0 {
		return RiversBootOp(target, cs.Agents, c.Options.Rivers)
	}

	// (2) Run kube-apiserver
	target = []*Node{}
	for _, n := range cpNodes {
		if !cs.NodeStatuses[n.Address].APIServer.Running {
			target = append(target, n)
		}
	}
	if len(target) > 0 {
		return APIServerBootOp(target, cs.Agents, c.Options.APIServer, c.ServiceSubnet)
	}

	// (3) Run kube-controller-manager
	target = []*Node{}
	for _, n := range cpNodes {
		if !cs.NodeStatuses[n.Address].ControllerManager.Running {
			target = append(target, n)
		}
	}
	if len(target) > 0 {
		return ControllerManagerBootOp(target, cs.Agents, c.Options.ControllerManager, c.ServiceSubnet)
	}

	// (3) Run kube-scheduler
	target = []*Node{}
	for _, n := range cpNodes {
		if !cs.NodeStatuses[n.Address].Scheduler.Running {
			target = append(target, n)
		}
	}
	if len(target) > 0 {
		return SchedulerBootOp(target, cs.Agents, c.Options.Scheduler, c.ServiceSubnet)
	}

	return nil
}
