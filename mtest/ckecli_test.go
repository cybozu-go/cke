package mtest

import (
	"encoding/json"
	"os"
	"os/exec"

	"github.com/cybozu-go/cke/cli"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("ckecli", func() {
	AfterEach(initializeControlPlane)

	It("should create etcd users with limited access rights", func() {
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
		Expect(err).ShouldNot(HaveOccurred())
		err = etcdctl(c.Name(), k.Name(), ca.Name(), "put", "/a", "test")
		Expect(err).Should(HaveOccurred())
	})

	It("should connect to the CKE managed etcd", func() {
		By("issuing root certificate")
		stdout := ckecli("etcd", "root-issue")
		var res cli.IssueResponse
		err := json.Unmarshal(stdout, &res)
		Expect(err).ShouldNot(HaveOccurred())

		By("executing etcdctl")
		c := localTempFile(res.Crt)
		k := localTempFile(res.Key)
		ca := localTempFile(res.CACrt)
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
