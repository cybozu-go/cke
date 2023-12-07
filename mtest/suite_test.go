package mtest

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/log"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	dummyCNIConf = "/etc/cni/net.d/00-dummy.conf"
)

func TestMtest(t *testing.T) {
	if os.Getenv("SSH_PRIVKEY") == "" {
		t.Skip("no SSH_PRIVKEY envvar")
	}
	RegisterFailHandler(Fail)
	RunSpecs(t, "Multi-host test for cke")
}

var _ = BeforeSuite(func() {
	img := ckeImageURL
	if testSuite == "upgrade" {
		img = "ghcr.io/cybozu/cke:" + cke.Version
	}

	fmt.Println("Preparing...")

	SetDefaultEventuallyPollingInterval(3 * time.Second)
	SetDefaultEventuallyTimeout(9 * time.Minute)

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

	By("stopping previous cke.service")
	for _, host := range []string{host1, host2} {
		execAt(host, "sudo", "systemctl", "reset-failed", "cke.service")
		execAt(host, "sudo", "systemctl", "stop", "cke.service")
	}

	By("copying test files")
	for _, testFile := range []string{kubectlPath} {
		func() {
			f, err := os.Open(testFile)
			Expect(err).NotTo(HaveOccurred())
			defer f.Close()
			remoteFilename := filepath.Join("/tmp", filepath.Base(testFile))
			for _, host := range []string{host1, host2} {
				_, err := f.Seek(0, io.SeekStart)
				Expect(err).NotTo(HaveOccurred())
				stdout, stderr, err := execAtWithStream(host, f, "dd", "of="+remoteFilename)
				Expect(err).NotTo(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)
				stdout, stderr, err = execAt(host, "sudo", "mv", remoteFilename, filepath.Join("/opt/bin", filepath.Base(testFile)))
				Expect(err).NotTo(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)
				stdout, stderr, err = execAt(host, "sudo", "chmod", "755", filepath.Join("/opt/bin", filepath.Base(testFile)))
				Expect(err).NotTo(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)
			}
		}()
	}

	By("loading test image")
	err = loadImage(ckeImagePath)
	Expect(err).NotTo(HaveOccurred())

	By("running install-tools")
	err = installTools(img)
	Expect(err).NotTo(HaveOccurred())

	f, err := os.Open(ckeConfigPath)
	Expect(err).NotTo(HaveOccurred())
	defer f.Close()
	remoteFilename := filepath.Join("/etc/cke", filepath.Base(ckeConfigPath))
	for _, host := range []string{host1, host2} {
		execSafeAt(host, "mkdir", "-p", "/etc/cke")
		_, err := f.Seek(0, io.SeekStart)
		Expect(err).NotTo(HaveOccurred())
		stdout, stderr, err := execAtWithStream(host, f, "sudo", "dd", "of="+remoteFilename)
		Expect(err).NotTo(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)
	}

	By("setup cke")
	for _, h := range []string{host1, host2} {
		_, stderr, err := execAt(h, "/data/setup-cke.sh")
		if err != nil {
			Fail("failed to complete setup-cke.sh: " + string(stderr))
		}
	}

	etcd, err := connectEtcd()
	Expect(err).NotTo(HaveOccurred())
	defer etcd.Close()

	resp, err := etcd.Get(context.Background(), "vault")
	Expect(err).NotTo(HaveOccurred())
	Expect(len(resp.Kvs)).NotTo(BeZero())
	err = cke.ConnectVault(context.Background(), resp.Kvs[0].Value)
	Expect(err).NotTo(HaveOccurred())

	setupCKE(img)

	By("initializing control plane")
	initializeControlPlane()
	execSafeAt(host1, "mkdir", "-p", ".kube")

	ckecliSafe("kubernetes", "issue", ">", ".kube/config")

	fmt.Println("Begin tests...")
})

// This must be the only top-level test container.
// Other tests and test containers must be listed in this.
var _ = Describe("Test CKE", func() {
	switch testSuite {
	case "functions":
		Context("ckecli", testCKECLI)
		Context("local-proxy", testLocalProxy)
		Context("kubernetes", testKubernetes)
	case "operators":
		Context("operators", func() {
			testOperators(false)
		})
	case "robustness":
		Context("operators", Ordered, func() {
			testStopCP()
			testOperators(true)
		})
	case "reboot":
		Context("reboot", func() {
			testRebootOperations()
		})
	case "upgrade":
		Context("upgrade", Ordered, func() {
			testUpgrade()
			Context("kubernetes", testKubernetes)
		})
	}
})
