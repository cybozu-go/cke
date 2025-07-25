package server

import (
	"strings"

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/cke/op"
	"github.com/cybozu-go/cke/op/etcd"
	"github.com/cybozu-go/cke/op/k8s"
	"github.com/cybozu-go/log"
	"github.com/google/go-cmp/cmp"
	"go.etcd.io/etcd/api/v3/etcdserverpb"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
)

// NodeFilter filters nodes to
type NodeFilter struct {
	cluster    *cke.Cluster
	status     *cke.ClusterStatus
	nodeMap    map[string]*cke.Node
	addressMap map[string]string
	cp         []*cke.Node
}

// NewNodeFilter creates and initializes NodeFilter.
func NewNodeFilter(cluster *cke.Cluster, status *cke.ClusterStatus) *NodeFilter {
	nodeMap := make(map[string]*cke.Node)
	addressMap := make(map[string]string)
	cp := make([]*cke.Node, 0, 5)

	for _, n := range cluster.Nodes {
		nodeMap[n.Address] = n
		if len(n.Hostname) != 0 {
			addressMap[n.Hostname] = n.Address
		}

		if n.ControlPlane {
			cp = append(cp, n)
		}
	}

	return &NodeFilter{
		cluster:    cluster,
		status:     status,
		nodeMap:    nodeMap,
		addressMap: addressMap,
		cp:         cp,
	}
}

func (nf *NodeFilter) nodeStatus(n *cke.Node) *cke.NodeStatus {
	return nf.status.NodeStatuses[n.Address]
}

// InCluster returns true if a node having address is defined in cluster YAML.
func (nf *NodeFilter) InCluster(address string) bool {
	_, ok := nf.nodeMap[address]
	return ok
}

// ControlPlane returns control plane nodes.
func (nf *NodeFilter) ControlPlane() []*cke.Node {
	return nf.cp
}

// RiversStoppedNodes returns nodes that are not running rivers.
func (nf *NodeFilter) RiversStoppedNodes() (nodes []*cke.Node) {
	for _, n := range nf.cluster.Nodes {
		if !nf.nodeStatus(n).Rivers.Running {
			nodes = append(nodes, n)
		}
	}
	return nodes
}

// RiversOutdatedNodes returns nodes that are running rivers with outdated image or params.
func (nf *NodeFilter) RiversOutdatedNodes() (nodes []*cke.Node) {
	currentBuiltIn := op.RiversParams(nf.cp, op.RiversUpstreamPort, op.RiversListenPort)
	currentExtra := nf.cluster.Options.Rivers

	for _, n := range nf.cluster.Nodes {
		st := nf.nodeStatus(n).Rivers
		switch {
		case !st.Running:
			// stopped nodes are excluded
		case cke.ToolsImage.Name() != st.Image:
			fallthrough
		case !currentBuiltIn.Equal(st.BuiltInParams):
			fallthrough
		case !currentExtra.Equal(st.ExtraParams):
			nodes = append(nodes, n)
		}
	}
	return nodes
}

// EtcdRiversStoppedNodes returns nodes that are not running rivers.
func (nf *NodeFilter) EtcdRiversStoppedNodes() (cps []*cke.Node) {
	for _, n := range nf.ControlPlane() {
		if !nf.nodeStatus(n).EtcdRivers.Running {
			cps = append(cps, n)
		}
	}
	return cps
}

// EtcdRiversOutdatedNodes returns nodes that are running rivers with outdated image or params.
func (nf *NodeFilter) EtcdRiversOutdatedNodes() (cps []*cke.Node) {
	currentBuiltIn := op.RiversParams(nf.cp, op.EtcdRiversUpstreamPort, op.EtcdRiversListenPort)
	currentExtra := nf.cluster.Options.EtcdRivers

	for _, n := range nf.ControlPlane() {
		st := nf.nodeStatus(n).EtcdRivers
		switch {
		case !st.Running:
			// stopped nodes are excluded
		case cke.ToolsImage.Name() != st.Image:
			fallthrough
		case !currentBuiltIn.Equal(st.BuiltInParams):
			fallthrough
		case !currentExtra.Equal(st.ExtraParams):
			cps = append(cps, n)
		}
	}
	return cps
}

