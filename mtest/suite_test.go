package mtest

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/log"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const (
	dummyCNIConf = "/etc/cni/net.d/00-dummy.conf"
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
	SetDefaultEventuallyTimeout(10 * time.Minute)

	log.DefaultLogger().SetThreshold(log.LvError)

	err := prepareSSHClients(host1, host2, node1, node2, node3, node4, node5, node6)
	Expect(err).NotTo(HaveOccurred())

	// sync VM root filesystem to store newly generated SSH host keys.
	for h := range sshClients {
		execSafeAt(h, "sync")
	}

	_, stderr, err := execAt(node1, "sudo", "mkdir", "-p", filepath.Dir(dummyCNIConf))
	if err != nil {
		Fail("failed to mkdir dummyCNIConf " + string(stderr))
	}
	_, stderr, err = execAt(node1, "sudo", "touch", dummyCNIConf)
	if err != nil {
		Fail("failed to touch dummyCNIConf " + string(stderr))
	}

	for _, h := range []string{host1, host2} {
		_, stderr, err := execAt(h, "/data/setup-cke.sh")
		if err != nil {
			Fail("failed to complete setup-cke.sh: " + string(stderr))
		}
	}

	_, stderr, err = execAt(node1, "sudo", "/data/setup-local-pv.sh")
	if err != nil {
		Fail("failed to complete setup-local-pv.sh: " + string(stderr))
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

	kubeconfig := ckecli("kubernetes", "issue")
	f, err := os.Create("/tmp/cke-mtest-kubeconfig")
	Expect(err).NotTo(HaveOccurred())
	defer f.Close()
	f.Write(kubeconfig)
	f.Sync()

	fmt.Println("Begin tests...")
})
