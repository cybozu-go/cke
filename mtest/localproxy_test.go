package mtest

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/cybozu-go/cke"
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

	It("should use digest-pinned images", func() {
		out := execSafeAt(host1, "docker", "inspect", "kube-proxy", "cke-unbound")
		var inspects []struct {
			Config struct {
				Image  string            `json:"Image"`
				Labels map[string]string `json:"Labels"`
			} `json:"Config"`
			Name string `json:"Name"`
		}
		err := json.Unmarshal(out, &inspects)
		Expect(err).NotTo(HaveOccurred())
		Expect(inspects).To(HaveLen(2))

		expected := map[string]cke.Image{
			"/kube-proxy":  cke.KubernetesImage,
			"/cke-unbound": cke.UnboundImage,
		}
		for _, inspect := range inspects {
			fmt.Fprintf(GinkgoWriter, "container=%s image=%s\n", inspect.Name, inspect.Config.Image)
			img, ok := expected[inspect.Name]
			Expect(ok).To(BeTrue(), "unexpected container %s", inspect.Name)

			// docker run is given the tag reference; the pinned reference is recorded in the CKE label.
			Expect(inspect.Config.Image).To(Equal(img.TagRef()), "container %s", inspect.Name)
			var label struct {
				Image string `json:"image"`
			}
			Expect(json.Unmarshal([]byte(inspect.Config.Labels[cke.CKELabelName]), &label)).To(Succeed(), "container %s", inspect.Name)
			Expect(label.Image).To(Equal(img.FullRef()), "container %s", inspect.Name)
		}
	})
}
