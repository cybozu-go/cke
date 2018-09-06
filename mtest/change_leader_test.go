package mtest

import (
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("etcd strategy when the leader is changed", func() {
	AfterEach(func() {
		setupCKE()
		initializeControlPlane()
	})

	It("should update node4 as control plane", func() {
		By("Checking current leader")
		firstLeader := strings.TrimSpace(string(ckecli("leader")))
		Expect(firstLeader).To(Or(Equal("host1"), Equal("host2")))

		By("Crashing CKE after adding etcd member")
		injectFailure("etcdAfterMemberAdd")

		By("Changing definition of node4")
		ckecli("constraints", "set", "control-plane-count", "4")
		cluster := getCluster()
		for i := 0; i < 4; i++ {
			cluster.Nodes[i].ControlPlane = true
		}
		ckecliClusterSet(cluster)

		By("Checking cluster status")
		Eventually(func() error {
			controlPlanes := []string{node1, node2, node3, node4}
			workers := []string{node5, node6}
			status, err := getClusterStatus()
			if err != nil {
				return err
			}
			return checkEtcdClusterStatus(status, controlPlanes, workers)
		}).Should(Succeed())

		By("Checking new leader")
		newLeader := strings.TrimSpace(string(ckecli("leader")))
		Expect(newLeader).To(Or(Equal("host1"), Equal("host2")))
		Expect(newLeader).NotTo(Equal(firstLeader))
	})

	It("should remove node3 from control plane", func() {
		By("Checking current leader")
		firstLeader := strings.TrimSpace(string(ckecli("leader")))
		Expect(firstLeader).To(Or(Equal("host1"), Equal("host2")))

		By("Crashing CKE after removing etcd member")
		injectFailure("etcdAfterMemberRemove")

		By("Changing definition of node3")
		ckecli("constraints", "set", "control-plane-count", "2")
		cluster := getCluster()
		for i := 0; i < 2; i++ {
			cluster.Nodes[i].ControlPlane = true
		}
		ckecliClusterSet(cluster)

		By("Checking cluster status")
		Eventually(func() error {
			controlPlanes := []string{node1, node2}
			workers := []string{node3, node4, node5, node6}
			status, err := getClusterStatus()
			if err != nil {
				return err
			}
			return checkEtcdClusterStatus(status, controlPlanes, workers)
		}).Should(Succeed())

		By("Checking new leader")
		newLeader := strings.TrimSpace(string(ckecli("leader")))
		Expect(newLeader).To(Or(Equal("host1"), Equal("host2")))
		Expect(newLeader).NotTo(Equal(firstLeader))
	})

	It("should remove and recover node2", func() {
		By("Checking current leader")
		firstLeader := strings.TrimSpace(string(ckecli("leader")))
		Expect(firstLeader).To(Or(Equal("host1"), Equal("host2")))

		By("Crashing CKE after stopping docker container")
		injectFailure("dockerAfterContainerStop")

		By("Remove node2 from control plane")
		ckecli("constraints", "set", "control-plane-count", "2")
		cluster := getCluster()
		cluster.Nodes[0].ControlPlane = true
		cluster.Nodes[1].ControlPlane = false
		cluster.Nodes[2].ControlPlane = true
		ckecliClusterSet(cluster)

		By("Checking cluster status")
		Eventually(func() error {
			controlPlanes := []string{node1, node3}
			workers := []string{node2, node4, node5, node6}
			status, err := getClusterStatus()
			if err != nil {
				return err
			}
			return checkEtcdClusterStatus(status, controlPlanes, workers)
		}).Should(Succeed())

		By("Add node2 as control plane")
		ckecli("constraints", "set", "control-plane-count", "3")
		cluster.Nodes[1].ControlPlane = true
		ckecliClusterSet(cluster)

		By("Checking cluster status")
		Eventually(func() error {
			controlPlanes := []string{node1, node2, node3}
			workers := []string{node4, node5, node6}
			status, err := getClusterStatus()
			if err != nil {
				return err
			}
			return checkEtcdClusterStatus(status, controlPlanes, workers)
		}).Should(Succeed())

		By("Checking new leader")
		newLeader := strings.TrimSpace(string(ckecli("leader")))
		Expect(newLeader).To(Or(Equal("host1"), Equal("host2")))
		Expect(newLeader).NotTo(Equal(firstLeader))
	})
})
