package mtest

import (
	"encoding/json"
	"fmt"

	"github.com/cybozu-go/cke"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

// TestCKECLI tests ckecli command
func TestCKECLI() {
	It("should create etcd users with limited access rights", func() {
		By("creating user and role for etcd")
		userName := "mtest"
		ckecliSafe("etcd", "user-add", userName, "/mtest/")

		By("issuing certificate")
		stdout := ckecliSafe("etcd", "issue", userName)
		var res cke.IssueResponse
		err := json.Unmarshal(stdout, &res)
		Expect(err).NotTo(HaveOccurred())

		By("copying certificate")
		rc := remoteTempFile(res.Cert)
		rk := remoteTempFile(res.Key)
		rca := remoteTempFile(res.CACert)

		By("executing etcdctl")
		stdout, stderr, err := etcdctl(rc, rk, rca, "put", "/mtest/a", "test")
		Expect(err).NotTo(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)
		stdout, stderr, err = etcdctl(rc, rk, rca, "put", "/a", "test")
		Expect(err).To(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)
	})

	It("should connect to the CKE managed etcd", func() {
		By("issuing root certificate")
		stdout := ckecliSafe("etcd", "root-issue")
		var res cke.IssueResponse
		err := json.Unmarshal(stdout, &res)
		Expect(err).ShouldNot(HaveOccurred())

		By("copying certificate")
		rc := remoteTempFile(res.Cert)
		rk := remoteTempFile(res.Key)
		rca := remoteTempFile(res.CACert)

		By("executing etcdctl")
		stdout, stderr, err := etcdctl(rc, rk, rca, "endpoint", "health")
		Expect(err).NotTo(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)
	})

	It("should connect to the kube-apiserver", func() {
		By("using admin certificate")
		ckecliSafe("kubernetes", "issue", ">", "/tmp/mtest-kube-config")

		stdout, stderr, err := kubectl("--kubeconfig", "/tmp/mtest-kube-config", "get", "nodes")
		Expect(err).NotTo(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)

		execSafeAt(host1, "rm", "-f", "/tmp/mtest-kube-config")
	})

	It("should ssh to all nodes", func() {
		for _, node := range []string{node1, node2, node3, node4, node5, node6} {
			Eventually(func() error {
				_, _, err := ckecli("ssh", "cybozu@"+node, "/bin/true")
				if err != nil {
					return err
				}
				return nil
			}).Should(Succeed())
		}
	})

	It("should scp to all nodes", func() {
		for _, node := range []string{node1, node2, node3, node4, node5, node6} {
			srcFile := "/tmp/scpData-" + node
			dstFile := srcFile + "-dest"
			execSafeAt(host1, "touch", srcFile)

			Eventually(func() error {
				stdout, _, err := ckecli("scp", srcFile, node+":"+dstFile)
				if err != nil {
					return fmt.Errorf("%v: stdout=%s", err, stdout)
				}
				stdout, _, err = ckecli("scp", node+":"+dstFile, "/tmp/")
				if err != nil {
					return fmt.Errorf("%v: stdout=%s", err, stdout)
				}
				_, _, err = execAt(host1, "test", "-f", dstFile)
				if err != nil {
					return err
				}
				return nil
			}).Should(Succeed())

			execSafeAt(host1, "rm", "-f", srcFile, dstFile)
		}
	})
}
