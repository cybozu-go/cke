package mtest

import (
	"errors"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
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
		// - ToDo: K8sRemoveNodeOp
		// - RiversRestartOp
		// - APIServerRestartOp
		// - KubeEtcdEndpointsUpdateOp
		stopCKE()
		execAt(node2, "sudo", "-b", "reboot", "-f", "-f")
		time.Sleep(5 * time.Second)
		ckecli("constraints", "set", "control-plane-count", "2")
		cluster.Nodes = append(cluster.Nodes[:1], cluster.Nodes[2:]...)
		ckecliClusterSet(cluster)
		runCKE()
		Eventually(func() error {
			return checkCluster(cluster)
		}).Should(Succeed())
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
		ckecliClusterSet(cluster)
		Eventually(func() error {
			return checkCluster(cluster)
		}).Should(Succeed())
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
})
