package cke

import (
	"slices"
	"strings"
)

//go:generate go run ./hack/update-images

// Image is a container image reference pinned to a digest.
type Image struct {
	repository string
	tag        string
	digest     string
}

func newImage(repository, tag, digest string) Image {
	return Image{repository: repository, tag: tag, digest: digest}
}

// FullRef returns "repository:tag@digest".
func (i Image) FullRef() string {
	return i.repository + ":" + i.tag + "@" + i.digest
}

// TagRef returns "repository:tag".
func (i Image) TagRef() string {
	return i.repository + ":" + i.tag
}

// DigestRef returns "repository@digest".
func (i Image) DigestRef() string {
	return i.repository + "@" + i.digest
}

// AllImages returns the list of all container images used by CKE.
func AllImages() []Image {
	return slices.Clone(allImages)
}

// MatchesRunning reports whether running, the image recorded for a running
// container, is this image. Containers started by older CKE record
// "repository:tag" only and match by tag.
func (i Image) MatchesRunning(running string) bool {
	if strings.Contains(running, "@") {
		return running == i.FullRef()
	}
	return running == i.TagRef()
}
