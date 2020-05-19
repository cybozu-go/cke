package mtest

import (
	"os"
	"testing"

	"github.com/cybozu-go/cke/mtest"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestMtest(t *testing.T) {
	if os.Getenv("SSH_PRIVKEY") == "" {
		t.Skip("no SSH_PRIVKEY envvar")
	}
	RegisterFailHandler(Fail)
	RunSpecs(t, "Multi-host test for cke")
}

var _ = BeforeSuite(func() {
	mtest.RunBeforeSuite("")
})

// This must be the only top-level test container.
// Other tests and test containers must be listed in this.
var _ = Describe("Test CKE functions", func() {
	mtest.FunctionsSuite()
})
