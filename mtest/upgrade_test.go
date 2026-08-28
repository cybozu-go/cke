package mtest

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func testUpgrade() {
	It("tests Kubernetes before reboot", func() {
		Eventually(func() error {
			_, _, err := kubectl("get", "sa/default")
			return err
		}).Should(Succeed())
	})

	It("reboots all nodes", func() {
		Expect(stopCKE()).To(Succeed())

		nodes := []string{node1, node2, node3, node4, node5, node6}
		for _, n := range nodes {
			_, _, err := execAt(n, "sudo", "systemd-run", "reboot", "-f", "-f")
			Expect(err).NotTo(HaveOccurred())
		}
		time.Sleep(10 * time.Second)
		Eventually(func() error {
			for _, n := range nodes {
				_, err := execAtLocal("ping", "-c", "1", "-W", "1", n)
				if err != nil {
					return err
				}
			}
			return nil
		}).Should(Succeed())

		Expect(prepareSSHClients(nodes...)).Should(Succeed())
	})

	It("runs new CKE", func() {
		Expect(runCKE(ckeImageURL)).To(Succeed())
		waitServerStatusCompletion()
	})

	It("removes kubectl cache", func() {
		execSafeAt(host1, "rm", "-rf", "~/.kube/cache")
	})
}
