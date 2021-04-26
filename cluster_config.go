package cke

const (
	defaultEtcdVolumeName           = "etcd-cke"
	defaultContainerRuntimeEndpoint = "/run/containerd/containerd.sock"
)

// NewCluster creates Cluster
func NewCluster() *Cluster {
	return &Cluster{
		Options: Options{
			Etcd: EtcdParams{
				VolumeName: defaultEtcdVolumeName,
			},
			Kubelet: KubeletParams{
				CRIEndpoint: defaultContainerRuntimeEndpoint,
			},
		},
	}
}
