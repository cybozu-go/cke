package cke

// Container image definitions
const (
	EtcdImage      = "quay.io/cybozu/etcd:3.3.9-1"
	CKETools       = "quay.io/cybozu/cke-tools:0.1-2"
	ToolsImage     = "quay.io/cybozu/ubuntu:18.04"
	HyperkubeImage = "quay.io/cybozu/hyperkube:1.11.2-1"
	PauseImage     = "quay.io/cybozu/pause:3.1-1"
)

// Image returns the image name for a given container.
func Image(name string) string {
	switch name {
	case "etcd":
		return EtcdImage
	case "rivers":
		return CKETools
	case "tools":
		return ToolsImage
	case "kube-apiserver", "kube-controller-manager", "kube-proxy", "kube-scheduler", "kubelet", "hyperkube":
		return HyperkubeImage
	case "pause":
		return PauseImage
	}

	panic("no such image: " + name)
}

// AllImages return container images list used by CKE
func AllImages() []string {
	return []string{
		EtcdImage,
		CKETools,
		ToolsImage,
		HyperkubeImage,
		PauseImage,
	}
}
