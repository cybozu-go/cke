package cke

const (
	defaultEtcdVolumeName           = "etcd-cke"
	defaultContainerRuntime         = "remote"
	defaultContainerRuntimeEndpoint = "/run/containerd/containerd.sock"
	defaultEtcdBackupRotate         = 14
)

// NewCluster creates Cluster
func NewCluster() *Cluster {
	return &Cluster{
		Options: Options{
			Etcd: EtcdParams{
				VolumeName: defaultEtcdVolumeName,
			},
			Kubelet: KubeletParams{
				ContainerRuntime: defaultContainerRuntime,
				CRIEndpoint:      defaultContainerRuntimeEndpoint,
			},
		},
		EtcdBackup: EtcdBackup{
			Rotate: defaultEtcdBackupRotate,
		},
	}
}
