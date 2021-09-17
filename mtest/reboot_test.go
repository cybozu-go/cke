package mtest

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/cke/op"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

func rebootEntriesShouldBeRemaining() {
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

func testRebootOperations() {
	AfterEach(initializeControlPlane)

	It("run reboot operations", func() {
		cluster := getCluster()
		for i := 0; i < 3; i++ {
			cluster.Nodes[i].ControlPlane = true
		}

		By("Rebooting nodes")
		// this will run:
		// - RebootOp
		// - RebootDequeueOp
		// - RebootUncordonOp
		rebootTargets := node1
		_, _, err := ckecliWithInput([]byte(rebootTargets), "reboot-queue", "add", "-")
		Expect(err).ShouldNot(HaveOccurred())
		rebootTargets = node2 + "\n" + node4
		_, _, err = ckecliWithInput([]byte(rebootTargets), "reboot-queue", "add", "-")
		Expect(err).ShouldNot(HaveOccurred())
		waitRebootCompletion(cluster)
		nodesShouldBeSchedulable(node1, node2, node4)
	})

	It("should give up waiting node startup if deadline is exceeded", func() {
		cluster := getCluster()
		for i := 0; i < 3; i++ {
			cluster.Nodes[i].ControlPlane = true
		}
		cluster.Reboot.Command = []string{"sleep", "3600"}
		clusterSetAndWait(cluster)

		rebootTargets := node1
		_, _, err := ckecliWithInput([]byte(rebootTargets), "reboot-queue", "add", "-")
		Expect(err).ShouldNot(HaveOccurred())
		ts := time.Now()
		waitRebootCompletion(cluster)
		nodesShouldBeSchedulable(node1)

		timeout := time.Second * time.Duration(*cluster.Reboot.CommandTimeoutSeconds)
		Expect(time.Now()).To(BeTemporally(">", ts.Add(timeout)))
	})

	It("should be controlled by 'ckecli reboot-queue' command", func() {
		cluster := getCluster()
		for i := 0; i < 3; i++ {
			cluster.Nodes[i].ControlPlane = true
		}

		By("ckecli reboot-queue disable disables reboot queue processing")
		ckecliSafe("reboot-queue", "disable")
		// wait for the previous reconciliation to be done
		time.Sleep(time.Second * 3)

		_, _, err := ckecliWithInput([]byte(node1), "reboot-queue", "add", "-")
		Expect(err).ShouldNot(HaveOccurred())
		rebootEntriesShouldBeRemaining()

		By("ckecli reboot-queue enable enables reboot queue processing")
		ckecliSafe("reboot-queue", "enable")
		waitRebootCompletion(cluster)
		nodesShouldBeSchedulable(node1)

		By("ckecli reboot-queue cancel cancels the specified reboot queue entry")
		ckecliSafe("reboot-queue", "disable")
		_, _, err = ckecliWithInput([]byte(node1), "reboot-queue", "add", "-")
		Expect(err).ShouldNot(HaveOccurred())
		ckecliSafe("reboot-queue", "cancel", "4")
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
	})

	It("should respect PodDisruptionBudget in protected_namespaces", func() {
		cluster := getCluster()
		for i := 0; i < 3; i++ {
			cluster.Nodes[i].ControlPlane = true
		}

		By("Preparing a deployment to test protected_namespaces")
		_, stderr, err := kubectlWithInput(rebootYAML, "apply", "-f", "-")
		Expect(err).ShouldNot(HaveOccurred(), "stderr: %s", stderr)

		Eventually(func() error {
			out, _, err := kubectl("get", "-n=reboot-sample", "deployments/sample", "-o=json")
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

		out, _, err := kubectl("get", "-n=reboot-sample", "pod", "-l=reboot-app=sample", "-o=json")
		Expect(err).ShouldNot(HaveOccurred(), "stderr: %s", stderr)
		var pods corev1.PodList
		err = json.Unmarshal(out, &pods)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(pods.Items).Should(HaveLen(3))

		By("Reboot operation will protect all pods if protected_namespaces is nil")
		nodeName := pods.Items[0].Spec.NodeName
		_, _, err = ckecliWithInput([]byte(nodeName), "reboot-queue", "add", "-")
		Expect(err).ShouldNot(HaveOccurred())
		rebootEntriesShouldBeRemaining()

		ckecliSafe("reboot-queue", "cancel-all")
		waitRebootCompletion(cluster)
		nodesShouldBeSchedulable(nodeName)

		By("Reboot operation will protect pods in protected_namespaces")
		cluster.Reboot.ProtectedNamespaces = &metav1.LabelSelector{
			MatchLabels: map[string]string{"reboot-test": "sample"},
		}
		_, err = ckecliClusterSet(cluster)
		Expect(err).ShouldNot(HaveOccurred())

		nodeName = pods.Items[1].Spec.NodeName
		_, _, err = ckecliWithInput([]byte(nodeName), "reboot-queue", "add", "-")
		Expect(err).ShouldNot(HaveOccurred())
		rebootEntriesShouldBeRemaining()

		ckecliSafe("reboot-queue", "cancel-all")
		waitRebootCompletion(cluster)
		nodesShouldBeSchedulable(nodeName)

		By("Reboot operation deletes non-protected pods")
		cluster.Reboot.ProtectedNamespaces = &metav1.LabelSelector{
			MatchLabels: map[string]string{"reboot-test": "test"},
		}
		_, err = ckecliClusterSet(cluster)
		Expect(err).ShouldNot(HaveOccurred())

		nodeName = pods.Items[2].Spec.NodeName
		_, _, err = ckecliWithInput([]byte(nodeName), "reboot-queue", "add", "-")
		Expect(err).ShouldNot(HaveOccurred())
		waitRebootCompletion(cluster)
		nodesShouldBeSchedulable(nodeName)
	})
}
