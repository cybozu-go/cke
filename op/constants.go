package op

import (
	"path/filepath"
	"time"
)

const (
	defaultEtcdVolumeName = "etcd-cke"

	EtcdContainerName                  = "etcd"
	KubeAPIServerContainerName         = "kube-apiserver"
	KubeControllerManagerContainerName = "kube-controller-manager"
	KubeProxyContainerName             = "kube-proxy"
	KubeSchedulerContainerName         = "kube-scheduler"
	KubeletContainerName               = "kubelet"
	RiversContainerName                = "rivers"

	rbacRoleName        = "system:kube-apiserver-to-kubelet"
	rbacRoleBindingName = "system:kube-apiserver"

	etcdEndpointsName = "cke-etcd"

	clusterDNSRBACRoleName = "system:cluster-dns"
	clusterDNSAppName      = "cluster-dns"

	nodeDNSAppName = "node-dns"

	timeoutDuration = 5 * time.Second

	etcdPKIPath = "/etc/etcd/pki"
	k8sPKIPath  = "/etc/kubernetes/pki"
)

const (
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
