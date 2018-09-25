package mtest

import (
	"encoding/json"

	"github.com/cybozu-go/cke/cli"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("ckecli", func() {
	AfterEach(initializeControlPlane)

	It("should create users with limited access rights", func() {
		By("creating user and role for etcd")
		userName := "mtest"
		stdout := ckecli("etcd", "user-add", userName, "/mtest")
		Expect(string(stdout)).Should(ContainSubstring(userName + " created"))

		By("issuing certificate")
		stdout = ckecli("etcd", "issue", userName)
		var res cli.IssueResponse
		err := json.Unmarshal(stdout, &res)
		Expect(err).NotTo(HaveOccurred())

		By("executing etcdctl")
		c := localTempFile(res.Crt)
		k := localTempFile(res.Key)
		ca := localTempFile(res.CACrt)
		err = etcdctl(c.Name(), k.Name(), ca.Name(), "put", "/mtest/a", "test")
		Expect(err).NotTo(HaveOccurred())
		err = etcdctl(c.Name(), k.Name(), ca.Name(), "put", "/a", "test")
		Expect(err).To(HaveOccurred())
	})

	It("should connect to the CKE managed etcd", func() {
		By("issuing root certificate")
		stdout := ckecli("etcd", "root-issue")
		var res cli.IssueResponse
		err := json.Unmarshal(stdout, &res)
		Expect(err).NotTo(HaveOccurred())

		By("executing etcdctl")
		c := localTempFile(res.Crt)
		k := localTempFile(res.Key)
		ca := localTempFile(res.CACrt)
		err = etcdctl(c.Name(), k.Name(), ca.Name(), "endpoint", "health")
		Expect(err).NotTo(HaveOccurred())
	})
})
