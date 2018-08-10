package cke

// Container image definitions
const (
	EtcdImage  = "quay.io/cybozu/etcd:3.3.5-1"
	CkeTools   = "quay.io/cybozu/cke-tools:0"
	ToolsImage = "quay.io/cybozu/ubuntu:18.04"
)

// Image returns the image name for a given container.
func Image(name string) string {
	switch name {
	case "etcd":
		return EtcdImage
	case "rivers":
		return CkeTools
	case "tools":
		return ToolsImage
	}

	panic("no such image: " + name)
}
