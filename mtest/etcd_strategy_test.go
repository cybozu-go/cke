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
			return checkEtcdClusterStatus(status, controlPlanes, workers)
		}).Should(BeTrue())

		By("Checking that CKE did not remove non-cluster node's data")
		status, err := getClusterStatus()
		Expect(err).NotTo(HaveOccurred())
		Expect(status.NodeStatuses[node1].Etcd.HasData).To(BeTrue())
	})

	It("should remove unhealthy-and-non-control-plane node2 from etcd cluster, and destroy it's etcd", func() {
		By("stopping etcd in node2")
		execSafeAt(node2, "docker", "stop", "etcd")
		execSafeAt(node2, "docker", "rm", "etcd")

		By("Changing definition of node2")
		ckecli("constraints", "set", "control-plane-count", "2")
		cluster := getCluster()
		cluster.Nodes[0].ControlPlane = true
		cluster.Nodes[2].ControlPlane = true
		ckecliClusterSet(cluster)

		By("Checking cluster status")
		Eventually(func() bool {
			controlPlanes := []string{node1, node3}
			workers := []string{node2, node4, node5, node6}
			status, err := getClusterStatus()
			if err != nil {
				return false
			}
			return checkEtcdClusterStatus(status, controlPlanes, workers)
		}).Should(BeTrue())

		By("Checking that CKE removed worker node's data")
		status, err := getClusterStatus()
		Expect(err).NotTo(HaveOccurred())
		Expect(status.NodeStatuses[node2].Etcd.HasData).To(BeFalse())
	})

	// unit test of etcd strategy contains a case of "start unstarted member",
	// but that case is not here, because it is difficult to make "unstarted member"

	It("should remove non-control-plane node2 from etcd cluster, and destroy it's etcd", func() {
		By("Changing definition of node2")
		ckecli("constraints", "set", "control-plane-count", "2")
		cluster := getCluster()
		cluster.Nodes[0].ControlPlane = true
		cluster.Nodes[2].ControlPlane = true
		ckecliClusterSet(cluster)

		By("Checking cluster status")
		Eventually(func() bool {
			controlPlanes := []string{node1, node3}
			workers := []string{node2, node4, node5, node6}
			status, err := getClusterStatus()
			if err != nil {
				return false
			}
			return checkEtcdClusterStatus(status, controlPlanes, workers)
		}).Should(BeTrue())

		By("Checking that CKE removed worker node's data")
		status, err := getClusterStatus()
		Expect(err).NotTo(HaveOccurred())
		Expect(status.NodeStatuses[node2].Etcd.HasData).To(BeFalse())
	})

	It("should remove unhealthy node1 from etcd cluster and add node4 in appropriate order", func() {
		By("Stopping etcd in node1 and changing definition of node1/node4 at once")
		stopCKE()
		execSafeAt(node1, "docker", "stop", "etcd")
		execSafeAt(node1, "docker", "rm", "etcd")
		cluster := getCluster()
		cluster.Nodes[0].ControlPlane = false
		cluster.Nodes[1].ControlPlane = true
		cluster.Nodes[2].ControlPlane = true
		cluster.Nodes[3].ControlPlane = true
		ckecliClusterSet(cluster)
		runCKE()

		By("Checking cluster status")
		Eventually(func() bool {
			controlPlanes := []string{node2, node3, node4}
			workers := []string{node1, node5, node6}
			status, err := getClusterStatus()
			if err != nil {
				return false
			}
			return checkEtcdClusterStatus(status, controlPlanes, workers)
		}).Should(BeTrue())
	})
})
