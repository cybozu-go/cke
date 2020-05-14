package mtest

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/cke/op"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
)

// TestOperators tests all CKE operators
func TestOperators(isDegraded bool) {
	AfterEach(initializeControlPlane)

	It("run all operators / commanders", func() {
		By("Preparing the cluster")
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

		cluster := getCluster()
		for i := 0; i < 3; i++ {
			cluster.Nodes[i].ControlPlane = true
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
			ts := time.Now()
			runCKE(ckeImageURL)
			Eventually(func() error {
				return checkCluster(cluster, ts)
			}).Should(Succeed())
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
		cluster.Nodes = append(cluster.Nodes[:1], cluster.Nodes[2:]...)
		ts, err := ckecliClusterSet(cluster)
		Expect(err).ShouldNot(HaveOccurred())
		runCKE(ckeImageURL)
		Eventually(func() error {
			return checkCluster(cluster, ts)
		}).Should(Succeed())

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
		injectFailure("op/etcd/etcdAfterMemberAdd")

		ckecliSafe("constraints", "set", "control-plane-count", "3")
		cluster = getCluster()
		for i := 0; i < 3; i++ {
			cluster.Nodes[i].ControlPlane = true
		}
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
		ckecliClusterSet(cluster)

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
		ts = time.Now()
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

		Eventually(func() error {
			return checkCluster(cluster, ts)
		}).Should(Succeed())

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
		injectFailure("op/etcd/etcdAfterMemberRemove")

		ckecliSafe("constraints", "set", "control-plane-count", "2")
		cluster = getCluster()
		for i := 0; i < 2; i++ {
			cluster.Nodes[i].ControlPlane = true
		}
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

		By("Changing options")
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
		cluster.Options.Kubelet.Domain = "neconeco"
		clusterSetAndWait(cluster)

		By("Adding a scheduler extender")
		// this will run these ops:
		// - SchedulerRestartOp
		cluster.Options.Scheduler.Extenders = []string{"urlPrefix: http://127.0.0.1:8000"}
		clusterSetAndWait(cluster)

		stdout, stderr, err := execAt(node1, "jq", "-r", "'.extenders[0].urlPrefix'", op.PolicyConfigPath)
		Expect(err).NotTo(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)
		Expect(strings.TrimSpace(string(stdout))).To(Equal("http://127.0.0.1:8000"))
	})

	It("updates Node resources", func() {
		By("adding non-existent labels, annotations, and taints")
		cluster := getCluster()
		for i := 0; i < 3; i++ {
			cluster.Nodes[i].ControlPlane = true
		}
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
		cluster = getCluster()
		for i := 0; i < 3; i++ {
			cluster.Nodes[i].ControlPlane = true
		}
		cluster.Nodes[0].Labels = map[string]string{"label2": "value2"}
		ts, err := ckecliClusterSet(cluster)
		Expect(err).ShouldNot(HaveOccurred())
		Eventually(func() error {
			err := checkCluster(cluster, ts)
			if err != nil {
				return err
			}

			stdout, stderr, err := kubectl("get", "nodes/"+node1, "-o", "json")
			if err != nil {
				return fmt.Errorf("stdout:%s, stderr:%s", stdout, stderr)
			}
			var node corev1.Node
			err = json.Unmarshal(stdout, &node)
			if err != nil {
				return err
			}
			if node.Labels["label1"] != "value" {
				return fmt.Errorf(`expect node.Labels["label1"] to be "value", but actual: %s`, node.Labels["label1"])
			}
			if node.Labels["label2"] != "value2" {
				return fmt.Errorf(`expect node.Labels["label2"] to be "value2", but actual: %s`, node.Labels["label2"])
			}
			if node.Annotations["annotation1"] != "value" {
				return fmt.Errorf(`expect node.Labels["annotation1"] to be "value", but actual: %s`, node.Annotations["annotation1"])
			}
			if len(node.Spec.Taints) != 1 {
				return fmt.Errorf(`expect len(node.Spec.Taints) to be 1, but actual: %d`, len(node.Spec.Taints))
			}
			taint := node.Spec.Taints[0]
			if taint.Key != "taint1" {
				return fmt.Errorf(`expect taint.Key to be "taint1", but actual: %s`, taint.Key)
			}
			return nil
		}).Should(Succeed())

		By("updating existing labels, annotations, and taints")
		cluster = getCluster()
		for i := 0; i < 3; i++ {
			cluster.Nodes[i].ControlPlane = true
		}
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
		cluster = getCluster()
		for i := 0; i < 3; i++ {
			cluster.Nodes[i].ControlPlane = true
		}
		cluster.Nodes[0].Hostname = "node1"
		ts, err = ckecliClusterSet(cluster)
		Expect(err).ShouldNot(HaveOccurred())
		Eventually(func() error {
			err := checkCluster(cluster, ts)
			if err != nil {
				return err
			}
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
		cluster := getCluster()
		for i := 0; i < 3; i++ {
			cluster.Nodes[i].ControlPlane = true
		}
		// remove node4
		cluster.Nodes = append(cluster.Nodes[:3], cluster.Nodes[4:]...)
		clusterSetAndWait(cluster)

		By("recovering the cluster")
		cluster = getCluster()
		for i := 0; i < 3; i++ {
			cluster.Nodes[i].ControlPlane = true
		}
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
		if !isDegraded {
			return
		}

		By("Preparing the cluster with available nodes")
		stopCKE()
		ckecliSafe("constraints", "set", "control-plane-count", "3")
		cluster := getCluster()
		cluster.Nodes[0].ControlPlane = true
		cluster.Nodes[3].ControlPlane = true
		cluster.Nodes[4].ControlPlane = true
		ts, err := ckecliClusterSet(cluster)
		Expect(err).ShouldNot(HaveOccurred())
		runCKE(ckeImageURL)
		Eventually(func() error {
			return checkCluster(cluster, ts)
		}).Should(Succeed())

		By("Terminating a control plane")
		execAt(node4, "sudo", "systemd-run", "halt", "-f", "-f")
		Eventually(func() error {
			_, err := execAtLocal("ping", "-c", "1", "-W", "1", node4)
			return err
		}).ShouldNot(Succeed())
		Eventually(func() error {
			return checkClusterAtPhase(cluster, ts, cke.PhaseEtcdBootAborted)
		}).Should(Succeed())

		By("recovering the cluster")
		stopCKE()
		ckecliSafe("constraints", "set", "control-plane-count", "2")
		cluster = getCluster()
		cluster.Nodes[0].ControlPlane = true
		cluster.Nodes[4].ControlPlane = true
		ts, err = ckecliClusterSet(cluster)
		Expect(err).ShouldNot(HaveOccurred())
		runCKE(ckeImageURL)
		Eventually(func() error {
			return checkCluster(cluster, ts)
		}).Should(Succeed())
	})
}
