package mtest

import (
	"bytes"
	"errors"
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func testLocalProxy() {
	It("should run kube-localproxy on a boot server", func() {
		execSafeAt(host1, "sudo", "systemd-run", "-u", "cke-localproxy",
			"/opt/bin/cke-localproxy", "--interval=5s")
	})

	It("should run kube-proxy", func() {
		Eventually(func() error {
			stdout, stderr, err := execAt(host1, "sudo", "ipvsadm", "-L", "-n")
			if err != nil {
				return fmt.Errorf("ipvsadm failed: %s: %w", stderr, err)
			}

			if bytes.Contains(stdout, []byte("10.34.56.1:443")) {
				return nil
			}
			return errors.New("kube-proxy has not configured proxy rules yet")
		}).Should(Succeed())
	})

	It("should run unbound", func() {
		Consistently(func() error {
			stdout, stderr, err := execAt(host1, "docker", "ps", "--format={{.Names}}")
			if err != nil {
				return fmt.Errorf("docker ps failed: %s: %w", stderr, err)
			}

			if bytes.Contains(stdout, []byte("cke-unbound")) {
				return nil
			}
			return errors.New("cke-unbound service is not running")
		}, 5, 0.1).Should(Succeed())
	})
}
