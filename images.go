package cke

// Image is the type of container images.
type Image string

// Name returns docker image name.
func (i Image) Name() string {
	return string(i)
}

// Container image definitions
const (
	EtcdImage      = Image("quay.io/cybozu/etcd:3.3.9-2")
	ToolsImage     = Image("quay.io/cybozu/cke-tools:1.1.1-1")
	HyperkubeImage = Image("quay.io/cybozu/hyperkube:1.11.2-2")
	PauseImage     = Image("quay.io/cybozu/pause:3.1-1")
)

// AllImages return container images list used by CKE
func AllImages() []string {
	return []string{
		EtcdImage.Name(),
		ToolsImage.Name(),
		HyperkubeImage.Name(),
		PauseImage.Name(),
	}
}
