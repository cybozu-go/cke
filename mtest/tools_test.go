//go:build tools
// +build tools

package mtest

// this is to avoid removal from go.mod. gofail and ginkgo are used in mtest/Makefile.
import (
	_ "github.com/onsi/ginkgo/v2/ginkgo"
	_ "go.etcd.io/gofail"
	_ "go.etcd.io/gofail/runtime"
)
