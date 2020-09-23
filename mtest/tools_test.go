// +build tools

package mtest

// this is to avoid removal from go.mod. gofail and ginkgo are used in mtest/Makefile.
import (
	_ "github.com/etcd-io/gofail"
	_ "github.com/etcd-io/gofail/runtime"
	_ "github.com/onsi/ginkgo"
	_ "github.com/onsi/ginkgo/ginkgo"
)
