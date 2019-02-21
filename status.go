package cke

import (
	"github.com/coreos/etcd/etcdserver/etcdserverpb"

	appsv1 "k8s.io/api/apps/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
)

// EtcdClusterStatus is the status of the etcd cluster.
type EtcdClusterStatus struct {
	IsHealthy     bool
	Members       map[string]*etcdserverpb.Member
	InSyncMembers map[string]bool
}

// ClusterDNSStatus contains cluster resolver status.
type ClusterDNSStatus struct {
	ServiceAccountExists  bool
	RBACRoleExists        bool
	RBACRoleBindingExists bool
	ConfigMap             *corev1.ConfigMap
	Deployment            *appsv1.Deployment
	ServiceExists         bool
	ClusterDomain         string
	ClusterIP             string
}

// NodeDNSStatus contains node local resolver status.
type NodeDNSStatus struct {
	DaemonSet *appsv1.DaemonSet
	ConfigMap *corev1.ConfigMap
}

// EtcdBackupStatus is the status of etcdbackup
type EtcdBackupStatus struct {
	ConfigMap *corev1.ConfigMap
	CronJob   *batchv1beta1.CronJob
	Pod       *corev1.Pod
	Secret    *corev1.Secret
	Service   *corev1.Service
}

// KubernetesClusterStatus contains kubernetes cluster configurations
type KubernetesClusterStatus struct {
	IsReady               bool
	Nodes                 []corev1.Node
	RBACRoleExists        bool
	RBACRoleBindingExists bool
	DNSService            *corev1.Service
	ClusterDNS            ClusterDNSStatus
	NodeDNS               NodeDNSStatus
	EtcdEndpoints         *corev1.Endpoints
	EtcdBackup            EtcdBackupStatus
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
	IsHealthy            bool
	Domain               string
	AllowSwap            bool
	ContainerLogMaxSize  string
	ContainerLogMaxFiles int32
}
