package mtest

import (
	"fmt"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestMtest(t *testing.T) {
	if len(sshKeyFile) == 0 {
		t.Skip("no SSH_PRIVKEY envvar")
	}
	RegisterFailHandler(Fail)
	RunSpecs(t, "Multi-host test for cke")
}

var _ = BeforeSuite(func() {
	fmt.Println("Preparing...")

	SetDefaultEventuallyPollingInterval(5 * time.Second)
	SetDefaultEventuallyTimeout(3 * time.Minute)

	err := prepareSSHClients(host1, host2, node1, node2, node3, node4, node5, node6)
	Expect(err).NotTo(HaveOccurred())

	// sync VM root filesystem to store newly generated SSH host keys.
	for h := range sshClients {
		execSafeAt(h, "sync")
	}

	err = stopManagementEtcd(sshClients[host1])
	Expect(err).NotTo(HaveOccurred())
	err = runManagementEtcd(sshClients[host1])
	Expect(err).NotTo(HaveOccurred())

	time.Sleep(time.Second)

	err = stopCke()
	Expect(err).NotTo(HaveOccurred())
	err = runCke()
	Expect(err).NotTo(HaveOccurred())

	// wait cke
	Eventually(func() error {
		_, _, err := execAt(host1, "/data/ckecli", "history")
		return err
	}).Should(Succeed())

	ckecli("constraints", "set", "control-plane-count", "3")
	cluster := getCluster()
	for i := 0; i < 3; i++ {
		cluster.Nodes[i].ControlPlane = true
	}
	ckecliClusterSet(cluster)
	Eventually(func() bool {
		controlPlanes := []string{node1, node2, node3}
		workers := []string{node4, node5, node6}
		status, err := getClusterStatus()
		if err != nil {
			return false
		}
		return checkEtcdClusterStatus(status, controlPlanes, workers)
	}).Should(BeTrue())

	time.Sleep(time.Second)
	fmt.Println("Begin tests...")
})
