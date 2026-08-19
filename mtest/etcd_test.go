package mtest

import (
	"encoding/json"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/cybozu-go/cke"
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

func testEtcd() {
	var crt, key, ca string

	BeforeAll(func() {
		crt, key, ca = rootCertFiles()
	})

	It("should not run etcd with auto-compaction", func() {
		for _, node := range []string{node1, node2, node3} {
			stdout := execSafeAt(node, "docker", "inspect", "--format='{{json .Args}}'", "etcd")
			var args []string
			err := json.Unmarshal(stdout, &args)
			Expect(err).NotTo(HaveOccurred(), "stdout=%s", stdout)

			// Compaction is done only by kube-apiserver.  See docs/etcd.md.
			Expect(args).NotTo(ContainElement(HavePrefix("--auto-compaction")), "node=%s", node)

			// Disabling auto-compaction must not disable data corruption detection.
			Expect(args).To(ContainElement("--feature-gates=InitialCorruptCheck=true,CompactHashCheck=true"), "node=%s", node)
			Expect(args).To(ContainElement("--corrupt-check-time=3h"), "node=%s", node)
		}
	})

	It("should allow kube-apiserver to compact etcd", func() {
		// kube-apiserver needs the root role because it updates "compact_rev_key",
		// which is out of its own key prefix, and calls the Compact API.
		stdout, stderr, err := etcdctl(crt, key, ca, "user", "get", "kube-apiserver")
		Expect(err).NotTo(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)
		Expect(string(stdout)).To(MatchRegexp(`(?m)^Roles:.*\broot\b`), "stdout=%s", stdout)
	})

	It("should compact revisions by kube-apiserver", func() {
		// old revisions become unreadable once compaction has been done
		Eventually(func(g Gomega) {
			_, stderr, err := etcdctl(crt, key, ca, "get", "compact_rev_key", "--rev=1")
			g.Expect(err).To(HaveOccurred())
			g.Expect(string(stderr)).To(ContainSubstring("has been compacted"))
		}, 15*time.Minute, 10*time.Second).Should(Succeed())
	})
}
