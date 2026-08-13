package mtest

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/cybozu-go/cke"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// rootCertFiles issues a certificate for the etcd root user and copies it to host1.
// It returns the paths of the certificate, the private key, and the CA certificate.
func rootCertFiles() (crt, key, ca string) {
	stdout := ckecliSafe("etcd", "root-issue")
	var res cke.IssueResponse
	err := json.Unmarshal(stdout, &res)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())

	return remoteTempFile(res.Cert), remoteTempFile(res.Key), remoteTempFile(res.CACert)
}

// etcdLogHas reports whether the etcd journal on the node has the given message.
// grep is run on the node not to transfer the whole journal.
func etcdLogHas(node, message string) bool {
	_, _, err := execAt(node, "sudo", "journalctl", "CONTAINER_NAME=etcd", "-q", "--no-pager",
		"|", "grep", "-qF", "'"+message+"'")
	return err == nil
}

func testEtcd() {
	var crt, key, ca string

	BeforeAll(func() {
		crt, key, ca = rootCertFiles()
	})

	It("should not run etcd with auto-compaction", func() {
		// Compaction is done only by kube-apiserver.  See docs/etcd.md.
		for _, node := range []string{node1, node2, node3} {
			stdout := execSafeAt(node, "docker", "inspect", "--format='{{json .Args}}'", "etcd")
			var args []string
			err := json.Unmarshal(stdout, &args)
			Expect(err).NotTo(HaveOccurred(), "stdout=%s", stdout)
			Expect(args).NotTo(ContainElement(HavePrefix("--auto-compaction")), "node=%s", node)
		}
	})

	It("should allow kube-apiserver to compact etcd", func() {
		// kube-apiserver needs the root role because it updates "compact_rev_key",
		// which is out of its own key prefix, and calls the Compact API.
		stdout, stderr, err := etcdctl(crt, key, ca, "user", "get", "kube-apiserver")
		Expect(err).NotTo(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)
		Expect(string(stdout)).To(MatchRegexp(`(?m)^Roles:.*\broot\b`), "stdout=%s", stdout)
	})

	It("should enable data corruption detection", func() {
		for _, node := range []string{node1, node2, node3} {
			Expect(etcdLogHas(node, "initial corruption checking passed")).To(BeTrue(), "node=%s", node)
			Expect(etcdLogHas(node, "enabled corruption checking")).To(BeTrue(), "node=%s", node)
		}
	})

	It("should compact revisions and check compaction hashes", func() {
		Eventually(func() error {
			// old revisions become unreadable once compaction has been done
			stdout, stderr, err := etcdctl(crt, key, ca, "get", "compact_rev_key", "--rev=1")
			if err == nil {
				return fmt.Errorf("revision 1 is not compacted yet: stdout=%s", stdout)
			}
			if !strings.Contains(string(stderr), "has been compacted") {
				return fmt.Errorf("unexpected error: stdout=%s, stderr=%s: %w", stdout, stderr, err)
			}

			// Compaction by kube-apiserver stores KV hashes as etcd's auto-compaction
			// does.  Only the leader logs this, after comparing the hash with all the
			// followers.
			for _, node := range []string{node1, node2, node3} {
				if etcdLogHas(node, "successfully checked hash on whole cluster") {
					return nil
				}
			}
			return fmt.Errorf("no etcd member has checked compaction hashes")
		}, 15*time.Minute, 10*time.Second).Should(Succeed())
	})
}
