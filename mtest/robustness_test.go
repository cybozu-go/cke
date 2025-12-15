package mtest

import (
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	"k8s.io/utils/ptr"
)

// TODO
func findNodeCondition(node *corev1.Node, condType corev1.NodeConditionType) corev1.ConditionStatus {
	for _, cond := range node.Status.Conditions {
		if cond.Type == condType {
			return cond.Status
		}
	}
	return ""
}

func testRobustness() {
	It("should stop control plane nodes", func() {
		// stop CKE temporarily to avoid hang-up in SSH session due to node2 shutdown
		stopCKE()

		execAt(node2, "sudo", "systemd-run", "halt", "-f", "-f")
		Eventually(func(g Gomega) {
			_, err := execAtLocal("ping", "-c", "1", "-W", "1", node2)
			g.Expect(err).To(HaveOccurred())
		}).Should(Succeed())

		// TODO: Should we test ssh not connected case?
		// execAt(node3, "sudo", "systemctl", "stop", "sshd.socket")

		runCKE(ckeImageURL)

		waitServerStatusCompletion()
	})

	It("should not update control plane node labels", func() {
		for _, n := range []string{node1, node2, node3} {
			out, _, err := kubectl("get", "-o=json", "node", n)
			Expect(err).ShouldNot(HaveOccurred())
			var node corev1.Node
			err = json.Unmarshal(out, &node)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(node.Labels).Should(HaveKeyWithValue("cke.cybozu.com/master", "true"))
		}

		for _, n := range []string{node4, node5} {
			out, _, err := kubectl("get", "-o=json", "node", n)
			Expect(err).ShouldNot(HaveOccurred())
			var node corev1.Node
			err = json.Unmarshal(out, &node)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(node.Labels).ShouldNot(HaveKey("cke.cybozu.com/master"))
		}
	})

	It("should update endpoints", func() {
		By("Testing default/kubernetes EndpointSlices")
		out, _, err := kubectl("get", "-o=json", "endpointslices/kubernetes")
		Expect(err).NotTo(HaveOccurred())
		var eps discoveryv1.EndpointSlice
		err = json.Unmarshal(out, &eps)
		Expect(err).NotTo(HaveOccurred())
		Expect(eps.Endpoints).To(ConsistOf(
			discoveryv1.Endpoint{
				Addresses:  []string{node1},
				Conditions: discoveryv1.EndpointConditions{Ready: ptr.To(true)},
			},
			discoveryv1.Endpoint{
				Addresses:  []string{node2},
				Conditions: discoveryv1.EndpointConditions{Ready: ptr.To(false)},
			},
			discoveryv1.Endpoint{
				Addresses:  []string{node3},
				Conditions: discoveryv1.EndpointConditions{Ready: ptr.To(true)},
			},
		))
	})

	It("should not update k8s components", func() {
		// When a control plane node is down, CKE should not maintain k8s components.

		By("stopping kubelet")
		execAt(node5, "docker", "stop", "kubelet")
		Eventually(func(g Gomega) {
			out, _, err := kubectl("get", "-o=json", "node", node5)
			g.Expect(err).NotTo(HaveOccurred())
			var node corev1.Node
			err = json.Unmarshal(out, &node)
			g.Expect(err).NotTo(HaveOccurred())
			st := findNodeCondition(&node, corev1.NodeReady)
			Expect(st).NotTo(Equal(corev1.ConditionTrue))
		}).Should(Succeed())

		By("wating for CKE processing")
		waitServerStatusCompletion()

		// kubelet should not be restarted.
		Eventually(func(g Gomega) {
			out, _, err := kubectl("get", "-o=json", "node", node5)
			g.Expect(err).NotTo(HaveOccurred())
			var node corev1.Node
			err = json.Unmarshal(out, &node)
			g.Expect(err).NotTo(HaveOccurred())
			st := findNodeCondition(&node, corev1.NodeReady)
			Expect(st).NotTo(Equal(corev1.ConditionTrue))
		}).Should(Succeed())
	})

	It("TODO", func() {
		By("Removing a control plane node from the cluster")
		stopCKE()
		ckecliSafe("constraints", "set", "control-plane-count", "2")
		cluster := getCluster(0, 1, 2)
		cluster.Nodes = append(cluster.Nodes[:1], cluster.Nodes[2:]...)
		err := ckecliClusterSet(cluster)
		Expect(err).NotTo(HaveOccurred())
		runCKE(ckeImageURL)
		waitServerStatusCompletion()
	})

	By("Testing default/kubernetes EndpointSlices")
	out, _, err := kubectl("get", "-o=json", "endpointslices/kubernetes")
	Expect(err).NotTo(HaveOccurred())
	eps := discoveryv1.EndpointSlice{}
	err = json.Unmarshal(out, &eps)
	Expect(err).NotTo(HaveOccurred())
	Expect(eps.Endpoints).To(ConsistOf(
		discoveryv1.Endpoint{
			Addresses:  []string{node1},
			Conditions: discoveryv1.EndpointConditions{Ready: ptr.To(true)},
		},
		discoveryv1.Endpoint{
			Addresses:  []string{node3},
			Conditions: discoveryv1.EndpointConditions{Ready: ptr.To(true)},
		},
	))

	for _, n := range []string{node1, node3} {
		out, _, err := kubectl("get", "-o=json", "node", n)
		Expect(err).ShouldNot(HaveOccurred())
		var node corev1.Node
		err = json.Unmarshal(out, &node)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(node.Labels).Should(HaveKeyWithValue("cke.cybozu.com/master", "true"))
		st := findNodeCondition(&node, corev1.NodeReady)
		Expect(st).To(Equal(corev1.ConditionTrue))
	}

	for _, n := range []string{node4, node5} {
		out, _, err := kubectl("get", "-o=json", "node", n)
		Expect(err).ShouldNot(HaveOccurred())
		var node corev1.Node
		err = json.Unmarshal(out, &node)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(node.Labels).ShouldNot(HaveKey("cke.cybozu.com/master"))
		st := findNodeCondition(&node, corev1.NodeReady)
		Expect(st).To(Equal(corev1.ConditionTrue))
	}
}