// EtcdBootstrapped returns true if etcd cluster has been bootstrapped.
func (nf *NodeFilter) EtcdBootstrapped() bool {
	for _, n := range nf.cp {
		if nf.nodeStatus(n).Etcd.HasData {
			return true
		}
	}
	return false
}

// EtcdIsGood returns true if etcd cluster is responding and all members are in sync.
func (nf *NodeFilter) EtcdIsGood() bool {
	st := nf.status.Etcd
	if !st.IsHealthy {
		return false
	}
	for k := range st.Members {
		if !st.InSyncMembers[k] {
			return false
		}
	}
	return true
}

// EtcdStoppedMembers returns control plane nodes that are not running etcd.
func (nf *NodeFilter) EtcdStoppedMembers() (nodes []*cke.Node) {
	for _, n := range nf.cp {
		if _, ok := nf.status.Etcd.Members[n.Address]; !ok && nf.status.Etcd.IsHealthy {
			continue
		}
		st := nf.nodeStatus(n).Etcd
		if st.Running {
			continue
		}
		if !st.IsAddedMember {
			continue
		}
		nodes = append(nodes, n)
	}
	return nodes
}

// EtcdNonClusterMembers returns etcd members whose IDs are not defined in cluster YAML.
func (nf *NodeFilter) EtcdNonClusterMembers(healthy bool) (members []*etcdserverpb.Member) {
	st := nf.status.Etcd
	for k, v := range st.Members {
		if nf.InCluster(k) {
			continue
		}
		if st.InSyncMembers[k] != healthy {
			continue
		}
		members = append(members, v)
	}
	return members
}

// EtcdNonCPMembers returns nodes and IDs of etcd members running on
// non control plane nodes.  The order of ids matches the order of nodes.
func (nf *NodeFilter) EtcdNonCPMembers(healthy bool) (nodes []*cke.Node, ids []uint64) {
	st := nf.status.Etcd
	for k, v := range st.Members {
		n, ok := nf.nodeMap[k]
		if !ok {
			continue
		}
		if n.ControlPlane {
			continue
		}
		if st.InSyncMembers[k] != healthy {
			continue
		}
		nodes = append(nodes, n)
		ids = append(ids, v.ID)
	}
	return nodes, ids
}

// EtcdUnstartedMembers returns nodes that are added to members but not really
// joined to the etcd cluster.  Such members need to be re-added.
func (nf *NodeFilter) EtcdUnstartedMembers() (nodes []*cke.Node) {
	st := nf.status.Etcd
	for k, v := range st.Members {
		n, ok := nf.nodeMap[k]
		if !ok {
			continue
		}
		if !n.ControlPlane {
			continue
		}
		if len(v.Name) > 0 {
			continue
		}
		nodes = append(nodes, n)
	}
	return nodes
}

// EtcdUnmarkedMembers returns nodes that are working as in-sync members
// of the etcd cluster but not marked as added members.
func (nf *NodeFilter) EtcdUnmarkedMembers() (nodes []*cke.Node) {
	st := nf.status.Etcd
	for k, v := range st.InSyncMembers {
		if !v {
			continue
		}
		n, ok := nf.nodeMap[k]
		if !ok {
			continue
		}
		if !n.ControlPlane {
			continue
		}
		if nf.nodeStatus(n).Etcd.IsAddedMember {
			continue
		}
		nodes = append(nodes, n)
	}
	return nodes
}

// EtcdNewMembers returns control plane nodes to be added to the etcd cluster.
func (nf *NodeFilter) EtcdNewMembers() (nodes []*cke.Node) {
	members := nf.status.Etcd.Members
	for _, n := range nf.cp {
		if _, ok := members[n.Address]; ok {
			continue
		}
		nodes = append(nodes, n)
	}
	return nodes
}

