package cke

import (
	"context"
	"net"
	"path/filepath"
)

// ClusterStatus represents the working cluster status.
// The structure reflects Cluster, of course.
type ClusterStatus struct {
	Name          string
	NodeStatuses  map[string]*NodeStatus // keys are IP address strings.
	Agents        map[string]Agent       // ditto.
	ServiceSubnet *net.IPNet
	RBAC          bool // true if RBAC is enabled
	// TODO:
	// CoreDNS will be deployed as k8s Pods.
	// We probably need to use k8s API to query CoreDNS service status.
}

// Destroy calls Close for all agents.
func (cs *ClusterStatus) Destroy() {
	for _, a := range cs.Agents {
		a.Close()
	}
	cs.Agents = nil
}

// NodeStatus status of a node.
type NodeStatus struct {
	Etcd       EtcdStatus
	APIServer  ServiceStatus
	Controller ServiceStatus
	Scheduler  ServiceStatus
	Proxy      ServiceStatus
	Kubelet    KubeletStatus
	Labels     map[string]string // are labels for k8s Node resource.
}

// IsControlPlane returns true if the node has been configured as a control plane.
func (ns *NodeStatus) IsControlPlane() bool {
	return ns.Etcd.HasData
}

// ServiceStatus represents statuses of a service.
//
// If Running is false, the service is not running on the node.
// ExtraXX are extra parameters of the running service, if any.
type ServiceStatus struct {
	Running        bool
	Image          string
	ExtraArguments []string
	ExtraBinds     []Mount
	ExtraEnvvar    map[string]string
}

// EtcdStatus is the status of kubelet.
type EtcdStatus struct {
	ServiceStatus
	HasData bool
}

// KubeletStatus is the status of kubelet.
type KubeletStatus struct {
	ServiceStatus
	Domain    string
	AllowSwap bool
}

// GetClusterStatus consults the whole cluster and constructs *ClusterStatus.
func GetClusterStatus(ctx context.Context, cluster *Cluster) (*ClusterStatus, error) {
	// TODO
	return new(ClusterStatus), nil
}

func getNodeStatus(agent Agent, cluster *Cluster) (*NodeStatus, error) {
	status := &NodeStatus{}

	etcd := container{"etcd", agent}
	ss, err := etcd.inspect()
	if err != nil {
		return status, nil
	}

	dataDir := etcdDataDir(cluster)
	_, _, err = agent.Run("test -d " + filepath.Join(dataDir, "default.etcd"))
	status.Etcd = EtcdStatus{*ss, err == nil}

	return status, nil
}
