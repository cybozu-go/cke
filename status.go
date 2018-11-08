package cke

import (
	"github.com/coreos/etcd/etcdserver/etcdserverpb"

	corev1 "k8s.io/api/core/v1"
)

// EtcdClusterStatus is the status of the etcd cluster.
type EtcdClusterStatus struct {
	IsHealthy     bool
	Members       map[string]*etcdserverpb.Member
	InSyncMembers map[string]bool
}

// CoreDNSStatus contains CoreDNS status.
type CoreDNSStatus struct {
	ServiceAccountExists  bool
	RBACRoleExists        bool
	RBACRoleBindingExists bool
	ConfigMapExists       bool
	DeploymentExists      bool
	ServiceExists         bool
	ClusterDomain         string
	ClusterIP             string
}

// NodeDNSStatus contains node local resolver status.
type NodeDNSStatus struct {
	ConfigMapExists bool
	DaemonSetExists bool
	Config          string
}

// KubernetesClusterStatus contains kubernetes cluster configurations
type KubernetesClusterStatus struct {
	IsReady               bool
	Nodes                 []corev1.Node
	RBACRoleExists        bool
	RBACRoleBindingExists bool
	DNSServers            []string
	CoreDNS               CoreDNSStatus
	CoreDNSClusterDomain  string
	CoreDNSClusterIP      string
	NodeDNS               NodeDNSStatus
	EtcdEndpoints         *corev1.Endpoints
}

// ClusterStatus represents the working cluster status.
// The structure reflects Cluster, of course.
type ClusterStatus struct {
	Name         string
	NodeStatuses map[string]*NodeStatus // keys are IP address strings.

	Etcd       EtcdClusterStatus
	Kubernetes KubernetesClusterStatus
}

// NodeStatus status of a node.
type NodeStatus struct {
	Etcd              EtcdStatus
	Rivers            ServiceStatus
	APIServer         KubeComponentStatus
	ControllerManager KubeComponentStatus
	Scheduler         KubeComponentStatus
	Proxy             KubeComponentStatus
	Kubelet           KubeletStatus
	Labels            map[string]string // are labels for k8s Node resource.
}

// ServiceStatus represents statuses of a service.
//
// If Running is false, the service is not running on the node.
// ExtraXX are extra parameters of the running service, if any.
type ServiceStatus struct {
	Running       bool
	Image         string
	BuiltInParams ServiceParams
	ExtraParams   ServiceParams
}

// EtcdStatus is the status of kubelet.
type EtcdStatus struct {
	ServiceStatus
	HasData bool
}

// KubeComponentStatus represents service status and endpoint's health
type KubeComponentStatus struct {
	ServiceStatus
	IsHealthy bool
}

// KubeletStatus represents kubelet status and health
type KubeletStatus struct {
	ServiceStatus
	IsHealthy bool
	Domain    string
	DNS       string
	AllowSwap bool
}
