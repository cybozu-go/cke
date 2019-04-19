// +build tools

package mtest

import (
	_ "github.com/etcd-io/gofail"
	_ "github.com/etcd-io/gofail/runtime"
	_ "github.com/onsi/ginkgo"
	_ "github.com/onsi/ginkgo/ginkgo"
)
