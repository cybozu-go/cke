package cke

// Image is the type of container images.
type Image string

// Name returns docker image name.
func (i Image) Name() string {
	return string(i)
}

// Container image definitions
const (
	EtcdImage       = Image("quay.io/cybozu/etcd:3.3.25.3")
	KubernetesImage = Image("quay.io/cybozu/kubernetes:1.19.7.2")
	ToolsImage      = Image("quay.io/cybozu/cke-tools:1.19.0")
	PauseImage      = Image("quay.io/cybozu/pause:3.2.0.2")
	CoreDNSImage    = Image("quay.io/cybozu/coredns:1.8.0.1")
	UnboundImage    = Image("quay.io/cybozu/unbound:1.13.0.1")
)

// AllImages return container images list used by CKE
func AllImages() []string {
	return []string{
		EtcdImage.Name(),
		ToolsImage.Name(),
		KubernetesImage.Name(),
		PauseImage.Name(),
		CoreDNSImage.Name(),
		UnboundImage.Name(),
	}
}
