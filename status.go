package cke

import (
	"context"
	"net"
)

// ClusterStatus represents the working cluster status.
// The structure reflects Cluster, of course.
type ClusterStatus struct {
	Name          string
	NodeStatuses  map[string]*NodeStatus // keys are IP address strings.
	ServiceSubnet *net.IPNet
	RBAC          bool // true if RBAC is enabled

	// TODO:
	// CoreDNS will be deployed as k8s Pods.
	// We probably need to use k8s API to query CoreDNS service status.
}

// NodeStatus status of a node.
type NodeStatus struct {
	Address    net.IP
	Etcd       ServiceStatus
	APIServer  ServiceStatus
	Controller ServiceStatus
	Scheduler  ServiceStatus
	Proxy      ServiceStatus
	Kubelet    KubeletStatus
	Labels     map[string]string // are labels for k8s Node resource.
}

// IsControlPlane returns true if the node has been configured as a control plane.
func (ns *NodeStatus) IsControlPlane() bool {
	return ns.Etcd.Configured
}

// ServiceStatus represents statuses of a service.
//
// If Configured is false, the service is not yet configured on the node.
// If Running is false, the service is not running on the node.
// ExtraXX are extra parameters of the running service, if any.
type ServiceStatus struct {
	Configured       bool
	Running          bool
	ContainerVersion string
	ExtraArguments   map[string]string
	ExtraBinds       map[string]string
	ExtraEnvvar      map[string]string
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
