package cke

// Image is the type of container images.
type Image string

// Name returns docker image name.
func (i Image) Name() string {
	return string(i)
}

// Container image definitions
const (
	EtcdImage            = Image("ghcr.io/cybozu/etcd:3.5.14.1")
	KubernetesImage      = Image("ghcr.io/cybozu/kubernetes:1.28.10.1")
	ToolsImage           = Image("ghcr.io/cybozu-go/cke-tools:1.28.0")
	PauseImage           = Image("ghcr.io/cybozu/pause:3.9.0.5")
	CoreDNSImage         = Image("ghcr.io/cybozu/coredns:1.11.1.3")
	UnboundImage         = Image("ghcr.io/cybozu/unbound:1.19.3.1")
	UnboundExporterImage = Image("ghcr.io/cybozu/unbound_exporter:0.4.6.1")
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