func etcdEqualParams(running, current cke.ServiceParams) bool {
	// NOTE ignore parameters starting with "--initial-" prefix.
	// There options are used only on starting etcd process at first time.
	var rarg, carg []string
	for _, s := range running.ExtraArguments {
		if !strings.HasPrefix(s, "--initial-") {
			rarg = append(rarg, s)
		}
	}
	for _, s := range current.ExtraArguments {
		if !strings.HasPrefix(s, "--initial-") {
			carg = append(carg, s)
		}
	}

	rparams := cke.ServiceParams{
		ExtraArguments: rarg,
		ExtraBinds:     running.ExtraBinds,
		ExtraEnvvar:    running.ExtraEnvvar,
	}
	cparams := cke.ServiceParams{
		ExtraArguments: carg,
		ExtraBinds:     current.ExtraBinds,
		ExtraEnvvar:    current.ExtraEnvvar,
	}
	return rparams.Equal(cparams)
}

// EtcdOutdatedMembers returns nodes that are running etcd with outdated image or params.
func (nf *NodeFilter) EtcdOutdatedMembers() (nodes []*cke.Node) {
	currentExtra := nf.cluster.Options.Etcd.ServiceParams

	for _, n := range nf.cp {
		st := nf.nodeStatus(n).Etcd
		if !st.Running {
			continue
		}
		currentBuiltIn := etcd.BuiltInParams(n, []string{}, "new")
		switch {
		case cke.EtcdImage.Name() != st.Image:
			fallthrough
		case !etcdEqualParams(st.BuiltInParams, currentBuiltIn):
			fallthrough
		case !etcdEqualParams(st.ExtraParams, currentExtra):
			nodes = append(nodes, n)
		}
	}
	return nodes
}

// APIServerStoppedNodes returns control plane nodes that are not running API server.
func (nf *NodeFilter) APIServerStoppedNodes() (nodes []*cke.Node) {
	for _, n := range nf.cp {
		if !nf.nodeStatus(n).APIServer.Running {
			nodes = append(nodes, n)
		}
	}
	return nodes
}

// APIServerOutdatedNodes returns nodes that are running API server with outdated image or params.
func (nf *NodeFilter) APIServerOutdatedNodes() (nodes []*cke.Node) {
	currentExtra := nf.cluster.Options.APIServer
	kubeletConfig := k8s.GenerateKubeletConfiguration(nf.cluster.Options.Kubelet, "0.0.0.0", nil)

	for _, n := range nf.cp {
		st := nf.nodeStatus(n).APIServer
		currentBuiltIn := k8s.APIServerParams(n.Address, nf.cluster.ServiceSubnet,
			currentExtra.AuditLogEnabled, currentExtra.AuditLogPolicy, currentExtra.AuditLogPath, kubeletConfig.ClusterDomain)
		switch {
		case !st.Running:
			// stopped nodes are excluded
		case cke.KubernetesImage.Name() != st.Image:
			fallthrough
		case !currentBuiltIn.Equal(st.BuiltInParams):
			fallthrough
		case !currentExtra.Equal(st.ExtraParams):
			nodes = append(nodes, n)
		}
	}
	return nodes
}

// ControllerManagerStoppedNodes returns control plane nodes that are not running controller manager.
func (nf *NodeFilter) ControllerManagerStoppedNodes() (nodes []*cke.Node) {
	for _, n := range nf.cp {
		if !nf.nodeStatus(n).ControllerManager.Running {
			nodes = append(nodes, n)
		}
	}
	return nodes
}

// ControllerManagerOutdatedNodes returns nodes that are running controller manager with outdated image or params.
func (nf *NodeFilter) ControllerManagerOutdatedNodes() (nodes []*cke.Node) {
	currentBuiltIn := k8s.ControllerManagerParams(nf.cluster.Name, nf.cluster.ServiceSubnet)
	currentExtra := nf.cluster.Options.ControllerManager

	for _, n := range nf.cp {
		st := nf.nodeStatus(n).ControllerManager
		switch {
		case !st.Running:
			// stopped nodes are excluded
		case cke.KubernetesImage.Name() != st.Image:
			fallthrough
		case !currentBuiltIn.Equal(st.BuiltInParams):
			fallthrough
		case !currentExtra.Equal(st.ExtraParams):
			nodes = append(nodes, n)
		}
	}
	return nodes
}

// SchedulerStoppedNodes returns control plane nodes that are not running kube-scheduler.
func (nf *NodeFilter) SchedulerStoppedNodes() (nodes []*cke.Node) {
	for _, n := range nf.cp {
		if !nf.nodeStatus(n).Scheduler.Running {
			nodes = append(nodes, n)
		}
	}
	return nodes
}

