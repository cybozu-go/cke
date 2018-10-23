package mtest

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/cybozu-go/cke"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
)

var _ = Describe("Operations", func() {
	AfterEach(initializeControlPlane)

	It("run all operators / commanders", func() {
		By("Preparing the cluster")
		// these operators ran already:
		// - RiversBootOp
		// - EtcdBootOp
		// - APIServerBootOp
		// - ControllerManagerBootOp
		// - SchedulerBootOp
		// - KubeletBootOp
		// - KubeProxyBootOp
		// - KubeWaitOp
		// - KubeRBACRoleInstallOp
		// - KubeEtcdEndpointsCreateOp

		By("Stopping etcd servers")
		// this will run:
		// - EtcdStartOp
		// - EtcdWaitClusterOp
		stopCKE()
		execSafeAt(node2, "docker", "stop", "etcd")
		execSafeAt(node2, "docker", "rm", "etcd")
		execSafeAt(node3, "docker", "stop", "etcd")
		execSafeAt(node3, "docker", "rm", "etcd")
		runCKE()
		cluster := getCluster()
		for i := 0; i < 3; i++ {
			cluster.Nodes[i].ControlPlane = true
		}
		Eventually(func() error {
			return checkCluster(cluster)
		}).Should(Succeed())

		By("Removing a control plane node from the cluster")
		// this will run:
		// - EtcdRemoveMemberOp
		// - KubeNodeRemoveOp
		// - RiversRestartOp
		// - APIServerRestartOp
		// - KubeEtcdEndpointsUpdateOp
		stopCKE()
		ckecli("constraints", "set", "control-plane-count", "2")
		cluster.Nodes = append(cluster.Nodes[:1], cluster.Nodes[2:]...)
		ckecliClusterSet(cluster)
		runCKE()
		Eventually(func() error {
			return checkCluster(cluster)
		}).Should(Succeed())

		execAt(node2, "sudo", "reboot", "-f", "-f")
		time.Sleep(5 * time.Second)
		Expect(reconnectSSH(node2)).NotTo(HaveOccurred())

		By("Adding a new node to the cluster as a control plane")
		// this will run EtcdAddMemberOp as well as other boot/restart ops.

		// inject failure into addEtcdMemberCommand to cause leader change
		firstLeader := strings.TrimSpace(string(ckecli("leader")))
		Expect(firstLeader).To(Or(Equal("host1"), Equal("host2")))
		injectFailure("op/etcdAfterMemberAdd")

		ckecli("constraints", "set", "control-plane-count", "3")
		cluster = getCluster()
		for i := 0; i < 3; i++ {
			cluster.Nodes[i].ControlPlane = true
		}
		cluster.Nodes = append(cluster.Nodes, &cke.Node{
			Address: node6,
			User:    "cybozu",
		})
		Expect(cluster.Validate()).NotTo(HaveOccurred())
		cluster.Options.Kubelet.BootTaints = []corev1.Taint{
			{
				Key:    "coil.cybozu.com/bootstrap",
				Effect: corev1.TaintEffectNoSchedule,
			},
		}
		ckecliClusterSet(cluster)
		Eventually(func() error {
			return checkCluster(cluster)
		}).Should(Succeed())

		// check node6 is added
		var status *cke.ClusterStatus
		Eventually(func() []corev1.Node {
			var err error
			status, err = getClusterStatus(cluster)
			if err != nil {
				return nil
			}
			return status.Kubernetes.Nodes
		}).Should(HaveLen(len(cluster.Nodes)))

		// check bootstrap taints for node6
		for _, n := range status.Kubernetes.Nodes {
			if n.Name != node6 {
				Expect(n.Spec.Taints).Should(BeEmpty())
				continue
			}

			Expect(n.Spec.Taints).Should(HaveLen(1))
			taint := n.Spec.Taints[0]
			Expect(taint.Key).Should(Equal("coil.cybozu.com/bootstrap"))
			Expect(taint.Value).Should(BeEmpty())
			Expect(taint.Effect).Should(Equal(corev1.TaintEffectNoSchedule))
		}

		// check leader change
		newLeader := strings.TrimSpace(string(ckecli("leader")))
		Expect(newLeader).To(Or(Equal("host1"), Equal("host2")))
		Expect(newLeader).NotTo(Equal(firstLeader))
		stopCKE()
		runCKE()

		By("Converting a control plane node to a worker node")
		// this will run these ops:
		// - EtcdDestroyMemberOp
		// - APIServerStopOp
		// - ControllerManagerStopOp
		// - SchedulerStopOp
		// - EtcdStopOp

		Eventually(func() error {
			stdout, err := ckecliUnsafe("leader")
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
		// inject failure into EtcdDestroyMemberOp
		injectFailure("op/etcdAfterMemberRemove")

		ckecli("constraints", "set", "control-plane-count", "2")
		cluster = getCluster()
		for i := 0; i < 2; i++ {
			cluster.Nodes[i].ControlPlane = true
		}
		ckecliClusterSet(cluster)
		Eventually(func() error {
			return checkCluster(cluster)
		}).Should(Succeed())
		// check leader change
		newLeader = strings.TrimSpace(string(ckecli("leader")))
		Expect(newLeader).To(Or(Equal("host1"), Equal("host2")))
		Expect(newLeader).NotTo(Equal(firstLeader))
		stopCKE()
		runCKE()

		By("Chainging options")
		// this will run these ops:
		// - EtcdRestartOp
		// - ControllerManagerRestartOp
		// - SchedulerRestartOp
		// - KubeProxyRestartOp
		// - KubeletRestartOp
		cluster.Options.Etcd.ExtraEnvvar = map[string]string{"AAA": "aaa"}
		cluster.Options.ControllerManager.ExtraEnvvar = map[string]string{"AAA": "aaa"}
		cluster.Options.Scheduler.ExtraEnvvar = map[string]string{"AAA": "aaa"}
		cluster.Options.Proxy.ExtraEnvvar = map[string]string{"AAA": "aaa"}
		cluster.Options.Kubelet.ExtraEnvvar = map[string]string{"AAA": "aaa"}
		ckecliClusterSet(cluster)
		Eventually(func() error {
			return checkCluster(cluster)
		}).Should(Succeed())
	})

	It("updates Node resources", func() {
		By("adding non-existent labels, annotations, and taints")
		cluster := getCluster()
		for i := 0; i < 3; i++ {
			cluster.Nodes[i].ControlPlane = true
		}
		cluster.Nodes[0].Labels = map[string]string{"label1": "value"}
		cluster.Nodes[0].Annotations = map[string]string{"annotation1": "value"}
		cluster.Nodes[0].Taints = []corev1.Taint{{
			Key:    "taint1",
			Value:  "value",
			Effect: corev1.TaintEffectNoSchedule,
		}}
		ckecliClusterSet(cluster)
		Eventually(func() error {
			return checkCluster(cluster)
		}).Should(Succeed())

		By("not removing existing labels, annotations, and taints")
		cluster = getCluster()
		for i := 0; i < 3; i++ {
			cluster.Nodes[i].ControlPlane = true
		}
		cluster.Nodes[0].Labels = map[string]string{"label2": "value2"}
		ckecliClusterSet(cluster)
		Eventually(func() error {
			return checkCluster(cluster)
		}).Should(Succeed())

		stdout, err := kubectl("get", "nodes/"+node1, "-o", "json")
		Expect(err).NotTo(HaveOccurred())

		var node corev1.Node
		err = json.Unmarshal(stdout, &node)
		Expect(err).NotTo(HaveOccurred())

		Expect(node.Labels["label1"]).Should(Equal("value"))
		Expect(node.Labels["label2"]).Should(Equal("value2"))
		Expect(node.Annotations["annotation1"]).Should(Equal("value"))
		Expect(node.Spec.Taints).To(HaveLen(1))
		taint := node.Spec.Taints[0]
		Expect(taint.Key).Should(Equal("taint1"))

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
		ckecliClusterSet(cluster)
		Eventually(func() error {
			return checkCluster(cluster)
		}).Should(Succeed())
	})
})
