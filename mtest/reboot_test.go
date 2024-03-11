package mtest

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/cke/op"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
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
		if num == 0 {
			return fmt.Errorf("reboot entry is empty")
		}
		return nil
	}, time.Second*60).Should(Succeed())
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

func testRebootOperations() {
	// this will run:
	// - RebootDrainStartOp
	// - RebootRebootOp
	// - RebootDrainTimeoutOp
	// - RebootUncordonOp
	// - RebootDequeueOp

	cluster := getCluster()
	for i := 0; i < 3; i++ {
		cluster.Nodes[i].ControlPlane = true
	}

	currentWriteIndex := 0
	rebootQueueAdd := func(nodes []string) {
		targets := strings.Join(nodes, "\n")
		_, _, err := ckecliWithInput([]byte(targets), "reboot-queue", "add", "-")
		ExpectWithOffset(1, err).ShouldNot(HaveOccurred())
		currentWriteIndex += len(nodes)
	}

	It("increases worker node", func() {
		cluster.Nodes = append(cluster.Nodes, &cke.Node{
			Address: node6,
			User:    "cybozu",
		})
		Expect(cluster.Validate(false)).NotTo(HaveOccurred())
		clusterSetAndWait(cluster)
	})

	It("checks basic reboot behavior", func() {
		By("Rebooting nodes")
		rebootQueueAdd([]string{node1})
		rebootQueueAdd([]string{node2, node4})
		waitRebootCompletion(cluster)
		nodesShouldBeSchedulable(node1, node2, node4)

		By("Reboot operation will get stuck if node does not boot up")
		originalBootCheckCommand := cluster.Reboot.BootCheckCommand
		cluster.Reboot.BootCheckCommand = []string{"bash", "-c", "echo 'false'"}
		_, err := ckecliClusterSet(cluster)
		Expect(err).ShouldNot(HaveOccurred())
		// wait for the previous reconciliation to be done
		time.Sleep(time.Second * 3)

		rebootQueueAdd([]string{node1})
		rebootShouldNotProceed()

		cluster.Reboot.BootCheckCommand = originalBootCheckCommand
		_, err = ckecliClusterSet(cluster)
		Expect(err).ShouldNot(HaveOccurred())
		waitRebootCompletion(cluster)

		By("ckecli reboot-queue disable disables reboot queue processing")
		ckecliSafe("reboot-queue", "disable")
		rebootQueueAdd([]string{node1})
		rebootShouldNotProceed()

		By("ckecli reboot-queue list the entries in the reboot queue")
		data := ckecliSafe("reboot-queue", "list")
		var entries []*cke.RebootQueueEntry
		err = json.Unmarshal(data, &entries)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(entries).Should(HaveLen(1))
		Expect(entries[0].Node).To(Equal(node1))

		By("ckecli reboot-queue enable enables reboot queue processing")
		ckecliSafe("reboot-queue", "enable")
		waitRebootCompletion(cluster)
		nodesShouldBeSchedulable(node1)

		By("ckecli reboot-queue cancel cancels the specified reboot queue entry")
		ckecliSafe("reboot-queue", "disable")
		rebootQueueAdd([]string{node1})
		ckecliSafe("reboot-queue", "cancel", fmt.Sprintf("%d", currentWriteIndex-1))
		entries, err = getRebootEntries()
		Expect(err).ShouldNot(HaveOccurred())
		Expect(entries).Should(HaveLen(1))
		Expect(entries[0].Status).To(Equal(cke.RebootStatusCancelled))

		ckecliSafe("reboot-queue", "enable")
		waitRebootCompletion(cluster)

		By("ckecli reboot-queue cancel-all cancels all the reboot queue entries")
		ckecliSafe("reboot-queue", "disable")
		rebootQueueAdd([]string{node1})
		rebootQueueAdd([]string{node2})
		ckecliSafe("reboot-queue", "cancel-all")
		entries, err = getRebootEntries()
		Expect(err).ShouldNot(HaveOccurred())
		Expect(entries).Should(HaveLen(2))
		Expect(entries[0].Status).To(Equal(cke.RebootStatusCancelled))
		Expect(entries[1].Status).To(Equal(cke.RebootStatusCancelled))

		ckecliSafe("reboot-queue", "enable")
		waitRebootCompletion(cluster)
	})

	It("checks Pod protection", func() {
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
		rebootQueueAdd([]string{nodeName})
		rebootShouldNotProceed()

		// Due to race condition between cke and `ckecli reboot-queue cancel/cancel-all`,
		// we need do it several times for stable test
		for i := 0; i < 5; i++ {
			// ignore error because the entry may have been deleted
			ckecli("reboot-queue", "cancel-all")
			time.Sleep(time.Second)
		}
		waitRebootCompletion(cluster)
		nodesShouldBeSchedulable(nodeName)

		By("Reboot operation will protect pods in protected namespaces")
		cluster.Reboot.ProtectedNamespaces = &metav1.LabelSelector{
			MatchLabels: map[string]string{"reboot-test": "sample"},
		}
		_, err = ckecliClusterSet(cluster)
		Expect(err).ShouldNot(HaveOccurred())

		nodeName = deploymentPods.Items[1].Spec.NodeName
		rebootQueueAdd([]string{nodeName})
		rebootShouldNotProceed()

		for i := 0; i < 5; i++ {
			// ignore error because the entry may have been deleted
			ckecli("reboot-queue", "cancel-all")
			time.Sleep(time.Second)
		}
		waitRebootCompletion(cluster)
		nodesShouldBeSchedulable(nodeName)

		By("Reboot operation deletes non-protected pods")
		cluster.Reboot.ProtectedNamespaces = &metav1.LabelSelector{
			MatchLabels: map[string]string{"reboot-test": "test"},
		}
		_, err = ckecliClusterSet(cluster)
		Expect(err).ShouldNot(HaveOccurred())

		nodeName = deploymentPods.Items[2].Spec.NodeName
		rebootQueueAdd([]string{nodeName})
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

		rebootQueueAdd([]string{runningJobPod.Spec.NodeName})
		rebootShouldNotProceed()

		for i := 0; i < 5; i++ {
			// ignore error because the entry may have been deleted
			ckecli("reboot-queue", "cancel-all")
			time.Sleep(time.Second)
		}
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

		rebootQueueAdd([]string{completedJobPod.Spec.NodeName})
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
	})

	It("checks drain timeout behavior", func() {
		By("Deploying a slow eviction pod")
		_, stderr, err := kubectlWithInput(rebootSlowEvictionDeploymentYAML, "apply", "-f", "-")
		Expect(err).ShouldNot(HaveOccurred(), "stderr: %s", stderr)

		Eventually(func() error {
			out, _, err := kubectl("get", "-n=reboot-test", "deployments/slow", "-o=json")
			if err != nil {
				return err
			}

			var deploy appsv1.Deployment
			err = json.Unmarshal(out, &deploy)
			if err != nil {
				return err
			}
			if deploy.Status.ReadyReplicas != 1 {
				return fmt.Errorf("deployment is not ready")
			}
			return nil
		}).Should(Succeed())

		out, _, err := kubectl("get", "-n=reboot-test", "pod", "-l=reboot-app=slow", "-o=json")
		Expect(err).ShouldNot(HaveOccurred(), "stderr: %s", stderr)
		var deploymentPods corev1.PodList
		err = json.Unmarshal(out, &deploymentPods)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(deploymentPods.Items).Should(HaveLen(1))

		By("Starting to reboot the node running the pod")
		nodeName := deploymentPods.Items[0].Spec.NodeName
		rebootQueueAdd([]string{nodeName})

		By("Checking the reboot entry becomes `draining` status")
		Eventually(func() error {
			re, err := getRebootEntries()
			if err != nil {
				return err
			}
			if len(re) != 1 {
				return fmt.Errorf("reboot queue should contain exactly 1 entry")
			}
			if re[0].Status != cke.RebootStatusDraining {
				return fmt.Errorf("reboot entry should have draining status")
			}
			if !re[0].DrainBackOffExpire.Equal(time.Time{}) {
				return fmt.Errorf("reboot entry should not have DrainBackOffExpire set yet")
			}
			return nil
		}).Should(Succeed())

		By("Sleeping until drain backoff")
		// a little longer than eviction_timeout_seconds, shorter than eviction_timeout_seconds + backoff
		time.Sleep(time.Second * time.Duration(*cluster.Reboot.EvictionTimeoutSeconds+10))

		By("Checking the reboot entry becomes back `queued` status")
		re, err := getRebootEntries()
		Expect(err).ShouldNot(HaveOccurred())
		Expect(re).Should(HaveLen(1))
		Expect(re[0].Status).Should(Equal(cke.RebootStatusQueued))
		Expect(re[0].DrainBackOffExpire).ShouldNot(Equal(time.Time{}))
		Expect(re[0].DrainBackOffCount).Should(Not(BeZero()))

		By("Checking reset-backoff command resets drain backoff")
		ckecliSafe("reboot-queue", "reset-backoff")
		re, err = getRebootEntries()
		Expect(err).ShouldNot(HaveOccurred())
		Expect(re).Should(HaveLen(1))
		Expect(re[0].DrainBackOffExpire).Should(Equal(time.Time{}))
		Expect(re[0].DrainBackOffCount).Should(BeZero())

		By("Waiting for reboot completion")
		waitRebootCompletion(cluster)

		By("Cleaning up the deployment")
		_, stderr, err = kubectlWithInput(rebootSlowEvictionDeploymentYAML, "delete", "-f", "-")
		Expect(err).ShouldNot(HaveOccurred(), "stderr: %s", stderr)

		Eventually(func() error {
			out, _, err = kubectl("get", "-n=reboot-test", "pod", "-l=reboot-app=slow", "-o=json")
			if err != nil {
				return err
			}
			err = json.Unmarshal(out, &deploymentPods)
			if err != nil {
				return err
			}
			if len(deploymentPods.Items) != 0 {
				return fmt.Errorf("Pod does not terminate")
			}
			return nil
		}).Should(Succeed())
	})

	It("checks parallel reboot behavior", func() {
		// Note: this test is incomplete if rq entries are processed in random order
		By("Modifying cluster configuration for this test")
		cluster.Reboot.MaxConcurrentReboots = ptr.To(2)
		originalBootCheckCommand := cluster.Reboot.BootCheckCommand
		cluster.Reboot.BootCheckCommand = []string{"bash", "-c", "if [ $0 = 10.0.0.104 ]; then echo false; else echo true; fi"}
		cluster.Reboot.ProtectedNamespaces = &metav1.LabelSelector{
			// avoid eviction failure due to cluster-dns in kube-system NS
			MatchExpressions: []metav1.LabelSelectorRequirement{
				{
					Key:      "kubernetes.io/metadata.name",
					Operator: metav1.LabelSelectorOpNotIn,
					Values:   []string{"kube-system"},
				},
			},
		}
		_, err := ckecliClusterSet(cluster)
		Expect(err).ShouldNot(HaveOccurred())
		// wait for the previous reconciliation to be done
		time.Sleep(time.Second * 3)

		By("Starting to reboot worker nodes")
		rebootQueueAdd([]string{node4, node5, node6})

		By("Waiting for reboot completion of the nodes whose reboot is not stuck")
		limit := time.Now().Add(time.Second * time.Duration(30))
		for {
			time.Sleep(time.Second)

			t := time.Now()
			Expect(t.After(limit)).ShouldNot(BeTrue(), "reboot queue processing timed out")

			re, err := getRebootEntries()
			Expect(err).ShouldNot(HaveOccurred())
			if len(re) > 1 {
				continue
			}
			Expect(re).ShouldNot(HaveLen(0), "reboot should not complete yet")
			Expect(re[0].Node).Should(Equal(node4), "unexpected node remains")

			break
		}

		By("Making reboot not stuck")
		cluster.Reboot.BootCheckCommand = originalBootCheckCommand
		_, err = ckecliClusterSet(cluster)
		Expect(err).ShouldNot(HaveOccurred())

		By("Waiting for reboot completion")
		waitRebootCompletion(cluster)

		By("Restoring cluster configuration")
		cluster.Reboot.MaxConcurrentReboots = nil
		cluster.Reboot.ProtectedNamespaces = nil
		_, err = ckecliClusterSet(cluster)
		Expect(err).ShouldNot(HaveOccurred())
		// wait for the previous reconciliation to be done
		time.Sleep(time.Second * 3)
	})

	It("checks API server reboot behavior", func() {
		// Note: this test is incomplete if rq entries are processed in random order
		By("Modifying cluster configuration for this test")
		cluster.Reboot.MaxConcurrentReboots = ptr.To(2)
		originalRebootCommand := cluster.Reboot.RebootCommand
		cluster.Reboot.ProtectedNamespaces = &metav1.LabelSelector{
			// avoid eviction failure due to cluster-dns in kube-system NS
			MatchExpressions: []metav1.LabelSelectorRequirement{
				{
					Key:      "kubernetes.io/metadata.name",
					Operator: metav1.LabelSelectorOpNotIn,
					Values:   []string{"kube-system"},
				},
			},
		}
		apiServerRebootSeconds := 10
		cluster.Reboot.RebootCommand = []string{"bash", "-c", "sleep " + fmt.Sprintf("%d", apiServerRebootSeconds)}
		_, err := ckecliClusterSet(cluster)
		Expect(err).ShouldNot(HaveOccurred())
		// wait for the previous reconciliation to be done
		time.Sleep(time.Second * 3)

		By("Enumerating nodes")
		stdout, stderr, err := kubectl("get", "node", "-o=json")
		Expect(err).ShouldNot(HaveOccurred(), "stderr: %s", stderr)
		var nodeList corev1.NodeList
		err = json.Unmarshal(stdout, &nodeList)
		Expect(err).ShouldNot(HaveOccurred())
		apiServerSlice := []string{}
		apiServerSet := map[string]bool{}
		workerNodeSlice := []string{}
		for _, node := range nodeList.Items {
			if node.Labels["cke.cybozu.com/master"] == "true" {
				apiServerSlice = append(apiServerSlice, node.Name)
				apiServerSet[node.Name] = true
			} else {
				workerNodeSlice = append(workerNodeSlice, node.Name)
			}
		}

		By("Starting to reboot nodes")
		// API servers first, then worker nodes.
		ckecliSafe("reboot-queue", "disable")
		rebootQueueAdd(apiServerSlice)
		rebootQueueAdd(workerNodeSlice)
		ckecliSafe("reboot-queue", "enable")

		// First, worker nodes are processed even though they are added to reboot queue later.
		// And then, API servers are processed one by one.

		classifyEntries := func() (int, int, int) {
			re, err := getRebootEntries()
			Expect(err).ShouldNot(HaveOccurred())
			apiServerRemain := 0
			workerNodeRemain := 0
			apiServerInProgress := 0
			for _, entry := range re {
				if apiServerSet[entry.Node] {
					apiServerRemain++
				} else {
					workerNodeRemain++
				}
				switch entry.Status {
				case cke.RebootStatusDraining, cke.RebootStatusRebooting:
					if apiServerSet[entry.Node] {
						apiServerInProgress++
					}
				}
			}
			return apiServerRemain, workerNodeRemain, apiServerInProgress
		}

		By("Waiting for reboot completion of worker nodes")
		// enough longer than apiServerRebootSeconds * roundup(number of worker nodes / MaxConcurrentReboots)
		limit := time.Now().Add(time.Second * time.Duration(apiServerRebootSeconds*2+40))
		for {
			t := time.Now()
			Expect(t.After(limit)).ShouldNot(BeTrue(), "reboot queue processing timed out")

			_, workerNodeRemain, apiServerInProgress := classifyEntries()

			Expect(apiServerInProgress > 0 && workerNodeRemain > 0).Should(BeFalse(), "API servers should be processed with lower priority")

			if workerNodeRemain == 0 {
				break
			}

			time.Sleep(time.Second)
		}

		By("Waiting for reboot completion of API servers")
		// enough longer than apiServerRebootSeconds * 3
		limit = time.Now().Add(time.Second * time.Duration(apiServerRebootSeconds*3+60))
		for {
			t := time.Now()
			Expect(t.After(limit)).ShouldNot(BeTrue(), "reboot queue processing timed out")

			apiServerRemain, _, apiServerInProgress := classifyEntries()

			Expect(apiServerInProgress).Should(BeNumerically("<=", 1), "API servers should be processed one by one")

			// Both default/kubernetes and kube-system/cke-etcd Endpoints should have no less than two endpoints during API server reboots.
			// (Check for EndpointsSlices is omitted since their members are same as Endpoints)
			epNames := []struct {
				name      string
				namespace string
			}{
				{
					name:      "kubernetes",
					namespace: metav1.NamespaceDefault,
				},
				{
					name:      op.EtcdEndpointsName,
					namespace: metav1.NamespaceSystem,
				},
			}
			for _, epName := range epNames {
				out, _, err := kubectl("get", "endpoints", "-n", epName.namespace, epName.name, "-o=json")
				Expect(err).ShouldNot(HaveOccurred())

				var ep corev1.Endpoints
				err = json.Unmarshal(out, &ep)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(ep.Subsets).Should(HaveLen(1))
				Expect(len(ep.Subsets[0].Addresses)).Should(BeNumerically(">=", 2))
			}

			if apiServerRemain == 0 {
				break
			}

			time.Sleep(time.Second)
		}

		// reboot is completed

		By("Restoring cluster configuration")
		cluster.Reboot.RebootCommand = originalRebootCommand
		cluster.Reboot.MaxConcurrentReboots = nil
		cluster.Reboot.ProtectedNamespaces = nil
		_, err = ckecliClusterSet(cluster)
		Expect(err).ShouldNot(HaveOccurred())
		// wait for the previous reconciliation to be done
		time.Sleep(time.Second * 3)
	})

	It("checks relationship with unreachable nodes", func() {
		// this test stops node4

		By("Shutting down a worker node")
		// poweroff after five seconds using systemd-run
		// `ssh node4 sudo poweroff` is unstable because it will returns error if shutdown process proceeds before ssh command completes.
		execSafeAt(node4, "sudo", "systemd-run", "--on-active=5s", "--timer-property=AccuracySec=1s", "poweroff")
		Eventually(func() error {
			stdout, stderr, err := kubectl("get", "nodes/"+node4, "-o", "json")
			if err != nil {
				return fmt.Errorf("stdout:%s, stderr:%s", stdout, stderr)
			}
			var node corev1.Node
			err = json.Unmarshal(stdout, &node)
			if err != nil {
				return err
			}
			for _, cond := range node.Status.Conditions {
				if cond.Type == "Ready" {
					if cond.Status == "True" {
						return fmt.Errorf("node is still Ready")
					} else {
						return nil
					}
				}
			}
			return fmt.Errorf("node ready status is unknown")
		}).Should(Succeed())

		By("Setting constraints to deny starting process")
		ckecliSafe("constraints", "set", "maximum-unreachable-nodes-for-reboot", "0")

		By("Deploying a little slow eviction pod")
		_, stderr, err := kubectlWithInput(rebootALittleSlowEvictionDeploymentYAML, "apply", "-f", "-")
		Expect(err).ShouldNot(HaveOccurred(), "stderr: %s", stderr)

		Eventually(func() error {
			out, _, err := kubectl("get", "-n=reboot-test", "deployments/alittleslow", "-o=json")
			if err != nil {
				return err
			}

			var deploy appsv1.Deployment
			err = json.Unmarshal(out, &deploy)
			if err != nil {
				return err
			}
			if deploy.Status.ReadyReplicas != 1 {
				return fmt.Errorf("deployment is not ready")
			}
			return nil
		}).Should(Succeed())

		out, _, err := kubectl("get", "-n=reboot-test", "pod", "-l=reboot-app=alittleslow", "-o=json")
		Expect(err).ShouldNot(HaveOccurred(), "stderr: %s", stderr)
		var deploymentPods corev1.PodList
		err = json.Unmarshal(out, &deploymentPods)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(deploymentPods.Items).Should(HaveLen(1))

		By("Starting to reboot the node running the pod")
		nodeName := deploymentPods.Items[0].Spec.NodeName
		rebootQueueAdd([]string{nodeName})

		By("Checking the reboot entry does not become `draining` status")
		Consistently(func() error {
			re, err := getRebootEntries()
			if err != nil {
				return err
			}
			if len(re) != 1 {
				return fmt.Errorf("reboot queue should contain exactly 1 entry")
			}
			if re[0].Status != cke.RebootStatusQueued {
				return fmt.Errorf("reboot entry should have queued status")
			}
			return nil
		}, time.Second*30).Should(Succeed())

		By("Setting constraints to allow starting process")
		ckecliSafe("constraints", "set", "maximum-unreachable-nodes-for-reboot", "1")

		By("Checking the reboot entry becomes `draining` status")
		Eventually(func() error {
			re, err := getRebootEntries()
			if err != nil {
				return err
			}
			if len(re) != 1 {
				return fmt.Errorf("reboot queue should contain exactly 1 entry")
			}
			if re[0].Status != cke.RebootStatusDraining {
				return fmt.Errorf("reboot entry should have draining status")
			}
			return nil
		}).Should(Succeed())

		By("Setting constraints to deny starting process again")
		ckecliSafe("constraints", "set", "maximum-unreachable-nodes-for-reboot", "0")

		By("Waiting for reboot completion")
		waitRebootCompletion(cluster)

		By("Cleaning up the deployment")
		_, stderr, err = kubectlWithInput(rebootALittleSlowEvictionDeploymentYAML, "delete", "-f", "-")
		Expect(err).ShouldNot(HaveOccurred(), "stderr: %s", stderr)

		Eventually(func() error {
			out, _, err := kubectl("get", "-n=reboot-test", "pod", "-l=reboot-app=worker", "-o=json")
			if err != nil {
				return err
			}
			var deploymentPods corev1.PodList
			err = json.Unmarshal(out, &deploymentPods)
			if err != nil {
				return err
			}
			if len(deploymentPods.Items) != 0 {
				return fmt.Errorf("Pod does not terminate")
			}
			return nil
		}).Should(Succeed())
	})
}
