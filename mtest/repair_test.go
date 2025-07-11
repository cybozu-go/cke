package mtest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/cybozu-go/cke"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func getRepairEntries() ([]*cke.RepairQueueEntry, error) {
	var entries []*cke.RepairQueueEntry
	data, stderr, err := ckecli("repair-queue", "list")
	if err != nil {
		return nil, fmt.Errorf("%w, stdout: %s, stderr: %s", err, data, stderr)
	}
	err = json.Unmarshal(data, &entries)
	if err != nil {
		return nil, err
	}
	return entries, nil
}

func waitRepairCompletion(cluster *cke.Cluster, statuses []cke.RepairStatus) {
	ts := time.Now()
	EventuallyWithOffset(2, func(g Gomega) {
		entries, err := getRepairEntries()
		g.Expect(err).NotTo(HaveOccurred())
		for _, entry := range entries {
			g.Expect(entry.Status).To(BeElementOf(statuses))
		}
		g.Expect(checkCluster(cluster, ts)).NotTo(HaveOccurred())
	}).Should(Succeed())
}

func waitRepairSuccess(cluster *cke.Cluster) {
	waitRepairCompletion(cluster, []cke.RepairStatus{cke.RepairStatusSucceeded})
}

func waitRepairFailure(cluster *cke.Cluster) {
	waitRepairCompletion(cluster, []cke.RepairStatus{cke.RepairStatusFailed})
}

func waitRepairEmpty(cluster *cke.Cluster) {
	waitRepairCompletion(cluster, nil)
}

func repairShouldNotProceed() {
	ConsistentlyWithOffset(1, func(g Gomega) {
		entries, err := getRepairEntries()
		g.Expect(err).NotTo(HaveOccurred())
		for _, entry := range entries {
			g.Expect(entry.Status).NotTo(BeElementOf(cke.RepairStatusSucceeded, cke.RepairStatusFailed))
		}
	}).WithTimeout(time.Second * 60).Should(Succeed())
}

func repairSuccessCommandSuccess(node string) {
	cmdSuccess := false
	for _, host := range []string{host1, host2} {
		_, _, err := execAt(host, "docker", "exec", "cke", "test", "-f", "/tmp/mtest-repair-success-"+node)
		if err == nil {
			cmdSuccess = true
		}
	}
	Expect(cmdSuccess).To(BeTrue())
}

