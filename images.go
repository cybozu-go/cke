package cke

// Image is the type of container images.
type Image string

// Name returns docker image name.
func (i Image) Name() string {
	return string(i)
}

// Container image definitions
const (
	EtcdImage            = Image("ghcr.io/cybozu/etcd:3.5.10.2")
	KubernetesImage      = Image("ghcr.io/cybozu/kubernetes:1.27.10.1")
	ToolsImage           = Image("ghcr.io/cybozu-go/cke-tools:1.27.1")
	PauseImage           = Image("ghcr.io/cybozu/pause:3.9.0.4")
	CoreDNSImage         = Image("ghcr.io/cybozu/coredns:1.11.1.2")
	UnboundImage         = Image("ghcr.io/cybozu/unbound:1.18.0.2")
	UnboundExporterImage = Image("ghcr.io/cybozu/unbound_exporter:0.4.4.2")
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
