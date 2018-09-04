// +build tools

package tools

import (
	_ "github.com/onsi/ginkgo/ginkgo"
	_ "github.com/cybozu-go/placemat/cmd/placemat"
	_ "github.com/cybozu-go/placemat/cmd/placemat-connect"
	_ "github.com/etcd-io/gofail"
	_ "github.com/etcd-io/gofail/runtime"
)
