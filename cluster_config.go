package cke

const (
	defaultEtcdVolumeName = "etcd-cke"
	defaultClusterDomain  = "cluster.local"
)

// NewCluster creates Cluster
func NewCluster() *Cluster {
	return &Cluster{
		Options: Options{
			Etcd: EtcdParams{
				VolumeName: defaultEtcdVolumeName,
			},
			Kubelet: KubeletParams{
				Domain: defaultClusterDomain,
			},
		},
	}
}
