package op

import (
	"path/filepath"
	"time"
)

const (
	defaultEtcdVolumeName = "etcd-cke"

	etcdContainerName                  = "etcd"
	kubeAPIServerContainerName         = "kube-apiserver"
	kubeControllerManagerContainerName = "kube-controller-manager"
	kubeProxyContainerName             = "kube-proxy"
	kubeSchedulerContainerName         = "kube-scheduler"
	kubeletContainerName               = "kubelet"
	riversContainerName                = "rivers"

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
	// ClusterDNSLabelDomain is search key of Domain
	ClusterDNSLabelDomain = "cke.cybozu.com/domain"
	// ClusterDNSLabelDNSServers is search key of DNSServers
	ClusterDNSLabelDNSServers = "cke.cybozu.com/dns-servers"
)

// EtcdPKIPath returns a certificate file path for k8s.
func EtcdPKIPath(p string) string {
	return filepath.Join(etcdPKIPath, p)
}

// K8sPKIPath returns a certificate file path for k8s.
func K8sPKIPath(p string) string {
	return filepath.Join(k8sPKIPath, p)
}
