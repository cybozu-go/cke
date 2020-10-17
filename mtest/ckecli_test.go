package mtest

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/cybozu-go/cke"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func testCKECLI() {
	It("should be able to re-run vault init", func() {
		execSafeAt(host1, "env", "VAULT_TOKEN=cybozu", "VAULT_ADDR=http://10.0.0.11:8200",
			"/opt/bin/ckecli", "vault", "init")
	})

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

	It("should be able to take backups locally", func() {
		dir, err := ioutil.TempDir("", "")
		Expect(err).NotTo(HaveOccurred())
		defer func() {
			os.RemoveAll(dir)
		}()

		ckecliSafe("etcd", "local-backup", "--dir="+dir)

		out := execSafeAt(host1, "ls", dir)
		names := strings.Fields(string(out))
		Expect(names).To(HaveLen(1))
		oldest := names[0]
		Expect(oldest).To(HavePrefix("etcd-"))

		time.Sleep(2 * time.Second)
		ckecliSafe("etcd", "local-backup", "--dir="+dir, "--max-backups=2")
		time.Sleep(2 * time.Second)
		ckecliSafe("etcd", "local-backup", "--dir="+dir, "--max-backups=2")
		out = execSafeAt(host1, "ls", dir)
		names = strings.Fields(string(out))
		Expect(names).To(HaveLen(2))
		Expect(names).NotTo(ContainElement(oldest))
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

	It("should invoke sabakan subcommand successfully", func() {
		ckecliSafe("sabakan", "set-url", "http://localhost:10080")
		ckecliSafe("sabakan", "disable")
		ckecliSafe("sabakan", "enable")
		ckecliSafe("sabakan", "get-url")
	})
}
