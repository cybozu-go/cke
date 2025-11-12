package mtest

import (
	"encoding/json"
	"slices"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	"k8s.io/utils/ptr"
)

func testRobustness() {
	It("should stop control plane nodes", func() {
		// stop CKE temporarily to avoid hang-up in SSH session due to node2 shutdown
		stopCKE()

		execAt(node2, "sudo", "systemd-run", "halt", "-f", "-f")
		Eventually(func(g Gomega) {
			_, err := execAtLocal("ping", "-c", "1", "-W", "1", node2)
			g.Expect(err).To(HaveOccurred())
		}).Should(Succeed())

		runCKE(ckeImageURL)

		waitServerStatusCompletion()
	})

	It("should update endpoints", func() {
		stdout, stderr, err := kubectl("get", "-o=json", "endpointslices/kubernetes")
		Expect(err).NotTo(HaveOccurred(), "stderr: %s", stderr)

		var eps discoveryv1.EndpointSlice
		err = json.Unmarshal(stdout, &eps)
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

	It("should not update control plane node labels", func() {
		stdout, stderr, err := kubectl("get", "-o=json", "node")
		Expect(err).NotTo(HaveOccurred(), "stderr: %s", stderr)

		var nodeList corev1.NodeList
		err = json.Unmarshal(stdout, &nodeList)
		Expect(err).NotTo(HaveOccurred())

		Expect(nodeList.Items).To(HaveLen(5))
		for _, node := range nodeList.Items {
			switch node.Name {
			case node1, node2, node3:
				Expect(node.Labels).To(HaveKeyWithValue("cke.cybozu.com/master", "true"), "node: %s", node.Name)
			case node4, node5:
				Expect(node.Labels).NotTo(HaveKey("cke.cybozu.com/master"), "node: %s", node.Name)
			}
		}
	})

	It("should not update k8s components", func() {
		// When a control plane node is down, CKE should not maintain k8s components.

		By("stopping kubelet")
		execAt(node5, "docker", "stop", "kubelet")
		Eventually(func(g Gomega) {
			stdout, stderr, err := kubectl("get", "-o=json", "node", node5)
			g.Expect(err).NotTo(HaveOccurred(), "stderr: %s", stderr)

			var node corev1.Node
			err = json.Unmarshal(stdout, &node)
			g.Expect(err).NotTo(HaveOccurred())

			st := nodeConditionStatus(&node, corev1.NodeReady)
			g.Expect(st).NotTo(Equal(corev1.ConditionTrue))
		}).WithTimeout(3 * time.Minute).Should(Succeed()) // It should take some time to detect the kubelet down.

		By("wating for CKE processing")
		waitServerStatusCompletion()

		By("checking that kubelet is not started")
		Consistently(func(g Gomega) {
			stdout, stderr, err := kubectl("get", "-o=json", "node", node5)
			g.Expect(err).NotTo(HaveOccurred(), "stderr: %s", stderr)

			var node corev1.Node
			err = json.Unmarshal(stdout, &node)
			g.Expect(err).NotTo(HaveOccurred())

			st := nodeConditionStatus(&node, corev1.NodeReady)
			g.Expect(st).NotTo(Equal(corev1.ConditionTrue))
		}).WithTimeout(1 * time.Minute).Should(Succeed())
	})

	It("can kick out the stopped control plane node", func() {
		stopCKE()
		ckecliSafe("constraints", "set", "control-plane-count", "2")
		cluster := getCluster(0, 1, 2)
		cluster.Nodes = slices.Delete(cluster.Nodes, 1, 2)
		err := ckecliClusterSet(cluster)
		Expect(err).NotTo(HaveOccurred())
		runCKE(ckeImageURL)
		waitServerStatusCompletion()
	})

	It("should update endpoints", func() {
		stdout, stderr, err := kubectl("get", "-o=json", "endpointslices/kubernetes")
		Expect(err).NotTo(HaveOccurred(), "stderr: %s", stderr)

		var eps discoveryv1.EndpointSlice
		err = json.Unmarshal(stdout, &eps)
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
	})

	It("should not update control plane node labels", func() {
		stdout, stderr, err := kubectl("get", "-o=json", "node")
		Expect(err).NotTo(HaveOccurred(), "stderr: %s", stderr)

		var nodeList corev1.NodeList
		err = json.Unmarshal(stdout, &nodeList)
		Expect(err).NotTo(HaveOccurred())

		Expect(nodeList.Items).To(HaveLen(4))
		for _, node := range nodeList.Items {
			switch node.Name {
			case node1, node3:
				Expect(node.Labels).To(HaveKeyWithValue("cke.cybozu.com/master", "true"), "node: %s", node.Name)
			case node4, node5:
				Expect(node.Labels).NotTo(HaveKey("cke.cybozu.com/master"), "node: %s", node.Name)
			}
		}
	})

	It("should update k8s components", func() {
		Eventually(func(g Gomega) {
			stdout, stderr, err := kubectl("get", "-o=json", "node", node5)
			g.Expect(err).NotTo(HaveOccurred(), "stderr: %s", stderr)

			var node corev1.Node
			err = json.Unmarshal(stdout, &node)
			g.Expect(err).NotTo(HaveOccurred())

			st := nodeConditionStatus(&node, corev1.NodeReady)
			g.Expect(st).To(Equal(corev1.ConditionTrue))
		}).WithTimeout(3 * time.Minute).Should(Succeed())
	})
}
