package mtest

import (
	"encoding/json"
	"os"
	"os/exec"

	"github.com/cybozu-go/cke"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("ckecli", func() {
	AfterEach(initializeControlPlane)

	It("should create etcd users with limited access rights", func() {
		By("creating user and role for etcd")
		userName := "mtest"
		ckecli("etcd", "user-add", userName, "/mtest/")

		By("issuing certificate")
		stdout := ckecli("etcd", "issue", userName)
		var res cke.IssueResponse
		err := json.Unmarshal(stdout, &res)
		Expect(err).NotTo(HaveOccurred())

		By("executing etcdctl")
		c := localTempFile(res.Cert)
		k := localTempFile(res.Key)
		ca := localTempFile(res.CACert)
		err = etcdctl(c.Name(), k.Name(), ca.Name(), "put", "/mtest/a", "test")
		Expect(err).ShouldNot(HaveOccurred())
		err = etcdctl(c.Name(), k.Name(), ca.Name(), "put", "/a", "test")
		Expect(err).Should(HaveOccurred())
	})

	It("should connect to the CKE managed etcd", func() {
		By("issuing root certificate")
		stdout := ckecli("etcd", "root-issue")
		var res cke.IssueResponse
		err := json.Unmarshal(stdout, &res)
		Expect(err).ShouldNot(HaveOccurred())

		By("executing etcdctl")
		c := localTempFile(res.Cert)
		k := localTempFile(res.Key)
		ca := localTempFile(res.CACert)
		err = etcdctl(c.Name(), k.Name(), ca.Name(), "endpoint", "health")
		Expect(err).ShouldNot(HaveOccurred())
	})

	It("should connect to the kube-apiserver", func() {
		By("using admin certificate")
		stdout := ckecli("kubernetes", "issue")
		kubeconfig := localTempFile(string(stdout))
		cmd := exec.Command(kubectlPath, "--kubeconfig", kubeconfig.Name(), "get", "nodes")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		Expect(cmd.Run()).ShouldNot(HaveOccurred())
	})
})
