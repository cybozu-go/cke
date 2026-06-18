package cke

//go:generate go run ./pkg/update-images/

// Image represents a container image reference.
type Image struct {
	fullRef string
	tagRef  string
}

func newImage(repository, tag, digest string) Image {
	return Image{
		fullRef: repository + ":" + tag + "@" + digest,
		tagRef:  repository + ":" + tag,
	}
}

// FullRef returns the full image reference (repository:tag@digest).
func (i Image) FullRef() string {
	return i.fullRef
}

// TagRef returns the repository:tag reference without the digest.
func (i Image) TagRef() string {
	return i.tagRef
}

// AllImages return container images list used by CKE
func AllImages() []string {
	return []string{
		EtcdImage.FullRef(),
		ToolsImage.FullRef(),
		KubernetesImage.FullRef(),
		PauseImage.FullRef(),
		CoreDNSImage.FullRef(),
		UnboundImage.FullRef(),
		UnboundExporterImage.FullRef(),
	}
}
