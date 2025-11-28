package mtest

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/cybozu-go/cke"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	kubeletv1beta1 "k8s.io/kubelet/config/v1beta1"
)

func testOperators(isDegraded bool) {
	AfterEach(initializeControlPlane)

	It("run all operators / commanders", func() {
		// these operators ran already:
		// - RiversBootOp
		// - EtcdRiversBootOp
		// - EtcdBootOp
		// - APIServerBootOp
		// - ControllerManagerBootOp
		// - SchedulerBootOp
		// - KubeletBootOp
		// - KubeProxyBootOp
		// - KubeWaitOp
		// - KubeRBACRoleInstallOp
		// - KubeClusterDNSCreateOp
		// - KubeEtcdServiceCreateOp
		// - KubeEndpointsCreateOp

		By("Testing control plane node labels")
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

		By("Testing default/kubernetes Endpoints")
		out, _, err := kubectl("get", "-o=json", "endpoints/kubernetes")
		Expect(err).ShouldNot(HaveOccurred())
		var ep corev1.Endpoints
		err = json.Unmarshal(out, &ep)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(ep.Subsets).Should(HaveLen(1))
		if isDegraded {
			Expect(ep.Subsets[0].Addresses).Should(ConsistOf(
				corev1.EndpointAddress{IP: node1},
			))
			Expect(ep.Subsets[0].NotReadyAddresses).Should(ConsistOf(
				corev1.EndpointAddress{IP: node2},
				corev1.EndpointAddress{IP: node3},
			))
		} else {
			Expect(ep.Subsets[0].Addresses).Should(ConsistOf(
				corev1.EndpointAddress{IP: node1},
				corev1.EndpointAddress{IP: node2},
				corev1.EndpointAddress{IP: node3},
			))
			Expect(ep.Subsets[0].NotReadyAddresses).Should(BeEmpty())
		}

		By("Stopping etcd servers")
		// this will run:
		// - EtcdStartOp
		// - EtcdWaitClusterOp
		if !isDegraded {
			stopCKE()
			execSafeAt(node2, "docker", "stop", "etcd")
			execSafeAt(node2, "docker", "rm", "etcd")
			execSafeAt(node3, "docker", "stop", "etcd")
			execSafeAt(node3, "docker", "rm", "etcd")
			runCKE(ckeImageURL)
			waitServerStatusCompletion()
		}

		By("Removing a control plane node from the cluster")
		// this will run:
		// - EtcdRemoveMemberOp
		// - KubeNodeRemoveOp
		// - RiversRestartOp
		// - EtcdRiversRestartOp
		// - APIServerRestartOp
		// - KubeEndpointsUpdateOp
		stopCKE()
		ckecliSafe("constraints", "set", "control-plane-count", "2")
		cluster := getCluster(0, 1, 2)
		cluster.Nodes = append(cluster.Nodes[:1], cluster.Nodes[2:]...)
		err = ckecliClusterSet(cluster)
		Expect(err).NotTo(HaveOccurred())
		runCKE(ckeImageURL)
		waitServerStatusCompletion()

		By("Testing default/kubernetes Endpoints")
		out, _, err = kubectl("get", "-o=json", "endpoints/kubernetes")
		Expect(err).ShouldNot(HaveOccurred())
		ep = corev1.Endpoints{}
		err = json.Unmarshal(out, &ep)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(ep.Subsets).Should(HaveLen(1))
		if isDegraded {
			Expect(ep.Subsets[0].Addresses).Should(ConsistOf(
				corev1.EndpointAddress{IP: node1},
			))
			Expect(ep.Subsets[0].NotReadyAddresses).Should(ConsistOf(
				corev1.EndpointAddress{IP: node3},
			))
		} else {
			Expect(ep.Subsets[0].Addresses).Should(ConsistOf(
				corev1.EndpointAddress{IP: node1},
				corev1.EndpointAddress{IP: node3},
			))
			Expect(ep.Subsets[0].NotReadyAddresses).Should(BeEmpty())
		}

		By("Adding a new node to the cluster as a control plane")
		// this will run AddMemberOp as well as other boot/restart ops.

		// inject failure into AddMemberOp to cause leader change
		firstLeader := strings.TrimSpace(string(ckecliSafe("leader")))
		Expect(firstLeader).To(Or(Equal("host1"), Equal("host2")))
		injectFailure("etcdAfterMemberAdd")

		ckecliSafe("constraints", "set", "control-plane-count", "3")
		cluster = getCluster(0, 1, 2)
		cluster.Nodes = append(cluster.Nodes, &cke.Node{
			Address: node6,
			User:    "cybozu",
		})
		Expect(cluster.Validate(false)).NotTo(HaveOccurred())
		cluster.Options.Kubelet.BootTaints = []corev1.Taint{
			{
				Key:    "coil.cybozu.com/bootstrap",
				Effect: corev1.TaintEffectNoSchedule,
			},
		}
		clusterSetAndWait(cluster)

		// reboot node2 and node4 to check bootstrap taints
		rebootTime := time.Now()
		var rebootedNodes []string
		if isDegraded {
			rebootedNodes = []string{node4}
		} else {
			rebootedNodes = []string{node2, node4}
		}
		for _, n := range rebootedNodes {
			execAt(n, "sudo", "systemd-run", "reboot", "-f", "-f")
		}

		Eventually(func() error {
			for _, n := range rebootedNodes {
				err := reconnectSSH(n)
				if err != nil {
					return err
				}
				since := fmt.Sprintf("-%dmin", int(time.Since(rebootTime).Minutes()+1.0))
				stdout, _, err := execAt(n, "last", "reboot", "-s", since)
				if err != nil {
					return err
				}
				if !strings.Contains(string(stdout), "reboot") {
					return fmt.Errorf("node: %s is not still reboot", n)
				}
			}
			return nil
		}).Should(Succeed())

		waitServerStatusCompletion()

		// check node6 is added
		var status *cke.ClusterStatus
		Eventually(func() error {
			var err error
			status, _, err = getClusterStatus(cluster)
			if err != nil {
				return err
			}
			for _, n := range status.Kubernetes.Nodes {
				nodeReady := false
				for _, cond := range n.Status.Conditions {
					if cond.Type == corev1.NodeReady && cond.Status == corev1.ConditionTrue {
						nodeReady = true
						break
					}
				}
				if !nodeReady {
					return errors.New("node is not ready: " + n.Name)
				}
			}
			var numKubernetesNodes int
			if isDegraded {
				numKubernetesNodes = len(cluster.Nodes) - 2 // 2 == (dummy 7th node) + (halted 2nd node)
			} else {
				numKubernetesNodes = len(cluster.Nodes)
			}
			if len(status.Kubernetes.Nodes) != numKubernetesNodes {
				return fmt.Errorf("nodes length should be %d, actual %d", numKubernetesNodes, len(status.Kubernetes.Nodes))
			}
			return nil
		}).Should(Succeed())

		// check bootstrap taints for node6
		// also check bootstrap taints for node2 and node4
		// node6: case of adding new node
		// node2: case of rebooting node with prior removal of Node resource
		// node4: case of rebooting node without prior manipulation on Node resource
		Eventually(func() error {
			for _, n := range status.Kubernetes.Nodes {
				if n.Name != node6 && n.Name != node2 && n.Name != node4 {
					continue
				}

				if len(n.Spec.Taints) != 1 {
					return errors.New("taints length should 1: " + n.Name)
				}
				taint := n.Spec.Taints[0]
				if taint.Key != "coil.cybozu.com/bootstrap" {
					return errors.New(`taint.Key != "coil.cybozu.com/bootstrap"`)
				}
				if taint.Value != "" {
					return errors.New("taint.Value is not empty: " + taint.Value)
				}
				if taint.Effect != corev1.TaintEffectNoSchedule {
					return errors.New("taint.Effect is not NoSchedule: " + string(taint.Effect))
				}
			}
			return nil
		}).Should(Succeed())

		// check leader change
		// AddMemberOp will not be called if degraded, and leader will not change
		if !isDegraded {
			newLeader := strings.TrimSpace(string(ckecliSafe("leader")))
			Expect(newLeader).To(Or(Equal("host1"), Equal("host2")))
			Expect(newLeader).NotTo(Equal(firstLeader))
			stopCKE()
			runCKE(ckeImageURL)
		}

		By("Converting a control plane node to a worker node")
		// this will run these ops:
		// - EtcdDestroyMemberOp
		// - APIServerStopOp
		// - ControllerManagerStopOp
		// - SchedulerStopOp
		// - EtcdStopOp

		Eventually(func() error {
			stdout, _, err := ckecli("leader")
			if err != nil {
				return err
			}
			leader := strings.TrimSpace(string(stdout))
			if leader != "host1" && leader != "host2" {
				return errors.New("unexpected leader: " + leader)
			}
			firstLeader = leader
			return nil
		}).Should(Succeed())
		// inject failure into RemoveMemberOp
		injectFailure("etcdAfterMemberRemove")

		ckecliSafe("constraints", "set", "control-plane-count", "2")
		cluster = getCluster(0, 1)
		clusterSetAndWait(cluster)

		// check control plane label
		out, _, err = kubectl("get", "-o=json", "node", node3)
		Expect(err).ShouldNot(HaveOccurred())
		var node corev1.Node
		err = json.Unmarshal(out, &node)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(node.Labels).ShouldNot(HaveKey("cke.cybozu.com/master"))

		// check leader change
		// RemoveMemberOp will not be called if degraded, and leader will not change
		if !isDegraded {
			newLeader := strings.TrimSpace(string(ckecliSafe("leader")))
			Expect(newLeader).To(Or(Equal("host1"), Equal("host2")))
			Expect(newLeader).NotTo(Equal(firstLeader))
			stopCKE()
			runCKE(ckeImageURL)
		}

		By("Changing service options")
		// this will run these ops:
		// - EtcdRestartOp
		// - ControllerManagerRestartOp
		// - SchedulerRestartOp
		// - KubeProxyRestartOp
		// - KubeletRestartOp
		// - KubeClusterDNSUpdateOp
		cluster.Options.Etcd.ExtraEnvvar = map[string]string{"AAA": "aaa"}
		cluster.Options.ControllerManager.ExtraEnvvar = map[string]string{"AAA": "aaa"}
		cluster.Options.Scheduler.ExtraEnvvar = map[string]string{"AAA": "aaa"}
		cluster.Options.Proxy.ExtraEnvvar = map[string]string{"AAA": "aaa"}
		cluster.Options.Kubelet.ExtraEnvvar = map[string]string{"AAA": "aaa"}
		config := &unstructured.Unstructured{}
		config.SetGroupVersionKind(kubeletv1beta1.SchemeGroupVersion.WithKind("KubeletConfiguration"))
		config.Object["clusterDomain"] = "neco.neco"
		cluster.Options.Kubelet.Config = config
		clusterSetAndWait(cluster)
	})

	It("updates Node resources", func() {
		By("adding non-existent labels, annotations, and taints")
		cluster := getCluster(0, 1, 2)
		cluster.Nodes[0].Labels = map[string]string{"label1": "value"}
		cluster.Nodes[0].Annotations = map[string]string{"annotation1": "value"}
		cluster.Nodes[0].Taints = []corev1.Taint{
			{
				Key:    "taint1",
				Value:  "value",
				Effect: corev1.TaintEffectNoSchedule,
			},
			{
				Key:    "hoge.cke.cybozu.com/foo",
				Value:  "bar",
				Effect: corev1.TaintEffectNoExecute,
			},
		}
		clusterSetAndWait(cluster)

		By("not removing existing labels, annotations, and taints")
		cluster = getCluster(0, 1, 2)
		cluster.Nodes[0].Labels = map[string]string{"label2": "value2"}
		clusterSetAndWait(cluster)

		out, stderr, err := kubectl("get", "nodes/"+node1, "-o", "json")
		Expect(err).NotTo(HaveOccurred(), "stdout: %s, stderr: %s", out, stderr)
		var node corev1.Node
		err = json.Unmarshal(out, &node)
		Expect(err).NotTo(HaveOccurred())
		Expect(node.Labels["label1"]).To(Equal("value"))
		Expect(node.Labels["label2"]).To(Equal("value2"))
		Expect(node.Annotations["annotation1"]).To(Equal("value"))
		Expect(node.Spec.Taints).To(HaveLen(1))
		Expect(node.Spec.Taints[0].Key).To(Equal("taint1"))

		By("updating existing labels, annotations, and taints")
		cluster = getCluster(0, 1, 2)
		cluster.Nodes[0].Labels = map[string]string{"label1": "updated"}
		cluster.Nodes[0].Annotations = map[string]string{
			"annotation1": "updated",
			"annotation2": "2",
		}
		cluster.Nodes[0].Taints = []corev1.Taint{
			{
				Key:    "taint2",
				Value:  "2",
				Effect: corev1.TaintEffectNoExecute,
			},
			{
				Key:    "taint1",
				Value:  "updated",
				Effect: corev1.TaintEffectNoExecute,
			},
		}
		cluster.TaintCP = true
		clusterSetAndWait(cluster)

		By("testing control plane taints")
		var runningCPs []string
		if isDegraded {
			// node2 has been removed from cluster once, and it cannot be restored in degraded mode
			runningCPs = []string{node1, node3}
		} else {
			runningCPs = []string{node1, node2, node3}
		}
		for _, n := range runningCPs {
			out, stderr, err := kubectl("get", "-o=json", "node", n)
			Expect(err).ShouldNot(HaveOccurred(), "stdout: %s, stderr: %s", out, stderr)
			var node corev1.Node
			err = json.Unmarshal(out, &node)
			Expect(err).ShouldNot(HaveOccurred(), "stdout: %s", out)
			Expect(node.Spec.Taints).Should(ContainElement(corev1.Taint{
				Key:    "cke.cybozu.com/master",
				Effect: corev1.TaintEffectPreferNoSchedule,
			}))
		}
		for _, n := range []string{node4, node5} {
			out, stderr, err := kubectl("get", "-o=json", "node", n)
			Expect(err).ShouldNot(HaveOccurred(), "stdout: %s, stderr: %s", out, stderr)
			var node corev1.Node
			err = json.Unmarshal(out, &node)
			Expect(err).ShouldNot(HaveOccurred(), "stdout: %s", out)
			Expect(node.Spec.Taints).ShouldNot(ContainElement(corev1.Taint{
				Key:    "cke.cybozu.com/master",
				Effect: corev1.TaintEffectPreferNoSchedule,
			}))
		}

		By("adding hostname")
		cluster = getCluster(0, 1, 2)
		cluster.Nodes[0].Hostname = "node1"
		clusterSetAndWait(cluster)

		Eventually(func() error {
			status, _, err := getClusterStatus(cluster)
			if err != nil {
				return err
			}

			var targetNode *corev1.Node
			for _, n := range status.Kubernetes.Nodes {
				if n.Name == "node1" {
					targetNode = &n
					break
				}
			}
			if targetNode == nil {
				return errors.New("node1 was not found")
			}

			nodeReady := false
			for _, cond := range targetNode.Status.Conditions {
				if cond.Type == corev1.NodeReady && cond.Status == corev1.ConditionTrue {
					nodeReady = true
					break
				}
			}
			if !nodeReady {
				return errors.New("node1 is not ready")
			}
			return nil
		}).Should(Succeed())

		By("clearing CNI configuration file directory")
		_, _, err = execAt(node1, "test", "-f", dummyCNIConf)
		Expect(err).Should(HaveOccurred())
	})

	It("removes all taints", func() {
		kubectl("taint", "--all=true", "node", "coil.cybozu.com/bootstrap-")
		kubectl("taint", "--all=true", "node", "taint1-")
		kubectl("taint", "--all=true", "node", "taint2-")
	})

	It("should recognize nodes that have recovered", func() {
		By("removing a worker node")
		cluster := getCluster(0, 1, 2)
		// remove node4
		cluster.Nodes = append(cluster.Nodes[:3], cluster.Nodes[4:]...)
		clusterSetAndWait(cluster)

		By("recovering the cluster")
		cluster = getCluster(0, 1, 2)
		clusterSetAndWait(cluster)

		stdout, stderr, err := kubectl("get", "nodes", "-o=json")
		Expect(err).ShouldNot(HaveOccurred(), "stdout: %s, stderr: %s", stdout, stderr)

		By("confirming that the removed node rejoined the cluster: " + node4)
		var nodes corev1.NodeList
		err = json.Unmarshal(stdout, &nodes)
		Expect(err).ShouldNot(HaveOccurred())

		var isFound bool
		for _, n := range nodes.Items {
			if node4 == n.Name {
				isFound = true
				break
			}
		}
		Expect(isFound).To(BeTrue())
	})

	It("should exclude updateOp of the shutdowned node", func() {
		if isDegraded {
			return
		}

		By("Terminating a control plane")
		stopCKE()
		execAt(node2, "sudo", "systemd-run", "halt", "-f", "-f")
		Eventually(func() error {
			_, err := execAtLocal("ping", "-c", "1", "-W", "1", node2)
			return err
		}).ShouldNot(Succeed())
		runCKE(ckeImageURL)
		waitServerStatusCompletion()

		By("Recovering the cluster by promoting a worker")
		cluster := getCluster(0, 2, 3)
		clusterSetAndWait(cluster)
	})
}
