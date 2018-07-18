package cke

// Container image definitions
const (
	EtcdImage = "quay.io/cybozu/etcd:3.3.5-1"
)

// Image returns the image name for a given container.
func Image(name string) string {
	switch name {
	case "etcd":
		return EtcdImage
	}

	panic("no such image: " + name)
}
