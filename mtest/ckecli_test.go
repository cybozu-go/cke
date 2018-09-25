package mtest

import (
	"encoding/json"

	"github.com/cybozu-go/cke/cli"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("ckecli", func() {
	AfterEach(initializeControlPlane)

	It("should issue client certificate for etcd and connect to the CKE managed etcd", func() {
		By("execute ckecli etcd user-add")
		stdout := ckecli("etcd", "user-add", "mtest", "/")
		var res cli.IssueResponse
		err := json.Unmarshal(stdout, &res)
		Expect(err).NotTo(HaveOccurred())

		By("execute etcdctl")
		c := localTempFile(res.Crt)
		k := localTempFile(res.Key)
		ca := localTempFile(res.CACrt)
		err = etcdctl(c.Name(), k.Name(), ca.Name(), "endpoint", "health")
		Expect(err).NotTo(HaveOccurred())
	})
})