func (nf *NodeFilter) SchedulerOutdatedNodes(params cke.SchedulerParams) (nodes []*cke.Node) {
	currentBuiltIn := k8s.SchedulerParams()
	currentExtra := nf.cluster.Options.Scheduler
	currentConfig := k8s.GenerateSchedulerConfiguration(params)

	for _, n := range nf.cp {
		st := nf.nodeStatus(n).Scheduler
		runningConfig := st.Config

		switch {
		case !st.Running:
			// stopped nodes are excluded
		case cke.KubernetesImage.Name() != st.Image:
			fallthrough
		case !currentBuiltIn.Equal(st.BuiltInParams):
			fallthrough
		case !currentExtra.ServiceParams.Equal(st.ExtraParams):
			fallthrough
		case !equality.Semantic.DeepEqual(currentConfig, runningConfig):
			log.Debug("kube-scheduler outdated", map[string]interface{}{
				"node":                 n.Nodename(),
				"st_builtin_args":      st.BuiltInParams.ExtraArguments,
				"st_builtin_env":       st.BuiltInParams.ExtraEnvvar,
				"st_extra_args":        st.ExtraParams.ExtraArguments,
				"st_extra_env":         st.ExtraParams.ExtraEnvvar,
				"current_builtin_args": currentBuiltIn.ExtraArguments,
				"current_builtin_env":  currentBuiltIn.ExtraEnvvar,
				"current_extra_args":   currentExtra.ExtraArguments,
				"current_extra_env":    currentExtra.ExtraEnvvar,
				"config":               currentConfig,
				"diff":                 cmp.Diff(currentConfig, runningConfig),
			})
			nodes = append(nodes, n)
		}
	}
	return nodes
}

// KubeletStoppedNodes returns nodes that are not running kubelet.
func (nf *NodeFilter) KubeletStoppedNodes() (nodes []*cke.Node) {
	for _, n := range nf.cluster.Nodes {
		if !nf.nodeStatus(n).Kubelet.Running {
			nodes = append(nodes, n)
		}
	}
	return nodes
}

// RegisteredNodes returns nodes that are registered on Kubernetes out of targets.
func (nf *NodeFilter) RegisteredNodes(target []*cke.Node) (nodes []*cke.Node) {
	registered := make(map[string]bool)
	for _, kn := range nf.status.Kubernetes.Nodes {
		registered[kn.Name] = true
	}

	for _, n := range target {
		if registered[n.Nodename()] {
			nodes = append(nodes, n)
		}
	}
	return nodes
}

// KubeletOutdatedNodes returns nodes that are running kubelet with outdated image or params.
func (nf *NodeFilter) KubeletOutdatedNodes() (nodes []*cke.Node) {
	currentOpts := nf.cluster.Options.Kubelet
	currentExtra := nf.cluster.Options.Kubelet.ServiceParams

	for _, n := range nf.cluster.Nodes {
		st := nf.nodeStatus(n).Kubelet
		currentConfig := k8s.GenerateKubeletConfiguration(currentOpts, n.Address, st.Config)
		currentBuiltIn := k8s.KubeletServiceParams(n, currentOpts)
		runningConfig := st.Config

		switch {
		case !st.Running:
			// stopped nodes are excluded
		case kubeletRuntimeChanged(st.BuiltInParams, currentBuiltIn):
			log.Warn("kubelet's container runtime cannot be changed", nil)
		case cke.KubernetesImage.Name() != st.Image:
			fallthrough
		case !currentBuiltIn.Equal(st.BuiltInParams):
			fallthrough
		case !currentExtra.Equal(st.ExtraParams):
			fallthrough
		case !equality.Semantic.DeepEqual(currentConfig, runningConfig):
			log.Debug("kubelet outdated", map[string]interface{}{
				"node":                 n.Nodename(),
				"st_builtin_args":      st.BuiltInParams.ExtraArguments,
				"st_builtin_env":       st.BuiltInParams.ExtraEnvvar,
				"st_extra_args":        st.ExtraParams.ExtraArguments,
				"st_extra_env":         st.ExtraParams.ExtraEnvvar,
				"current_builtin_args": currentBuiltIn.ExtraArguments,
				"current_builtin_env":  currentBuiltIn.ExtraEnvvar,
				"current_extra_args":   currentExtra.ExtraArguments,
				"current_extra_env":    currentExtra.ExtraEnvvar,
				"config":               currentConfig,
				"diff":                 cmp.Diff(currentConfig, runningConfig),
			})
			nodes = append(nodes, n)
		}
	}
	return nodes
}

