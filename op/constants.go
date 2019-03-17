package op

import (
	"path/filepath"
	"time"
)

const (
	etcdEndpointsName = "cke-etcd"

	etcdPKIPath = "/etc/etcd/pki"
	k8sPKIPath  = "/etc/kubernetes/pki"
)

const (
	// EtcdContainerName is container name of etcd
	EtcdContainerName = "etcd"
	// KubeAPIServerContainerName is name of kube-apiserver
	KubeAPIServerContainerName = "kube-apiserver"
	// KubeControllerManagerContainerName is name of kube-controller-manager
	KubeControllerManagerContainerName = "kube-controller-manager"
	// KubeProxyContainerName is container name of kube-proxy
	KubeProxyContainerName = "kube-proxy"
	// KubeSchedulerContainerName is container name of kube-scheduler
	KubeSchedulerContainerName = "kube-scheduler"
	// KubeletContainerName is container name of kubelet
	KubeletContainerName = "kubelet"
	// RiversContainerName is container name of rivers
	RiversContainerName = "rivers"

	// ClusterDNSRBACRoleName is role name of cluster DNS
	ClusterDNSRBACRoleName = "system:cluster-dns"
	// ClusterDNSAppName is app name of cluster DNS
	ClusterDNSAppName = "cluster-dns"
	// NodeDNSAppName is app name of node-dns
	NodeDNSAppName = "node-dns"

	// DefaultEtcdVolumeName is etcd default volume name
	DefaultEtcdVolumeName = "etcd-cke"

	// TimeoutDuration is default timeout duration
	TimeoutDuration = 5 * time.Second

	// CKELabelAppName is application name
	CKELabelAppName = "cke.cybozu.com/appname"
	// EtcdBackupAppName is application name for etcdbackup
	EtcdBackupAppName = "etcdbackup"
)

// EtcdPKIPath returns a certificate file path for k8s.
func EtcdPKIPath(p string) string {
	return filepath.Join(etcdPKIPath, p)
}

// K8sPKIPath returns a certificate file path for k8s.
func K8sPKIPath(p string) string {
	return filepath.Join(k8sPKIPath, p)
}