func testRepairOperations() {
	// this will run:
	// - RepairDrainStartOp
	// - RepairExecuteOp
	// - RepairDrainTimeoutOp
	// - RepairFinishOp
	// - RepairDequeueOp

	// This test examines status gathering and CLI commands as well as operations.
	// It is not necessary to test the behaviors examined in "server/strategy_test.go".

	// This test uses "touch" and "test -f" for repair_command and health_check_command.
	// "true" and "echo true" are insufficient for repair queue test because
	// CKE first checks health and never calls "RepairDrainStartOp" for healthy machines.
	It("should execute repair commands", func() {
		cluster := getCluster()
		for i := 0; i < 3; i++ {
			cluster.Nodes[i].ControlPlane = true
		}

		currentWriteIndex := 0
		repairQueueAdd := func(address string) {
			execSafeAt(host1, "docker", "exec", "cke", "find", "/tmp", "-maxdepth", "1", "-name", "mtest-repair-*", "-delete")
			execSafeAt(host2, "docker", "exec", "cke", "find", "/tmp", "-maxdepth", "1", "-name", "mtest-repair-*", "-delete")
			_, stderr, err := ckecli("repair-queue", "add", "op1", "type1", address, "SN1234")
			ExpectWithOffset(1, err).NotTo(HaveOccurred(), "stderr: %s", stderr)
			currentWriteIndex++
		}

		By("disabling repair queue")
		ckecliSafe("repair-queue", "disable")
		stdout := ckecliSafe("repair-queue", "is-enabled")
		Expect(bytes.TrimSpace(stdout)).To(Equal([]byte("false")))

		repairQueueAdd(node1)
		repairShouldNotProceed()

		ckecliSafe("repair-queue", "delete-unfinished")
		waitRepairEmpty(cluster)

		By("enabling repair queue")
		ckecliSafe("repair-queue", "enable")
		stdout = ckecliSafe("repair-queue", "is-enabled")
		Expect(bytes.TrimSpace(stdout)).To(Equal([]byte("true")))

		repairQueueAdd(node1)
		waitRepairSuccess(cluster)
		nodesShouldBeSchedulable(node1)
		repairSuccessCommandSuccess(node1)

		ckecliSafe("repair-queue", "delete-finished")
		waitRepairEmpty(cluster)

		By("setting erroneous success command")
		originalSuccessCommand := cluster.Repair.RepairProcedures[0].RepairOperations[0].SuccessCommand
		cluster.Repair.RepairProcedures[0].RepairOperations[0].SuccessCommand = []string{"false"}
		_, err := ckecliClusterSet(cluster)
		Expect(err).NotTo(HaveOccurred())
		time.Sleep(time.Second * 3)

		repairQueueAdd(node1)
		waitRepairFailure(cluster)

		ckecliSafe("repair-queue", "delete-finished")
		waitRepairEmpty(cluster)

		By("restoring success command")
		cluster.Repair.RepairProcedures[0].RepairOperations[0].SuccessCommand = originalSuccessCommand
		_, err = ckecliClusterSet(cluster)
		Expect(err).NotTo(HaveOccurred())
		time.Sleep(time.Second * 3)

		By("setting erroneous repair command")
		originalRepairCommand := cluster.Repair.RepairProcedures[0].RepairOperations[0].RepairSteps[0].RepairCommand
		cluster.Repair.RepairProcedures[0].RepairOperations[0].RepairSteps[0].RepairCommand = []string{"false"}
		_, err = ckecliClusterSet(cluster)
		Expect(err).NotTo(HaveOccurred())
		time.Sleep(time.Second * 3)

		repairQueueAdd(node1)
		waitRepairFailure(cluster)

		ckecliSafe("repair-queue", "delete-finished")
		waitRepairEmpty(cluster)

		By("setting non-returning repair command")
		cluster.Repair.RepairProcedures[0].RepairOperations[0].RepairSteps[0].RepairCommand = []string{"sh", "-c", "exec sleep infinity", "sleep-infinity"}
		_, err = ckecliClusterSet(cluster)
		Expect(err).NotTo(HaveOccurred())
		time.Sleep(time.Second * 3)

		repairQueueAdd(node1)
		waitRepairFailure(cluster)

		ckecliSafe("repair-queue", "delete-finished")
		waitRepairEmpty(cluster)

		By("setting non-returning repair command and long command timeout")
		originalCommandTimeoutSeconds := cluster.Repair.RepairProcedures[0].RepairOperations[0].RepairSteps[0].CommandTimeoutSeconds

		longCommandTimeout := 90 // > (timeout of repairShouldNotProceed())
		cluster.Repair.RepairProcedures[0].RepairOperations[0].RepairSteps[0].CommandTimeoutSeconds = &longCommandTimeout
		_, err = ckecliClusterSet(cluster)
		Expect(err).NotTo(HaveOccurred())
		time.Sleep(time.Second * 3)

		repairQueueAdd(node1)
		repairShouldNotProceed()

		time.Sleep(time.Second * time.Duration(longCommandTimeout)) // wait for CKE to update the queue entry
		ckecliSafe("repair-queue", "delete-finished")
		waitRepairEmpty(cluster)

		By("setting noop repair command")
		cluster.Repair.RepairProcedures[0].RepairOperations[0].RepairSteps[0].RepairCommand = []string{"true"}
		_, err = ckecliClusterSet(cluster)
		Expect(err).NotTo(HaveOccurred())
		time.Sleep(time.Second * 3)

		repairQueueAdd(node1)
		waitRepairFailure(cluster)

		ckecliSafe("repair-queue", "delete-finished")
		waitRepairEmpty(cluster)

		By("setting noop repair command and long watch duration")
		originalWatchSeconds := cluster.Repair.RepairProcedures[0].RepairOperations[0].RepairSteps[0].WatchSeconds

		longWatch := 600
		cluster.Repair.RepairProcedures[0].RepairOperations[0].RepairSteps[0].WatchSeconds = &longWatch
		_, err = ckecliClusterSet(cluster)
		Expect(err).NotTo(HaveOccurred())
		time.Sleep(time.Second * 3)

		repairQueueAdd(node1)
		repairShouldNotProceed()

		ckecliSafe("repair-queue", "delete", strconv.Itoa(currentWriteIndex-1))
		waitRepairEmpty(cluster)

		By("restoring repair command, command timeout, and watch duration")
		cluster.Repair.RepairProcedures[0].RepairOperations[0].RepairSteps[0].RepairCommand = originalRepairCommand
		cluster.Repair.RepairProcedures[0].RepairOperations[0].RepairSteps[0].CommandTimeoutSeconds = originalCommandTimeoutSeconds
		cluster.Repair.RepairProcedures[0].RepairOperations[0].RepairSteps[0].WatchSeconds = originalWatchSeconds
		_, err = ckecliClusterSet(cluster)
		Expect(err).NotTo(HaveOccurred())
		time.Sleep(time.Second * 3)

		By("deploying drain-blocking workload")
		_, stderr, err := kubectl("create", "namespace", "repair-test")
		Expect(err).NotTo(HaveOccurred(), "stderr: %s", stderr)
		_, stderr, err = kubectl("label", "namespace", "repair-test", "protected=true")
		Expect(err).NotTo(HaveOccurred(), "stderr: %s", stderr)
		_, stderr, err = kubectlWithInput(repairDeploymentYAML, "apply", "-f", "-")
		Expect(err).NotTo(HaveOccurred(), "stderr: %s", stderr)
		nodeNames := make([]string, 3)
		Eventually(func(g Gomega) {
			stdout, stderr, err := kubectl("get", "-n=repair-test", "deployment", "sample", "-o=json")
			g.Expect(err).NotTo(HaveOccurred(), "stderr: %s", stderr)
			var deploy appsv1.Deployment
			err = json.Unmarshal(stdout, &deploy)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(deploy.Status.ReadyReplicas).To(Equal(int32(3)))

			stdout, stderr, err = kubectl("get", "-n=repair-test", "pod", "-l=app=sample", "-o=json")
			g.Expect(err).NotTo(HaveOccurred(), "stderr: %s", stderr)
			var pods corev1.PodList
			err = json.Unmarshal(stdout, &pods)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(pods.Items).To(HaveLen(3))
			for i, pod := range pods.Items {
				nodeNames[i] = pod.Spec.NodeName
				g.Expect(nodeNames[i]).NotTo(BeEmpty())
			}
		}).Should(Succeed())

		repairQueueAdd(nodeNames[0])
		repairShouldNotProceed()

		entries, err := getRepairEntries()
		Expect(err).NotTo(HaveOccurred())
		Expect(entries).To(HaveLen(1))
		Expect(entries[0].Status).To(Equal(cke.RepairStatusProcessing))
		Expect(entries[0].StepStatus).To(Equal(cke.RepairStepStatusWaiting))
		Expect(entries[0].DrainBackOffExpire).NotTo(Equal(time.Time{}))
		Expect(entries[0].DrainBackOffCount).NotTo(BeZero())

		ckecliSafe("repair-queue", "reset-backoff")
		entries, err = getRepairEntries()
		Expect(err).NotTo(HaveOccurred())
		Expect(entries).To(HaveLen(1))
		Expect(entries[0].DrainBackOffExpire).To(Equal(time.Time{}))
		Expect(entries[0].DrainBackOffCount).To(BeZero())

		ckecliSafe("repair-queue", "delete-unfinished")
		waitRepairEmpty(cluster)

		By("setting protected_namespace to include workload")
		cluster.Repair.ProtectedNamespaces = &metav1.LabelSelector{
			MatchLabels: map[string]string{"protected": "true"},
		}
		_, err = ckecliClusterSet(cluster)
		Expect(err).NotTo(HaveOccurred())
		time.Sleep(time.Second * 3)

		repairQueueAdd(nodeNames[0])
		repairShouldNotProceed()

		ckecliSafe("repair-queue", "delete-unfinished")
		waitRepairEmpty(cluster)

		By("setting protected_namespace not to include workload")
		cluster.Repair.ProtectedNamespaces = &metav1.LabelSelector{
			MatchLabels: map[string]string{"foo": "bar"},
		}
		_, err = ckecliClusterSet(cluster)
		Expect(err).NotTo(HaveOccurred())
		time.Sleep(time.Second * 3)

		repairQueueAdd(nodeNames[0])
		waitRepairSuccess(cluster)
		nodesShouldBeSchedulable(nodeNames[0])

		ckecliSafe("repair-queue", "delete-finished")
		waitRepairEmpty(cluster)

		By("restoring protected_namespace and disabling need_drain")
		cluster.Repair.ProtectedNamespaces = nil
		cluster.Repair.RepairProcedures[0].RepairOperations[0].RepairSteps[0].NeedDrain = false
		_, err = ckecliClusterSet(cluster)
		Expect(err).NotTo(HaveOccurred())
		time.Sleep(time.Second * 3)

		repairQueueAdd(nodeNames[1])
		waitRepairSuccess(cluster)
		nodesShouldBeSchedulable(nodeNames[1])
	})
}
