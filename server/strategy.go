package server

import (
	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/cke/op"
	"github.com/cybozu-go/log"
	corev1 "k8s.io/api/core/v1"
)

// DecideOps returns the next operations to do.
// This returns nil when no operation need to be done.
func DecideOps(c *cke.Cluster, cs *cke.ClusterStatus) []cke.Operator {

	nf := NewNodeFilter(c, cs)

	// 1. Run or restart rivers.  This guarantees:
	// - CKE tools image is pulled on all nodes.
	// - Rivers runs on all nodes and will proxy requests only to control plane nodes.
	if ops := riversOps(c, nf); len(ops) > 0 {
		return ops
	}

	// 2. Bootstrap etcd cluster, if not yet.
	if !nf.EtcdBootstrapped() {
		return []cke.Operator{op.EtcdBootOp(nf.ControlPlane(), c.Options.Etcd)}
	}

	// 3. Start etcd containers.
	if nodes := nf.EtcdStoppedMembers(); len(nodes) > 0 {
		return []cke.Operator{op.EtcdStartOp(nodes, c.Options.Etcd)}
	}

	// 4. Wait for etcd cluster to become ready
	if !cs.Etcd.IsHealthy {
		return []cke.Operator{op.EtcdWaitClusterOp(nf.ControlPlane())}
	}

	// 5. Run or restart kubernetes components.
	if ops := k8sOps(c, nf); len(ops) > 0 {
		return ops
	}

	// 6. Maintain etcd cluster.
	if o := etcdMaintOp(c, nf); o != nil {
		return []cke.Operator{o}
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

func riversOps(c *cke.Cluster, nf *NodeFilter) (ops []cke.Operator) {
	if nodes := nf.RiversStoppedNodes(); len(nodes) > 0 {
		ops = append(ops, op.RiversBootOp(nodes, nf.ControlPlane(), c.Options.Rivers))
	}
	if nodes := nf.RiversOutdatedNodes(); len(nodes) > 0 {
		ops = append(ops, op.RiversRestartOp(nodes, nf.ControlPlane(), c.Options.Rivers))
	}
	return ops
}

func k8sOps(c *cke.Cluster, nf *NodeFilter) (ops []cke.Operator) {
	if nodes := nf.APIServerStoppedNodes(); len(nodes) > 0 {
		ops = append(ops, op.APIServerBootOp(nodes, nf.ControlPlane(), c.ServiceSubnet, c.Options.Kubelet.Domain, c.Options.APIServer))
	}
	if nodes := nf.APIServerOutdatedNodes(); len(nodes) > 0 {
		ops = append(ops, op.APIServerRestartOp(nodes, nf.ControlPlane(), c.ServiceSubnet, c.Options.APIServer))
	}
	if nodes := nf.ControllerManagerStoppedNodes(); len(nodes) > 0 {
		ops = append(ops, op.ControllerManagerBootOp(nodes, c.Name, c.ServiceSubnet, c.Options.ControllerManager))
	}
	if nodes := nf.ControllerManagerOutdatedNodes(); len(nodes) > 0 {
		ops = append(ops, op.ControllerManagerRestartOp(nodes, c.Name, c.ServiceSubnet, c.Options.ControllerManager))
	}
	if nodes := nf.SchedulerStoppedNodes(); len(nodes) > 0 {
		ops = append(ops, op.SchedulerBootOp(nodes, c.Name, c.Options.Scheduler))
	}
	if nodes := nf.SchedulerOutdatedNodes(); len(nodes) > 0 {
		ops = append(ops, op.SchedulerRestartOp(nodes, c.Name, c.Options.Scheduler))
	}
	if nodes := nf.KubeletStoppedNodes(); len(nodes) > 0 {
		ops = append(ops, op.KubeletBootOp(nodes, c.Name, c.PodSubnet, c.Options.Kubelet))
	}
	if nodes := nf.KubeletOutdatedNodes(); len(nodes) > 0 {
		ops = append(ops, op.KubeletRestartOp(nodes, c.Name, c.ServiceSubnet, c.Options.Kubelet))
	}
	if nodes := nf.ProxyStoppedNodes(); len(nodes) > 0 {
		ops = append(ops, op.KubeProxyBootOp(nodes, c.Name, c.Options.Proxy))
	}
	if nodes := nf.ProxyOutdatedNodes(); len(nodes) > 0 {
		ops = append(ops, op.KubeProxyRestartOp(nodes, c.Name, c.Options.Proxy))
	}
	return ops
}

func etcdMaintOp(c *cke.Cluster, nf *NodeFilter) cke.Operator {
	if ids := nf.EtcdNonClusterMemberIDs(false); len(ids) > 0 {
		return op.EtcdRemoveMemberOp(nf.ControlPlane(), ids)
	}
	if nodes, ids := nf.EtcdNonCPMembers(false); len(nodes) > 0 {
		return op.EtcdDestroyMemberOp(nf.ControlPlane(), nodes, ids)
	}
	if nodes := nf.EtcdUnstartedMembers(); len(nodes) > 0 {
		return op.EtcdAddMemberOp(nf.ControlPlane(), nodes[0], c.Options.Etcd)
	}

	if !nf.EtcdIsGood() {
		log.Warn("etcd is not good for maintenance", nil)
		// return nil to proceed to k8s maintenance.
		return nil
	}

	// Adding members or removing/restarting healthy members is done only when
	// all members are in sync.

	if nodes := nf.EtcdNewMembers(); len(nodes) > 0 {
		return op.EtcdAddMemberOp(nf.ControlPlane(), nodes[0], c.Options.Etcd)
	}
	if ids := nf.EtcdNonClusterMemberIDs(true); len(ids) > 0 {
		return op.EtcdRemoveMemberOp(nf.ControlPlane(), ids)
	}
	if nodes, ids := nf.EtcdNonCPMembers(true); len(ids) > 0 {
		return op.EtcdDestroyMemberOp(nf.ControlPlane(), nodes, ids)
	}
	if nodes := nf.EtcdOutdatedMembers(); len(nodes) > 0 {
		return op.EtcdRestartOp(nf.ControlPlane(), nodes[0], c.Options.Etcd)
	}

	return nil
}

func k8sMaintOps(cs *cke.ClusterStatus, nf *NodeFilter) (ops []cke.Operator) {
	ks := cs.Kubernetes
	apiServer := nf.HealthyAPIServer()

	if !ks.IsReady {
		return []cke.Operator{op.KubeWaitOp(apiServer)}
	}

	if !ks.RBACRoleExists || !ks.RBACRoleBindingExists {
		ops = append(ops, op.KubeRBACRoleInstallOp(apiServer, ks.RBACRoleExists))
	}

	epOp := decideEpOp(ks.EtcdEndpoints, apiServer, nf.ControlPlane())
	if epOp != nil {
		ops = append(ops, epOp)
	}

	// TODO: maintain Node resources
	return ops
}

func decideEpOp(ep *corev1.Endpoints, apiServer *cke.Node, cpNodes []*cke.Node) cke.Operator {
	if ep == nil {
		return op.KubeEtcdEndpointsCreateOp(apiServer, cpNodes)
	}

	updateOp := op.KubeEtcdEndpointsUpdateOp(apiServer, cpNodes)
	if len(ep.Subsets) != 1 {
		return updateOp
	}

	subset := ep.Subsets[0]
	if len(subset.Ports) != 1 || subset.Ports[0].Port != 2379 {
		return updateOp
	}

	if len(subset.Addresses) != len(cpNodes) {
		return updateOp
	}

	endpoints := make(map[string]bool)
	for _, n := range cpNodes {
		endpoints[n.Address] = true
	}
	for _, a := range subset.Addresses {
		if !endpoints[a.IP] {
			return updateOp
		}
	}

	return nil
}

func cleanOps(c *cke.Cluster, nf *NodeFilter) (ops []cke.Operator) {
	var apiServers, controllerManagers, schedulers, etcds []*cke.Node

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
		ops = append(ops, op.APIServerStopOp(apiServers))
	}
	if len(controllerManagers) > 0 {
		ops = append(ops, op.ControllerManagerStopOp(controllerManagers))
	}
	if len(schedulers) > 0 {
		ops = append(ops, op.SchedulerStopOp(schedulers))
	}
	if len(etcds) > 0 {
		ops = append(ops, op.EtcdStopOp(etcds))
	}
	return ops
}
