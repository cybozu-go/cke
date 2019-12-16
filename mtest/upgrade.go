package mtest

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

// TestUpgrade tests CKE upgrade operators
func TestUpgrade() {
	It("tests Kubernetes before reboot", func() {
		Eventually(func() error {
			_, _, err := kubectl("get", "sa/default")
			return err
		}).Should(Succeed())
		Eventually(func() error {
			for resource, name := range map[string]string{
				"serviceaccounts":     "cke-cluster-dns",
				"clusterroles":        "system:cluster-dns",
				"clusterrolebindings": "system:cluster-dns",
				"configmaps":          "cluster-dns",
				"deployments":         "cluster-dns",
				"services":            "cluster-dns",
			} {
				_, stderr, err := kubectl("-n", "kube-system", "get", resource+"/"+name)
				if err != nil {
					return fmt.Errorf("stderr: %s, err: %s", stderr, err)
				}
			}
			return nil
		}).Should(Succeed())
	})

	It("reboots all nodes", func() {
		stopCKE()

		nodes := []string{node1, node2, node3, node4, node5, node6}
		for _, n := range nodes {
			execAt(n, "sudo", "systemd-run", "reboot", "-f", "-f")
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
		runCKE(ckeImageURL)
		cluster := getCluster()
		for i := 0; i < 3; i++ {
			cluster.Nodes[i].ControlPlane = true
		}
		looseCheck = false
		Eventually(func() error {
			return checkCluster(cluster)
		}).Should(Succeed())
	})
}
