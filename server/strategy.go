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
		return []cke.Operator{op.EtcdBootOp(nf.ControlPlane(), c.Options.Etcd, c.Options.Kubelet.Domain)}
	}

	// 3. Start etcd containers.
	if nodes := nf.EtcdStoppedMembers(); len(nodes) > 0 {
		return []cke.Operator{op.EtcdStartOp(nodes, c.Options.Etcd, c.Options.Kubelet.Domain)}
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
	if ops := k8sMaintOps(c, cs, nf); len(ops) > 0 {
		return ops
	}

	// 8. Stop and delete control plane services running on non control plane nodes.
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
		ops = append(ops, op.KubeletBootOp(nodes, nf.KubeletStoppedRegisteredNodes(), nf.HealthyAPIServer(), c.Name, c.PodSubnet, c.Options.Kubelet))
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
		return op.EtcdAddMemberOp(nf.ControlPlane(), nodes[0], c.Options.Etcd, c.Options.Kubelet.Domain)
	}

	if !nf.EtcdIsGood() {
		log.Warn("etcd is not good for maintenance", nil)
		// return nil to proceed to k8s maintenance.
		return nil
	}

	// Adding members or removing/restarting healthy members is done only when
	// all members are in sync.

	if nodes := nf.EtcdNewMembers(); len(nodes) > 0 {
		return op.EtcdAddMemberOp(nf.ControlPlane(), nodes[0], c.Options.Etcd, c.Options.Kubelet.Domain)
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

func k8sMaintOps(c *cke.Cluster, cs *cke.ClusterStatus, nf *NodeFilter) (ops []cke.Operator) {
	ks := cs.Kubernetes
	apiServer := nf.HealthyAPIServer()

	if !ks.IsReady {
		return []cke.Operator{op.KubeWaitOp(apiServer)}
	}

	if !ks.RBACRoleExists || !ks.RBACRoleBindingExists {
		ops = append(ops, op.KubeRBACRoleInstallOp(apiServer, ks.RBACRoleExists))
	}

	if dnsOps := decideClusterDNSOps(apiServer, c, ks); len(dnsOps) != 0 {
		ops = append(ops, dnsOps...)
	}

	if nodeDNSOps := decideNodeDNSOps(apiServer, c, ks); len(nodeDNSOps) != 0 {
		ops = append(ops, nodeDNSOps...)
	}

	epOp := decideEpOp(ks.EtcdEndpoints, apiServer, nf.ControlPlane())
	if epOp != nil {
		ops = append(ops, epOp)
	}

	if nodes := nf.OutdatedAttrsNodes(); len(nodes) > 0 {
		ops = append(ops, op.KubeNodeUpdateOp(apiServer, nodes))
	}

	if nodes := nf.NonClusterNodes(); len(nodes) > 0 {
		ops = append(ops, op.KubeNodeRemoveOp(apiServer, nodes))
	}

	if etcdBackupOps := decideEtcdBackupOps(apiServer, c, ks); len(etcdBackupOps) != 0 {
		ops = append(ops, etcdBackupOps...)
	}

	return ops
}

func decideClusterDNSOps(apiServer *cke.Node, c *cke.Cluster, ks cke.KubernetesClusterStatus) (ops []cke.Operator) {
	desiredDNSServers := c.DNSServers
	if ks.DNSService != nil {
		switch ip := ks.DNSService.Spec.ClusterIP; ip {
		case "", "None":
		default:
			desiredDNSServers = []string{ip}
		}
	}
	desiredClusterDomain := c.Options.Kubelet.Domain

	if len(desiredClusterDomain) == 0 {
		panic("Options.Kubelet.Domain is empty")
	}

	if !ks.ClusterDNS.ServiceAccountExists {
		ops = append(ops, op.KubeClusterDNSCreateServiceAccountOp(apiServer))
	}
	if !ks.ClusterDNS.RBACRoleExists {
		ops = append(ops, op.KubeClusterDNSCreateRBACRoleOp(apiServer))
	}
	if !ks.ClusterDNS.RBACRoleBindingExists {
		ops = append(ops, op.KubeClusterDNSCreateRBACRoleBindingOp(apiServer))
	}
	if ks.ClusterDNS.ConfigMap == nil {
		ops = append(ops, op.KubeClusterDNSCreateConfigMapOp(apiServer, desiredClusterDomain, desiredDNSServers))
	} else {
		actualConfigData := ks.ClusterDNS.ConfigMap.Data
		expectedConfig := op.ClusterDNSConfigMap(desiredClusterDomain, desiredDNSServers)
		if actualConfigData["Corefile"] != expectedConfig.Data["Corefile"] {
			ops = append(ops, op.KubeClusterDNSUpdateConfigMapOp(apiServer, expectedConfig))
		}
	}
	if ks.ClusterDNS.Deployment == nil {
		ops = append(ops, op.KubeClusterDNSCreateDeploymentOp(apiServer))
	} else {
		if ks.ClusterDNS.Deployment.Annotations["cke.cybozu.com/image"] != cke.CoreDNSImage.Name() ||
			ks.ClusterDNS.Deployment.Annotations["cke.cybozu.com/template-version"] != op.CoreDNSTemplateVersion {
			ops = append(ops, op.KubeClusterDNSUpdateDeploymentOp(apiServer))
		}
	}
	if !ks.ClusterDNS.ServiceExists {
		ops = append(ops, op.KubeClusterDNSCreateOp(apiServer))
	}

	return ops
}

func decideNodeDNSOps(apiServer *cke.Node, c *cke.Cluster, ks cke.KubernetesClusterStatus) (ops []cke.Operator) {
	if len(ks.ClusterDNS.ClusterIP) == 0 {
		return nil
	}

	if ks.NodeDNS.DaemonSet == nil {
		ops = append(ops, op.KubeNodeDNSCreateDaemonSetOp(apiServer))
	} else {
		if ks.NodeDNS.DaemonSet.Annotations["cke.cybozu.com/image"] != cke.UnboundImage.Name() ||
			ks.NodeDNS.DaemonSet.Annotations["cke.cybozu.com/template-version"] != op.UnboundTemplateVersion {
			ops = append(ops, op.KubeNodeDNSUpdateDaemonSetOp(apiServer))
		}
	}

	desiredDNSServers := c.DNSServers
	if ks.DNSService != nil {
		switch ip := ks.DNSService.Spec.ClusterIP; ip {
		case "", "None":
		default:
			desiredDNSServers = []string{ip}
		}
	}

	if ks.NodeDNS.ConfigMap == nil {
		ops = append(ops, op.KubeNodeDNSCreateConfigMapOp(apiServer, ks.ClusterDNS.ClusterIP, c.Options.Kubelet.Domain, desiredDNSServers))
	} else {
		actualConfigData := ks.NodeDNS.ConfigMap.Data
		expectedConfig := op.NodeDNSConfigMap(ks.ClusterDNS.ClusterIP, c.Options.Kubelet.Domain, desiredDNSServers)
		if actualConfigData["unbound.conf"] != expectedConfig.Data["unbound.conf"] {
			ops = append(ops, op.KubeNodeDNSUpdateConfigMapOp(apiServer, expectedConfig))
		}
	}

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

func decideEtcdBackupOps(apiServer *cke.Node, c *cke.Cluster, ks cke.KubernetesClusterStatus) (ops []cke.Operator) {
	if c.EtcdBackup.Enabled == false {
		if ks.EtcdBackup.ConfigMap != nil {
			ops = append(ops, op.EtcdBackupConfigMapRemoveOp(apiServer))
		}
		if ks.EtcdBackup.Secret != nil {
			ops = append(ops, op.EtcdBackupSecretRemoveOp(apiServer))
		}
		if ks.EtcdBackup.CronJob != nil {
			ops = append(ops, op.EtcdBackupCronJobRemoveOp(apiServer))
		}
		return ops
	}

	if ks.EtcdBackup.ConfigMap == nil {
		ops = append(ops, op.EtcdBackupConfigMapCreateOp(apiServer))
	}

	if ks.EtcdBackup.Secret == nil {
		ops = append(ops, op.EtcdBackupSecretCreateOp(apiServer))
	}

	if ks.EtcdBackup.CronJob == nil {
		ops = append(ops, op.EtcdBackupCronJobCreateOp(apiServer, c.EtcdBackup))
	} else if needUpdateEtcdBackupCronJob(c, ks) {
		ops = append(ops, op.EtcdBackupCronJobUpdateOp(apiServer, c.EtcdBackup))
	}

	return ops
}

func needUpdateEtcdBackupCronJob(c *cke.Cluster, ks cke.KubernetesClusterStatus) bool {
	if ks.EtcdBackup.CronJob.Spec.Schedule != c.EtcdBackup.Schedule {
		return true
	}
	volumes := ks.EtcdBackup.CronJob.Spec.JobTemplate.Spec.Template.Spec.Volumes
	vol := new(corev1.Volume)
	for _, v := range volumes {
		if v.Name == "etcd-backup" {
			vol = &v
			break
		}
	}
	if vol == nil {
		return true
	}

	if vol.PersistentVolumeClaim == nil {
		return true
	}
	if vol.PersistentVolumeClaim.ClaimName != c.EtcdBackup.PVCName {
		return true
	}
	return false
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
