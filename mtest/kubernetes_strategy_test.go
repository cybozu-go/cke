package mtest

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("kubernetes strategy", func() {
	AfterEach(initializeControlPlane)

	It("should deploy HA control plane", func() {
		By("Checking cluster status")
		Eventually(func() error {
			controlPlanes := []string{node1, node2, node3}
			workers := []string{node4, node5, node6}
			status, err := getClusterStatus()
			if err != nil {
				return err
			}
			return checkKubernetesClusterStatus(status, controlPlanes, workers)
		}).Should(Succeed())
	})
})
