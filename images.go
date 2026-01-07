package cke

// Image is the type of container images.
type Image string

// Name returns docker image name.
func (i Image) Name() string {
	return string(i)
}

// Container image definitions
const (
	EtcdImage            = Image("ghcr.io/cybozu/etcd:3.6.7.1")
	KubernetesImage      = Image("ghcr.io/cybozu/kubernetes:1.32.7.1")
	ToolsImage           = Image("ghcr.io/cybozu-go/cke-tools:1.32.0")
	PauseImage           = Image("ghcr.io/cybozu/pause:3.10.1.2")
	CoreDNSImage         = Image("ghcr.io/cybozu/coredns:1.13.2.1")
	UnboundImage         = Image("ghcr.io/cybozu/unbound:1.24.1.1")
	UnboundExporterImage = Image("ghcr.io/cybozu/unbound_exporter:0.4.6.3")
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
