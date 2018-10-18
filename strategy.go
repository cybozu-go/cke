package cke

import (
	"github.com/cybozu-go/log"
	corev1 "k8s.io/api/core/v1"
)

// DecideOps returns the next operations to do.
// This returns nil when no operation need to be done.
func DecideOps(c *Cluster, cs *ClusterStatus) []Operator {

	nf := NewNodeFilter(c, cs)

	// 1. Run or restart rivers.  This guarantees:
	// - CKE tools image is pulled on all nodes.
	// - Rivers runs on all nodes and will proxy requests only to control plane nodes.
	if ops := riversOps(c, nf); len(ops) > 0 {
		return ops
	}

	// 2. Bootstrap etcd cluster, if not yet.
	if !nf.EtcdBootstrapped() {
		return []Operator{EtcdBootOp(nf.ControlPlane(), c.Options.Etcd)}
	}

	// 3. Start etcd containers.
	if nodes := nf.EtcdStoppedMembers(); len(nodes) > 0 {
		return []Operator{EtcdStartOp(nodes, c.Options.Etcd)}
	}

	// 4. Wait for etcd cluster to become ready
	if !cs.Etcd.IsHealthy {
		return []Operator{EtcdWaitClusterOp(nf.ControlPlane())}
	}

	// 5. Run or restart kubernetes components.
	if ops := k8sOps(c, nf); len(ops) > 0 {
		return ops
	}

	// 6. Maintain etcd cluster.
	if op := etcdMaintOp(c, nf); op != nil {
		return []Operator{op}
	}

	// 7. Maintain k8s resources.
	if ops := k8sMaintOps(cs, nf); len(ops) > 0 {
		return ops
	}

	// 8. Stop and delele control plane services running on non control plane nodes.
	if ops := cleanOps(c, nf); len(ops) > 0 {
		return ops
	}

	return nil
}

func riversOps(c *Cluster, nf *NodeFilter) (ops []Operator) {
	if nodes := nf.RiversStoppedNodes(); len(nodes) > 0 {
		ops = append(ops, RiversBootOp(nodes, nf.ControlPlane(), c.Options.Rivers))
	}
	if nodes := nf.RiversOutdatedNodes(); len(nodes) > 0 {
		ops = append(ops, RiversRestartOp(nodes, nf.ControlPlane(), c.Options.Rivers))
	}
	return ops
}

func k8sOps(c *Cluster, nf *NodeFilter) (ops []Operator) {
	if nodes := nf.APIServerStoppedNodes(); len(nodes) > 0 {
		ops = append(ops, APIServerBootOp(nodes, nf.ControlPlane(), c.ServiceSubnet, c.Options.Kubelet.Domain, c.Options.APIServer))
	}
	if nodes := nf.APIServerOutdatedNodes(); len(nodes) > 0 {
		ops = append(ops, APIServerRestartOp(nodes, nf.ControlPlane(), c.ServiceSubnet, c.Options.APIServer))
	}
	if nodes := nf.ControllerManagerStoppedNodes(); len(nodes) > 0 {
		ops = append(ops, ControllerManagerBootOp(nodes, c.Name, c.ServiceSubnet, c.Options.ControllerManager))
	}
	if nodes := nf.ControllerManagerOutdatedNodes(); len(nodes) > 0 {
		ops = append(ops, ControllerManagerRestartOp(nodes, c.Name, c.ServiceSubnet, c.Options.ControllerManager))
	}
	if nodes := nf.SchedulerStoppedNodes(); len(nodes) > 0 {
		ops = append(ops, SchedulerBootOp(nodes, c.Name, c.Options.Scheduler))
	}
	if nodes := nf.SchedulerOutdatedNodes(); len(nodes) > 0 {
		ops = append(ops, SchedulerRestartOp(nodes, c.Name, c.Options.Scheduler))
	}
	if nodes := nf.KubeletStoppedNodes(); len(nodes) > 0 {
		ops = append(ops, KubeletBootOp(nodes, c.Name, c.PodSubnet, c.Options.Kubelet))
	}
	if nodes := nf.KubeletOutdatedNodes(); len(nodes) > 0 {
		ops = append(ops, KubeletRestartOp(nodes, c.Name, c.ServiceSubnet, c.Options.Kubelet))
	}
	if nodes := nf.ProxyStoppedNodes(); len(nodes) > 0 {
		ops = append(ops, KubeProxyBootOp(nodes, c.Name, c.Options.Proxy))
	}
	if nodes := nf.ProxyOutdatedNodes(); len(nodes) > 0 {
		ops = append(ops, KubeProxyRestartOp(nodes, c.Name, c.Options.Proxy))
	}
	return ops
}

