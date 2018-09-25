package mtest

import (
	"bytes"
	"encoding/json"
	"os/exec"

	"github.com/cybozu-go/cke/cli"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
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
		etcdctlLocal(c.Name(), k.Name(), ca.Name(), "endpoint", "health")
	})
})

func etcdctlLocal(crt, key, ca string, args ...string) []byte {
	args = append([]string{"--endpoints=https://" + node1 + ":2379,https://" + node2 + ":2379,https://" + node3 + ":2379",
		"--cert=" + crt, "--key=" + key, "--cacert=" + ca},
		args...)
	command := exec.Command("output/etcdctl", args...)
	command.Env = append(command.Env, "ETCDCTL_API=3")
	stdout := new(bytes.Buffer)
	session, err := gexec.Start(command, stdout, GinkgoWriter)
	Ω(err).ShouldNot(HaveOccurred())
	Eventually(session).Should(gexec.Exit(0))
	return stdout.Bytes()
}
