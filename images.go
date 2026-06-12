package cke

import "strings"

//go:generate go run ./pkg/update-images/

// Image is the type of container images.
type Image string

// Name returns the full image reference.
func (i Image) Name() string {
	return string(i)
}

// Repository returns the repository part of the image reference (without tag and digest).
func (i Image) Repository() string {
	name := string(i)
	if idx := strings.Index(name, "@"); idx >= 0 {
		name = name[:idx]
	}
	if idx := strings.LastIndex(name, ":"); idx >= 0 && !strings.Contains(name[idx:], "/") {
		name = name[:idx]
	}
	return name
}

// Digest returns the digest part of the image reference (e.g. "sha256:...").
func (i Image) Digest() string {
	name := string(i)
	if idx := strings.Index(name, "@"); idx >= 0 {
		return name[idx+1:]
	}
	return ""
}

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