func etcdMaintOp(c *Cluster, nf *NodeFilter) Operator {
	if ids := nf.EtcdNonClusterMemberIDs(false); len(ids) > 0 {
		return EtcdRemoveMemberOp(nf.ControlPlane(), ids)
	}
	if nodes, ids := nf.EtcdNonCPMembers(false); len(nodes) > 0 {
		return EtcdDestroyMemberOp(nf.ControlPlane(), nodes, ids)
	}
	if nodes := nf.EtcdUnstartedMembers(); len(nodes) > 0 {
		return EtcdAddMemberOp(nf.ControlPlane(), nodes[0], c.Options.Etcd)
	}

	if !nf.EtcdIsGood() {
		log.Warn("etcd is not good for maintenance", nil)
		// return nil to proceed to k8s maintenance.
		return nil
	}

	// Adding members or removing/restarting healthy members is done only when
	// all members are in sync.

	if nodes := nf.EtcdNewMembers(); len(nodes) > 0 {
		return EtcdAddMemberOp(nf.ControlPlane(), nodes[0], c.Options.Etcd)
	}
	if ids := nf.EtcdNonClusterMemberIDs(true); len(ids) > 0 {
		return EtcdRemoveMemberOp(nf.ControlPlane(), ids)
	}
	if nodes, ids := nf.EtcdNonCPMembers(true); len(ids) > 0 {
		return EtcdDestroyMemberOp(nf.ControlPlane(), nodes, ids)
	}
	if nodes := nf.EtcdOutdatedMembers(); len(nodes) > 0 {
		return EtcdRestartOp(nf.ControlPlane(), nodes[0], c.Options.Etcd)
	}

	return nil
}

func k8sMaintOps(cs *ClusterStatus, nf *NodeFilter) (ops []Operator) {
	ks := cs.Kubernetes
	apiServer := nf.HealthyAPIServer()

	if !ks.RBACRoleExists || !ks.RBACRoleBindingExists {
		ops = append(ops, KubeRBACRoleInstallOp(apiServer, ks.RBACRoleExists))
	}

	epOp := decideEpOp(ks.EtcdEndpoints, apiServer, nf.ControlPlane())
	if epOp != nil {
		ops = append(ops, epOp)
	}

	// TODO: maintain Node resources
	return ops
}

func decideEpOp(ep *corev1.Endpoints, apiServer *Node, cpNodes []*Node) Operator {
	if ep == nil {
		return KubeEtcdEndpointsCreateOp(apiServer, cpNodes)
	}

	op := KubeEtcdEndpointsUpdateOp(apiServer, cpNodes)
	if len(ep.Subsets) != 1 {
		return op
	}

	subset := ep.Subsets[0]
	if len(subset.Ports) != 1 || subset.Ports[0].Port != 2379 {
		return op
	}

	if len(subset.Addresses) != len(cpNodes) {
		return op
	}

	endpoints := make(map[string]bool)
	for _, n := range cpNodes {
		endpoints[n.Address] = true
	}
	for _, a := range subset.Addresses {
		if !endpoints[a.IP] {
			return op
		}
	}

	return nil
}

func cleanOps(c *Cluster, nf *NodeFilter) (ops []Operator) {
	var apiServers, controllerManagers, schedulers, etcds []*Node

	for _, n := range c.Nodes {
		if n.ControlPlane {
			continue
		}

		st := nf.nodeStatus(n)
		if st.Etcd.Running && nf.EtcdIsGood() {
			etcds = append(etcds, n)
		}
		if st.APIServer.Running {
			apiServers = append(apiServers, n)
		}
		if st.ControllerManager.Running {
			controllerManagers = append(controllerManagers, n)
		}
		if st.Scheduler.Running {
			schedulers = append(schedulers, n)
		}
	}

	if len(apiServers) > 0 {
		ops = append(ops, ContainerStopOp(apiServers, kubeAPIServerContainerName))
	}
	if len(controllerManagers) > 0 {
		ops = append(ops, ContainerStopOp(controllerManagers, kubeControllerManagerContainerName))
	}
	if len(schedulers) > 0 {
		ops = append(ops, ContainerStopOp(schedulers, kubeSchedulerContainerName))
	}
	if len(etcds) > 0 {
		ops = append(ops, ContainerStopOp(etcds, etcdContainerName))
	}
	return ops
}
