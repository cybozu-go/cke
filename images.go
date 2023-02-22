package cke

// Image is the type of container images.
type Image string

// Name returns docker image name.
func (i Image) Name() string {
	return string(i)
}

// Container image definitions
const (
	EtcdImage            = Image("quay.io/cybozu/etcd:3.5.7.1")
	KubernetesImage      = Image("quay.io/cybozu/kubernetes:1.25.6.1")
	ToolsImage           = Image("quay.io/cybozu/cke-tools:1.25.0")
	PauseImage           = Image("quay.io/cybozu/pause:3.8.0.1")
	CoreDNSImage         = Image("quay.io/cybozu/coredns:1.10.0.1")
	UnboundImage         = Image("quay.io/cybozu/unbound:1.17.1.1")
	UnboundExporterImage = Image("quay.io/cybozu/unbound_exporter:0.4.1.4")
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
		UnboundExporterImage.Name(),
	}
}
