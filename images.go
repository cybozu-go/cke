package cke

// Image is the type of container images.
type Image string

// Name returns docker image name.
func (i Image) Name() string {
	return string(i)
}

// Container image definitions
const (
	EtcdImage      = Image("quay.io/cybozu/etcd:3.3.18.1")
	HyperkubeImage = Image("quay.io/cybozu/hyperkube:1.16.4.1")
	ToolsImage     = Image("quay.io/cybozu/cke-tools:1.6.4")
	PauseImage     = Image("quay.io/cybozu/pause:3.1.0.3")
	CoreDNSImage   = Image("quay.io/cybozu/coredns:1.6.6.1")
	UnboundImage   = Image("quay.io/cybozu/unbound:1.9.5.1")
)

// AllImages return container images list used by CKE
func AllImages() []string {
	return []string{
		EtcdImage.Name(),
		ToolsImage.Name(),
		HyperkubeImage.Name(),
		PauseImage.Name(),
		CoreDNSImage.Name(),
		UnboundImage.Name(),
	}
}