// KubeletUnrecognizedNodes returns nodes of which kubelet is still running but not recognized by k8s.
func (nf *NodeFilter) KubeletUnrecognizedNodes() (nodes []*cke.Node) {
	for _, n := range nf.cluster.Nodes {
		if nf.nodeStatus(n).Kubelet.Running && !nf.existsNodeResource(n.Nodename()) {
			nodes = append(nodes, n)
		}
	}
	return nodes
}

func (nf *NodeFilter) existsNodeResource(name string) bool {
	for _, kn := range nf.status.Kubernetes.Nodes {
		if kn.Name == name {
			return true
		}
	}
	return false
}

// NonClusterNodes returns nodes not defined in cluster YAML.
func (nf *NodeFilter) NonClusterNodes() (nodes []*corev1.Node) {
	members := nf.status.Kubernetes.Nodes
	for _, member := range members {
		address, ok := nf.addressMap[member.Name]
		if !ok {
			address = member.Name
		}
		if nf.InCluster(address) {
			continue
		}
		member := member
		nodes = append(nodes, &member)
	}
	return nodes
}

func kubeletRuntimeChanged(running, current cke.ServiceParams) bool {
	runningRuntimeEndpoint := ""
	for _, arg := range running.ExtraArguments {
		if strings.HasPrefix(arg, "--container-runtime-endpoint=") {
			runningRuntimeEndpoint = arg
			break
		}
	}

	currentRuntimeEndpoint := ""
	for _, arg := range current.ExtraArguments {
		if strings.HasPrefix(arg, "--container-runtime-endpoint=") {
			currentRuntimeEndpoint = arg
			break
		}
	}
	return runningRuntimeEndpoint != currentRuntimeEndpoint
}

// ProxyStoppedNodes returns nodes that are not running kube-proxy.
func (nf *NodeFilter) ProxyStoppedNodes() (nodes []*cke.Node) {
	if nf.cluster.Options.Proxy.Disable {
		return nil
	}

	for _, n := range nf.cluster.Nodes {
		if !nf.nodeStatus(n).Proxy.Running {
			nodes = append(nodes, n)
		}
	}
	return nodes
}

// ProxyRunningUnexpectedlyNodes returns nodes that are running kube-proxy unexpectedly.
func (nf *NodeFilter) ProxyRunningUnexpectedlyNodes() (nodes []*cke.Node) {
	if !nf.cluster.Options.Proxy.Disable {
		return nil
	}

	for _, n := range nf.cluster.Nodes {
		if nf.nodeStatus(n).Proxy.Running {
			nodes = append(nodes, n)
		}
	}
	return nodes
}

// ProxyOutdatedNodes returns nodes that are running kube-proxy with outdated image or params.
func (nf *NodeFilter) ProxyOutdatedNodes(params cke.ProxyParams) (nodes []*cke.Node) {
	if nf.cluster.Options.Proxy.Disable {
		return nil
	}
	currentExtra := nf.cluster.Options.Proxy

	for _, n := range nf.cluster.Nodes {
		st := nf.nodeStatus(n).Proxy
		currentBuiltIn := k8s.ProxyParams()
		currentConfig := k8s.GenerateProxyConfiguration(params, n)
		runningConfig := st.Config
		switch {
		case !st.Running:
			// stopped nodes are excluded
		case cke.KubernetesImage.Name() != st.Image:
			fallthrough
		case !currentBuiltIn.Equal(st.BuiltInParams):
			fallthrough
		case !currentExtra.Equal(st.ExtraParams):
			fallthrough
		case !equality.Semantic.DeepEqual(currentConfig, runningConfig):
			log.Debug("proxy outdated", map[string]interface{}{
				"node":                 n.Nodename(),
				"st_builtin_args":      st.BuiltInParams.ExtraArguments,
				"st_builtin_env":       st.BuiltInParams.ExtraEnvvar,
				"st_extra_args":        st.ExtraParams.ExtraArguments,
				"st_extra_env":         st.ExtraParams.ExtraEnvvar,
				"current_builtin_args": currentBuiltIn.ExtraArguments,
				"current_builtin_env":  currentBuiltIn.ExtraEnvvar,
				"current_extra_args":   currentExtra.ExtraArguments,
				"current_extra_env":    currentExtra.ExtraEnvvar,
				"config":               currentConfig,
				"diff":                 cmp.Diff(currentConfig, runningConfig),
			})
			nodes = append(nodes, n)
		}
	}
	return nodes
}

