package server

import (
	"strings"

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/cke/op"
)

// NodeFilter filters nodes to
type NodeFilter struct {
	cluster *cke.Cluster
	status  *cke.ClusterStatus
	nodeMap map[string]*cke.Node
	cp      []*cke.Node
}

// NewNodeFilter creates and initializes NodeFilter.
func NewNodeFilter(cluster *cke.Cluster, status *cke.ClusterStatus) *NodeFilter {
	nodeMap := make(map[string]*cke.Node)
	cp := make([]*cke.Node, 0, 5)

	for _, n := range cluster.Nodes {
		nodeMap[n.Address] = n
		if n.ControlPlane {
			cp = append(cp, n)
		}
	}

	return &NodeFilter{
		cluster: cluster,
		status:  status,
		nodeMap: nodeMap,
		cp:      cp,
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
	currentBuiltIn := op.RiversParams(nf.cp)
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
	return len(st.Members) == len(st.InSyncMembers)
}

// EtcdStoppedMembers returns control plane nodes that are not running etcd.
func (nf *NodeFilter) EtcdStoppedMembers() (nodes []*cke.Node) {
	for _, n := range nf.cp {
		st := nf.nodeStatus(n).Etcd
		if st.Running {
			continue
		}
		if !st.HasData {
			continue
		}
		nodes = append(nodes, n)
	}
	return nodes
}

// EtcdNonClusterMemberIDs returns IDs of etcd members not defined in cluster YAML.
func (nf *NodeFilter) EtcdNonClusterMemberIDs(healthy bool) (ids []uint64) {
	st := nf.status.Etcd
	for k, v := range st.Members {
		if nf.InCluster(k) {
			continue
		}
		if st.InSyncMembers[k] != healthy {
			continue
		}
		ids = append(ids, v.ID)
	}
	return ids
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
		currentBuiltIn := op.EtcdBuiltInParams(n, []string{}, "new")
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

	for _, n := range nf.cp {
		st := nf.nodeStatus(n).APIServer
		currentBuiltIn := op.APIServerParams(nf.ControlPlane(), n.Address, nf.cluster.ServiceSubnet)
		switch {
		case !st.Running:
			// stopped nodes are excluded
		case cke.HyperkubeImage.Name() != st.Image:
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
	currentBuiltIn := op.ControllerManagerParams(nf.cluster.Name, nf.cluster.ServiceSubnet)
	currentExtra := nf.cluster.Options.ControllerManager

	for _, n := range nf.cp {
		st := nf.nodeStatus(n).ControllerManager
		switch {
		case !st.Running:
			// stopped nodes are excluded
		case cke.HyperkubeImage.Name() != st.Image:
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

// SchedulerOutdatedNodes returns nodes that are running kube-scheduler with outdated image or params.
func (nf *NodeFilter) SchedulerOutdatedNodes() (nodes []*cke.Node) {
	currentBuiltIn := op.SchedulerParams()
	currentExtra := nf.cluster.Options.Scheduler

	for _, n := range nf.cp {
		st := nf.nodeStatus(n).Scheduler
		switch {
		case !st.Running:
			// stopped nodes are excluded
		case cke.HyperkubeImage.Name() != st.Image:
			fallthrough
		case !currentBuiltIn.Equal(st.BuiltInParams):
			fallthrough
		case !currentExtra.Equal(st.ExtraParams):
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

// KubeletOutdatedNodes returns nodes that are running kubelet with outdated image or params.
func (nf *NodeFilter) KubeletOutdatedNodes() (nodes []*cke.Node) {
	currentOpts := nf.cluster.Options.Kubelet
	currentExtra := nf.cluster.Options.Kubelet.ServiceParams

	for _, n := range nf.cluster.Nodes {
		st := nf.nodeStatus(n).Kubelet
		currentBuiltIn := op.KubeletServiceParams(n)
		switch {
		case !st.Running:
			// stopped nodes are excluded
		case cke.HyperkubeImage.Name() != st.Image:
			fallthrough
		case currentOpts.Domain != st.Domain:
			fallthrough
		case currentOpts.AllowSwap != st.AllowSwap:
			fallthrough
		case !kubeletEqualParams(st.BuiltInParams, currentBuiltIn):
			fallthrough
		case !currentExtra.Equal(st.ExtraParams):
			nodes = append(nodes, n)
		}
	}
	return nodes
}

func kubeletEqualParams(running, current cke.ServiceParams) bool {
	// NOTE ignore parameter "--register-with-taints".
	// This option is used only when kubelet registers the node first time.
	var rarg []string
	for _, s := range running.ExtraArguments {
		if !strings.HasPrefix(s, "--register-with-taints") {
			rarg = append(rarg, s)
		}
	}

	running.ExtraArguments = rarg
	return running.Equal(current)
}

// ProxyStoppedNodes returns nodes that are not running kube-proxy.
func (nf *NodeFilter) ProxyStoppedNodes() (nodes []*cke.Node) {
	for _, n := range nf.cluster.Nodes {
		if !nf.nodeStatus(n).Proxy.Running {
			nodes = append(nodes, n)
		}
	}
	return nodes
}

// ProxyOutdatedNodes returns nodes that are running kube-proxy with outdated image or params.
func (nf *NodeFilter) ProxyOutdatedNodes() (nodes []*cke.Node) {
	currentBuiltIn := op.ProxyParams()
	currentExtra := nf.cluster.Options.Proxy

	for _, n := range nf.cluster.Nodes {
		st := nf.nodeStatus(n).Proxy
		switch {
		case !st.Running:
			// stopped nodes are excluded
		case cke.HyperkubeImage.Name() != st.Image:
			fallthrough
		case !currentBuiltIn.Equal(st.BuiltInParams):
			fallthrough
		case !currentExtra.Equal(st.ExtraParams):
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
