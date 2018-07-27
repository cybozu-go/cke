package mtest

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("etcd strategy", func() {
	AfterEach(initializeControlPlane)

	It("should remove unhealthy-and-not-in-cluster node1 from etcd cluster", func() {
		By("Stopping etcd in node1")
		execSafeAt(node1, "docker", "stop", "etcd")
		execSafeAt(node1, "docker", "rm", "etcd")

		By("Removing definition of node1")
		ckecli("constraints", "set", "control-plane-count", "2")
		cluster := getCluster()
		for i := 0; i < 3; i++ {
			cluster.Nodes[i].ControlPlane = true
		}
		cluster.Nodes = cluster.Nodes[1:]
		ckecliClusterSet(cluster)

		By("Checking cluster status")
		Eventually(func() bool {
			controlPlanes := []string{node2, node3}
			workers := []string{node4, node5, node6}
			status, err := getClusterStatus()
			if err != nil {
				return false
			}
			defer status.Destroy()
			return checkEtcdClusterStatus(status, controlPlanes, workers)
		}).Should(BeTrue())

		By("Checking that CKE did not remove non-cluster node's data")
		status, err := getClusterStatus()
		Expect(err).NotTo(HaveOccurred())
		defer status.Destroy()
		Expect(status.NodeStatuses[node1].Etcd.HasData).To(BeTrue())
	})

	It("should update node4 as control plane", func() {
		By("Changing definition of node4")
		ckecli("constraints", "set", "control-plane-count", "4")
		cluster := getCluster()
		for i := 0; i < 4; i++ {
			cluster.Nodes[i].ControlPlane = true
		}
		ckecliClusterSet(cluster)

		By("Checking cluster status")
		Eventually(func() bool {
			controlPlanes := []string{node1, node2, node3, node4}
			workers := []string{node5, node6}
			status, err := getClusterStatus()
			if err != nil {
				return false
			}
			defer status.Destroy()
			return checkEtcdClusterStatus(status, controlPlanes, workers)
		}).Should(BeTrue())
	})
})