// HealthyAPIServer returns a control plane node running healthy API server.
// If there is no healthy API server, it returns the first control plane node.
func (nf *NodeFilter) HealthyAPIServer() *cke.Node {
	var node *cke.Node
	for _, n := range nf.ControlPlane() {
		node = n
		if nf.nodeStatus(n).APIServer.IsHealthy {
			break
		}
	}
	return node
}

// HealthyAPIServerNodes returns nodes which have healthy API servers
func (nf *NodeFilter) HealthyAPIServerNodes() (nodes []*cke.Node) {
	for _, n := range nf.ControlPlane() {
		if nf.nodeStatus(n).APIServer.IsHealthy {
			nodes = append(nodes, n)
		}
	}
	return nodes
}

// UnhealthyAPIServerNodes returns nodes which have unhealthy API servers
func (nf *NodeFilter) UnhealthyAPIServerNodes() (nodes []*cke.Node) {
	for _, n := range nf.ControlPlane() {
		if !nf.nodeStatus(n).APIServer.IsHealthy {
			nodes = append(nodes, n)
		}
	}
	return nodes
}

func isInternal(name string, n *cke.Node) bool {
	if name == op.CKEAnnotationReboot {
		return false
	}
	if strings.HasPrefix(name, "cke.cybozu.com/") {
		return true
	}
	if strings.Contains(name, ".cke.cybozu.com/") {
		return true
	}
	for k := range n.Labels {
		if strings.HasPrefix(k, "node-role.kubernetes.io/") && name == k {
			return true
		}
	}
	if name == "node-role.kubernetes.io/master" || name == "node-role.kubernetes.io/control-plane" {
		return true
	}
	return false
}

// OutdatedAttrsNodes returns nodes that have outdated set of labels,
// attributes, and/or taints.
func (nf *NodeFilter) OutdatedAttrsNodes() (nodes []*corev1.Node) {
	curNodes := make(map[string]*corev1.Node)
	for _, cn := range nf.status.Kubernetes.Nodes {
		curNodes[cn.Name] = cn.DeepCopy()
	}

	for _, n := range nf.cluster.Nodes {
		current, ok := curNodes[n.Nodename()]
		if !ok {
			log.Warn("missing Kubernetes Node resource", map[string]interface{}{
				"name":    n.Nodename(),
				"address": n.Address,
			})
			continue
		}

		if nodeIsOutdated(n, current, nf.cluster.TaintCP) {
			labels := make(map[string]string)
			for k, v := range current.Labels {
				if isInternal(k, n) {
					continue
				}
				labels[k] = v
			}
			for k, v := range n.Labels {
				labels[k] = v
			}
			if n.ControlPlane {
				labels[op.CKELabelMaster] = "true"
			}
			current.Labels = labels

			annotations := make(map[string]string)
			for k, v := range current.Annotations {
				if isInternal(k, n) {
					continue
				}
				annotations[k] = v
			}
			for k, v := range n.Annotations {
				annotations[k] = v
			}
			current.Annotations = annotations

			nTaints := make(map[string]bool)
			for _, taint := range n.Taints {
				nTaints[taint.Key] = true
			}
			taints := make([]corev1.Taint, len(n.Taints))
			copy(taints, n.Taints)
			for _, taint := range current.Spec.Taints {
				if isInternal(taint.Key, n) || nTaints[taint.Key] {
					continue
				}
				taints = append(taints, taint)
			}
			if nf.cluster.TaintCP && n.ControlPlane {
				taints = append(taints, corev1.Taint{
					Key:    op.CKETaintMaster,
					Effect: corev1.TaintEffectPreferNoSchedule,
				})
			}
			current.Spec.Taints = taints

			nodes = append(nodes, current)
		}
	}

	return nodes
}

