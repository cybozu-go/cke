package cke

// Container image definitions
const (
	EtcdImage      = "quay.io/cybozu/etcd:3.3.5-1"
	CKETools       = "quay.io/cybozu/cke-tools:0.1-2"
	ToolsImage     = "quay.io/cybozu/ubuntu:18.04"
	HyperkubeImage = "gcr.io/googole_containers/hyperkube:1.11.1"
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
	case "kube-apiserver", "kube-controller-manager", "kube-scheduler", "kubelet", "kube-proxy":
		return HyperkubeImage
	}

	panic("no such image: " + name)
}
