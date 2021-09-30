package cke

// Image is the type of container images.
type Image string

// Name returns docker image name.
func (i Image) Name() string {
	return string(i)
}

// Container image definitions
const (
	EtcdImage       = Image("quay.io/cybozu/etcd:3.4.16.2")
	KubernetesImage = Image("quay.io/cybozu/kubernetes:1.21.5.1")
	ToolsImage      = Image("quay.io/cybozu/cke-tools:1.21.0")
	PauseImage      = Image("quay.io/cybozu/pause:3.6.0.1")
	CoreDNSImage    = Image("quay.io/cybozu/coredns:1.8.5.1")
	UnboundImage    = Image("quay.io/cybozu/unbound:1.13.2.1")
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
