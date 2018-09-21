package mtest

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/cybozu-go/cke"
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

	SetDefaultEventuallyPollingInterval(3 * time.Second)
	SetDefaultEventuallyTimeout(6 * time.Minute)

	err := prepareSSHClients(host1, host2, node1, node2, node3, node4, node5, node6)
	Expect(err).NotTo(HaveOccurred())

	// sync VM root filesystem to store newly generated SSH host keys.
	for h := range sshClients {
		execSafeAt(h, "sync")
	}

	// wait cke
	Eventually(func() error {
		_, _, err := execAt(host1, "test", "-f", "/usr/bin/jq")
		if err != nil {
			return err
		}
		_, _, err = execAt(host2, "test", "-f", "/usr/bin/jq")
		return err
	}).Should(Succeed())

	err = stopManagementEtcd(sshClients[host1])
	Expect(err).NotTo(HaveOccurred())
	err = stopVault(sshClients[host1])
	Expect(err).NotTo(HaveOccurred())

	for _, h := range []string{host1, host2} {
		execSafeAt(h, "/data/setup-cke.sh")
	}

	etcd, err := connectEtcd()
	Expect(err).NotTo(HaveOccurred())
	defer etcd.Close()

	resp, err := etcd.Get(context.Background(), "vault")
	Expect(err).NotTo(HaveOccurred())
	Expect(int(resp.Count)).NotTo(BeZero())
	err = cke.ConnectVault(context.Background(), resp.Kvs[0].Value)
	Expect(err).NotTo(HaveOccurred())

	setupCKE()

	initializeControlPlane()

	fmt.Println("Begin tests...")
})