func nodeIsOutdated(n *cke.Node, current *corev1.Node, taintCP bool) bool {
	for k, v := range n.Labels {
		cv, ok := current.Labels[k]
		if !ok || v != cv {
			return true
		}
	}
	// Labels for CKE internal use need to be synchronized.
	for k := range current.Labels {
		if !isInternal(k, n) {
			continue
		}
		if k == op.CKELabelMaster {
			continue
		}
		if _, ok := n.Labels[k]; !ok {
			return true
		}
	}

	if n.ControlPlane {
		cv, ok := current.Labels[op.CKELabelMaster]
		if !ok || cv != "true" {
			return true
		}
	} else {
		if _, ok := current.Labels[op.CKELabelMaster]; ok {
			return true
		}
	}

	for k, v := range n.Annotations {
		cv, ok := current.Annotations[k]
		if !ok || v != cv {
			return true
		}
	}

	// Annotations for CKE internal use need to be synchronized.
	for k := range current.Annotations {
		if !isInternal(k, n) {
			continue
		}
		if _, ok := n.Annotations[k]; !ok {
			return true
		}
	}

	curTaints := make(map[string]corev1.Taint)
	for _, taint := range current.Spec.Taints {
		curTaints[taint.Key] = taint
	}
	for _, taint := range n.Taints {
		cv, ok := curTaints[taint.Key]
		if !ok {
			return true
		}
		if taint.Value != cv.Value {
			return true
		}
		if taint.Effect != cv.Effect {
			return true
		}
	}

	// Taints for CKE internal use need to be synchronized.
	nTaints := make(map[string]corev1.Taint)
	for _, taint := range n.Taints {
		nTaints[taint.Key] = taint
	}
	for _, taint := range current.Spec.Taints {
		if !isInternal(taint.Key, n) {
			continue
		}
		if taint.Key == op.CKETaintMaster {
			continue
		}
		if _, ok := nTaints[taint.Key]; !ok {
			return true
		}
	}

	if taintCP && n.ControlPlane {
		taint, ok := curTaints[op.CKETaintMaster]
		if !ok || taint.Effect != corev1.TaintEffectPreferNoSchedule {
			return true
		}
	} else {
		if _, ok := curTaints[op.CKETaintMaster]; ok {
			return true
		}
	}

	return false
}

// CordonedNodes returns nodes that are cordoned and annotated as reboot/repair operation targets.
func (nf *NodeFilter) CordonedNodes() (nodes []*corev1.Node) {
	for i := range nf.status.Kubernetes.Nodes {
		n := &nf.status.Kubernetes.Nodes[i]
		if n.Spec.Unschedulable && n.Annotations[op.CKEAnnotationReboot] == "true" {
			nodes = append(nodes, n)
		}
	}
	return nodes
}

// SSHNotConnectedNodes returns nodes that are not connected via SSH out of targets.
func (nf *NodeFilter) SSHNotConnectedNodes(targets []*cke.Node, includeControlPlane, includeWorker bool) (nodes []*cke.Node) {
	for _, n := range targets {
		if n.ControlPlane && !includeControlPlane {
			continue
		}
		if !n.ControlPlane && !includeWorker {
			continue
		}
		if nf.status.NodeStatuses[n.Address].SSHConnected {
			continue
		}

		nodes = append(nodes, n)
	}
	return nodes
}

// SSHConnectedNodes returns nodes that are connected via SSH out of targets
func (nf *NodeFilter) SSHConnectedNodes(targets []*cke.Node, includeControlPlane, includeWorker bool) (nodes []*cke.Node) {
	for _, n := range targets {
		if n.ControlPlane && !includeControlPlane {
			continue
		}
		if !n.ControlPlane && !includeWorker {
			continue
		}
		if !nf.status.NodeStatuses[n.Address].SSHConnected {
			continue
		}

		nodes = append(nodes, n)
	}
	return nodes
}
