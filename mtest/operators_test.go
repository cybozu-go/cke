package mtest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/cke/op"
	"github.com/cybozu-go/well"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	kubeletv1beta1 "k8s.io/kubelet/config/v1beta1"
)

func getRebootEntries() ([]*cke.RebootQueueEntry, error) {
	var entries []*cke.RebootQueueEntry
	data, stderr, err := ckecli("reboot-queue", "list")
	if err != nil {
		return nil, fmt.Errorf("%w, stdout: %s, stderr: %s", err, data, stderr)
	}
	err = json.Unmarshal(data, &entries)
	if err != nil {
		return nil, err
	}
	return entries, nil
}

func numRebootEntries() (int, error) {
	entries, err := getRebootEntries()
	if err != nil {
		return 0, err
	}
	return len(entries), nil
}

func waitRebootCompletion(cluster *cke.Cluster) {
	ts := time.Now()
	EventuallyWithOffset(1, func() error {
		num, err := numRebootEntries()
		if err != nil {
			return err
		}
		if num != 0 {
			return fmt.Errorf("reboot entry is remaining")
		}
		return checkCluster(cluster, ts)
	}).Should(Succeed())
}

func rebootShouldNotProceed() {
	ConsistentlyWithOffset(1, func() error {
		num, err := numRebootEntries()
		if err != nil {
			return err
		}
		if num != 0 {
			return fmt.Errorf("reboot entry is remaining")
		}
		return nil
	}, time.Second*60).Should(HaveOccurred())
}

func nodesShouldBeSchedulable(nodes ...string) {
	EventuallyWithOffset(1, func() error {
		for _, name := range nodes {
			stdout, stderr, err := kubectl("get", "nodes", name, "-o=json")
			if err != nil {
				return fmt.Errorf("stderr: %s, err: %w", stderr, err)
			}
			var node corev1.Node
			err = json.Unmarshal(stdout, &node)
			if err != nil {
				return err
			}
			if node.Spec.Unschedulable {
				return fmt.Errorf("node %s is unschedulable", name)
			}
			if node.Annotations[op.CKEAnnotationReboot] == "true" {
				return fmt.Errorf("node %s is annotated as a reboot target", name)
			}
		}
		return nil
	}).Should(Succeed())
}

