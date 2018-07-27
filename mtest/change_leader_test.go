package mtest

import (
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("etcd strategy when the leader is changed", func() {
	AfterEach(initializeControlPlane)

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

		By("Checking new leader")
		newLeader := strings.TrimSpace(string(ckecli("leader")))
		Expect(newLeader).To(Or(Equal("host1"), Equal("host2")))
		Expect(newLeader).NotTo(Equal(firstLeader))
	})
})
