package mtest

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"

	"github.com/cybozu-go/cke"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

// TestCKECLI tests ckecli command
func TestCKECLI() {
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

	It("should ssh to all nodes", func() {
		for _, node := range []string{node1, node2, node3, node4, node5, node6} {
			Eventually(func() error {
				_, err := ckecliUnsafe("", "ssh", "cybozu@"+node, "/bin/true")
				if err != nil {
					return err
				}
				return nil
			}).Should(Succeed())
		}
	})

	It("should scp to all nodes", func() {
		scpData := localTempFile("scpData")
		for _, node := range []string{node1, node2, node3, node4, node5, node6} {
			destName := scpData.Name() + node

			Eventually(func() error {
				stdout, err := ckecliUnsafe("", "scp", scpData.Name(), "cybozu@"+node+":"+destName)
				if err != nil {
					return fmt.Errorf("%v: stdout=%s", err, stdout)
				}
				stdout, err = ckecliUnsafe("", "scp", "cybozu@"+node+":"+destName, "/tmp/")
				if err != nil {
					return fmt.Errorf("%v: stdout=%s", err, stdout)
				}
				_, err = os.Stat(destName)
				if err != nil {
					return err
				}
				return nil
			}).Should(Succeed())

			os.Remove(destName)
		}
	})
}