func testRebootOperations(cluster *cke.Cluster) {
	By("Rebooting nodes")
	rebootTargets := node1
	_, _, err := ckecliWithInput([]byte(rebootTargets), "reboot-queue", "add", "-")
	Expect(err).ShouldNot(HaveOccurred())
	rebootTargets = node2 + "\n" + node4
	_, _, err = ckecliWithInput([]byte(rebootTargets), "reboot-queue", "add", "-")
	Expect(err).ShouldNot(HaveOccurred())
	waitRebootCompletion(cluster)
	nodesShouldBeSchedulable(node1, node2, node4)

	By("Reboot operation will stuck if node does not boot up")
	originalBootCheckCommand := cluster.Reboot.BootCheckCommand
	cluster.Reboot.BootCheckCommand = []string{"bash", "-c", "echo 'false'"}
	_, err = ckecliClusterSet(cluster)
	Expect(err).ShouldNot(HaveOccurred())
	// wait for the previous reconciliation to be done
	time.Sleep(time.Second * 3)

	rebootTargets = node1
	_, _, err = ckecliWithInput([]byte(rebootTargets), "reboot-queue", "add", "-")
	Expect(err).ShouldNot(HaveOccurred())
	rebootShouldNotProceed()

	cluster.Reboot.BootCheckCommand = originalBootCheckCommand
	_, err = ckecliClusterSet(cluster)
	Expect(err).ShouldNot(HaveOccurred())
	waitRebootCompletion(cluster)

	By("ckecli reboot-queue disable disables reboot queue processing")
	ckecliSafe("reboot-queue", "disable")
	// wait for the previous reconciliation to be done
	time.Sleep(time.Second * 3)
	_, _, err = ckecliWithInput([]byte(node1), "reboot-queue", "add", "-")
	Expect(err).ShouldNot(HaveOccurred())
	rebootShouldNotProceed()

	By("ckecli reboot-queue enable enables reboot queue processing")
	ckecliSafe("reboot-queue", "enable")
	waitRebootCompletion(cluster)
	nodesShouldBeSchedulable(node1)

	By("ckecli reboot-queue cancel cancels the specified reboot queue entry")
	ckecliSafe("reboot-queue", "disable")
	_, _, err = ckecliWithInput([]byte(node1), "reboot-queue", "add", "-")
	Expect(err).ShouldNot(HaveOccurred())
	ckecliSafe("reboot-queue", "cancel", "5")
	entries, err := getRebootEntries()
	Expect(err).ShouldNot(HaveOccurred())
	Expect(entries).Should(HaveLen(1))
	Expect(entries[0].Status).To(Equal(cke.RebootStatusCancelled))

	ckecliSafe("reboot-queue", "enable")
	waitRebootCompletion(cluster)

	By("ckecli reboot-queue cancel-all cancels all the reboot queue entries")
	ckecliSafe("reboot-queue", "disable")
	_, _, err = ckecliWithInput([]byte(node1), "reboot-queue", "add", "-")
	Expect(err).ShouldNot(HaveOccurred())
	_, _, err = ckecliWithInput([]byte(node2), "reboot-queue", "add", "-")
	Expect(err).ShouldNot(HaveOccurred())
	ckecliSafe("reboot-queue", "cancel-all")
	entries, err = getRebootEntries()
	Expect(err).ShouldNot(HaveOccurred())
	Expect(entries).Should(HaveLen(2))
	Expect(entries[0].Status).To(Equal(cke.RebootStatusCancelled))
	Expect(entries[1].Status).To(Equal(cke.RebootStatusCancelled))

	ckecliSafe("reboot-queue", "enable")
	waitRebootCompletion(cluster)

	By("Preparing a deployment to test protected_namespaces")
	_, stderr, err := kubectl("create", "namespace", "reboot-test")
	Expect(err).ShouldNot(HaveOccurred(), "stderr: %s", stderr)
	_, stderr, err = kubectl("label", "namespaces", "reboot-test", "reboot-test=sample")
	Expect(err).ShouldNot(HaveOccurred(), "stderr: %s", stderr)
	_, stderr, err = kubectlWithInput(rebootDeploymentYAML, "apply", "-f", "-")
	Expect(err).ShouldNot(HaveOccurred(), "stderr: %s", stderr)

	Eventually(func() error {
		out, _, err := kubectl("get", "-n=reboot-test", "deployments/sample", "-o=json")
		if err != nil {
			return err
		}

		var deploy appsv1.Deployment
		err = json.Unmarshal(out, &deploy)
		if err != nil {
			return err
		}
		if deploy.Status.ReadyReplicas != 3 {
			return fmt.Errorf("deployment is not ready")
		}
		return nil
	}).Should(Succeed())

	out, _, err := kubectl("get", "-n=reboot-test", "pod", "-l=reboot-app=sample", "-o=json")
	Expect(err).ShouldNot(HaveOccurred(), "stderr: %s", stderr)
	var deploymentPods corev1.PodList
	err = json.Unmarshal(out, &deploymentPods)
	Expect(err).ShouldNot(HaveOccurred())
	Expect(deploymentPods.Items).Should(HaveLen(3))

	By("Reboot operation will protect all pods if protected_namespaces is nil")
	nodeName := deploymentPods.Items[0].Spec.NodeName
	_, _, err = ckecliWithInput([]byte(nodeName), "reboot-queue", "add", "-")
	Expect(err).ShouldNot(HaveOccurred())
	rebootShouldNotProceed()

	env := well.NewEnvironment(context.Background())
	env.Go(func(ctx context.Context) error {
		for i := 0; i < 5; i++ {
			// ignore error because the entry may have been deleted
			ckecli("reboot-queue", "cancel-all")
			time.Sleep(time.Second)
		}
		return nil
	})
	env.Stop()
	env.Wait()
	waitRebootCompletion(cluster)
	nodesShouldBeSchedulable(nodeName)

	By("Reboot operation will protect pods in protected namespaces")
	cluster.Reboot.ProtectedNamespaces = &metav1.LabelSelector{
		MatchLabels: map[string]string{"reboot-test": "sample"},
	}
	_, err = ckecliClusterSet(cluster)
	Expect(err).ShouldNot(HaveOccurred())

	nodeName = deploymentPods.Items[1].Spec.NodeName
	_, _, err = ckecliWithInput([]byte(nodeName), "reboot-queue", "add", "-")
	Expect(err).ShouldNot(HaveOccurred())
	rebootShouldNotProceed()

	env = well.NewEnvironment(context.Background())
	env.Go(func(ctx context.Context) error {
		for i := 0; i < 5; i++ {
			// ignore error because the entry may have been deleted
			ckecli("reboot-queue", "cancel-all")
			time.Sleep(time.Second)
		}
		return nil
	})
	env.Stop()
	env.Wait()
	waitRebootCompletion(cluster)
	nodesShouldBeSchedulable(nodeName)

	By("Reboot operation deletes non-protected pods")
	cluster.Reboot.ProtectedNamespaces = &metav1.LabelSelector{
		MatchLabels: map[string]string{"reboot-test": "test"},
	}
	_, err = ckecliClusterSet(cluster)
	Expect(err).ShouldNot(HaveOccurred())

	nodeName = deploymentPods.Items[2].Spec.NodeName
	_, _, err = ckecliWithInput([]byte(nodeName), "reboot-queue", "add", "-")
	Expect(err).ShouldNot(HaveOccurred())
	waitRebootCompletion(cluster)
	nodesShouldBeSchedulable(nodeName)

	By("Deleting a deployment")
	_, stderr, err = kubectlWithInput(rebootDeploymentYAML, "delete", "-f", "-")
	Expect(err).ShouldNot(HaveOccurred(), "stderr: %s", stderr)
	cluster.Reboot.ProtectedNamespaces = nil
	_, err = ckecliClusterSet(cluster)
	Expect(err).ShouldNot(HaveOccurred())

	By("Reboot operation will protect running job-managed pods")
	_, stderr, err = kubectlWithInput(rebootJobRunningYAML, "apply", "-f", "-")
	Expect(err).ShouldNot(HaveOccurred(), "stderr: %s", stderr)

	var runningJobPod *corev1.Pod
	Eventually(func() error {
		out, _, err := kubectl("get", "-n=reboot-test", "pod", "-l=job-name=job-running", "-o=json")
		if err != nil {
			return err
		}

		var pods corev1.PodList
		err = json.Unmarshal(out, &pods)
		if err != nil {
			return err
		}
		if len(pods.Items) != 1 {
			return fmt.Errorf("pod is not created")
		}
		if pods.Items[0].Status.Phase != corev1.PodRunning {
			return fmt.Errorf("pod is not running")
		}
		runningJobPod = &pods.Items[0]
		return nil
	}).Should(Succeed())

	_, _, err = ckecliWithInput([]byte(runningJobPod.Spec.NodeName), "reboot-queue", "add", "-")
	Expect(err).ShouldNot(HaveOccurred())
	rebootShouldNotProceed()

	env = well.NewEnvironment(context.Background())
	env.Go(func(ctx context.Context) error {
		for i := 0; i < 5; i++ {
			// ignore error because the entry may have been deleted
			ckecli("reboot-queue", "cancel-all")
			time.Sleep(time.Second)
		}
		return nil
	})
	env.Stop()
	env.Wait()
	waitRebootCompletion(cluster)
	nodesShouldBeSchedulable(nodeName)

	_, stderr, err = kubectlWithInput(rebootJobRunningYAML, "delete", "-f", "-")
	Expect(err).ShouldNot(HaveOccurred(), "stderr: %s", stderr)

	By("Reboot operation will not delete completed job-managed pods")
	_, stderr, err = kubectlWithInput(rebootJobCompletedYAML, "apply", "-f", "-")
	Expect(err).ShouldNot(HaveOccurred(), "stderr: %s", stderr)

	var completedJobPod *corev1.Pod
	Eventually(func() error {
		out, _, err := kubectl("get", "-n=reboot-test", "pod", "-l=job-name=job-completed", "-o=json")
		if err != nil {
			return err
		}

		var pods corev1.PodList
		err = json.Unmarshal(out, &pods)
		if err != nil {
			return err
		}
		if len(pods.Items) != 1 {
			return fmt.Errorf("pod is not created")
		}
		if pods.Items[0].Status.Phase != corev1.PodSucceeded {
			return fmt.Errorf("pod is not succeeded")
		}
		completedJobPod = &pods.Items[0]
		return nil
	}).Should(Succeed())

	_, _, err = ckecliWithInput([]byte(completedJobPod.Spec.NodeName), "reboot-queue", "add", "-")
	Expect(err).ShouldNot(HaveOccurred())
	waitRebootCompletion(cluster)
	nodesShouldBeSchedulable(completedJobPod.Spec.NodeName)

	out, _, err = kubectl("get", "-n=reboot-test", "pod", "-l=job-name=job-completed", "-o=json")
	Expect(err).ShouldNot(HaveOccurred(), "stderr: %s", stderr)
	var afterPods corev1.PodList
	err = json.Unmarshal(out, &afterPods)
	Expect(err).ShouldNot(HaveOccurred())
	Expect(afterPods.Items).Should(HaveLen(1))
	Expect(afterPods.Items[0].UID).Should(Equal(completedJobPod.UID))

	_, stderr, err = kubectlWithInput(rebootJobCompletedYAML, "delete", "-f", "-")
	Expect(err).ShouldNot(HaveOccurred(), "stderr: %s", stderr)
	_, stderr, err = kubectl("delete", "namespace", "reboot-test")
	Expect(err).ShouldNot(HaveOccurred(), "stderr: %s", stderr)
}

func testOperators(isDegraded bool) {
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

		if !isDegraded {
			testRebootOperations(cluster)
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
		_, err = ckecliClusterSet(cluster)
		Expect(err).ShouldNot(HaveOccurred())

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
		if isDegraded {
			return
		}

		By("Preparing the cluster with available nodes")
		ckecliSafe("constraints", "set", "control-plane-count", "3")
		cluster := getCluster()
		for i := 0; i < 3; i++ {
			cluster.Nodes[i].ControlPlane = true
		}
		clusterSetAndWait(cluster)

		By("Terminating a control plane")

		stopCKE()
		execAt(node2, "sudo", "systemd-run", "halt", "-f", "-f")
		Eventually(func() error {
			_, err := execAtLocal("ping", "-c", "1", "-W", "1", node2)
			return err
		}).ShouldNot(Succeed())
		runCKE(ckeImageURL)
		clusterSetAndWait(cluster)

		By("Recovering the cluster by promoting a worker")
		cluster = getCluster()
		for i := range []int{0, 2, 3} {
			cluster.Nodes[i].ControlPlane = true
		}
		clusterSetAndWait(cluster)
	})
}
