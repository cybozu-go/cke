package mtest

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cybozu-go/cke"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func testTrustedRESTMapping() {
	It("applies custom resources using trusted REST mappings even when CRD is applied after CR", func() {
		By("setting trusted REST mappings in the cluster config")
		cluster := getCluster(0, 1, 2)
		cluster.TrustedRESTMappings = []cke.TrustedRESTMapping{
			{
				Group:      "mtest.cybozu.com",
				Version:    "v1",
				Kind:       "TestResource",
				Resource:   "testresources",
				Namespaced: true,
			},
		}
		clusterSetAndWait(cluster)

		By("registering a custom resource as a user-defined resource (CRD does not exist yet)")
		_, stderr, err := ckecliWithInput(trustedRESTMappingCRYAML, "resource", "set", "-")
		Expect(err).NotTo(HaveOccurred(), "stderr: %s", stderr)
		defer func() {
			ckecliWithInput(trustedRESTMappingCRYAML, "resource", "delete", "-")
		}()

		By("creating the CRD via kubectl")
		_, stderr, err = kubectlWithInput(trustedRESTMappingCRDYAML, "apply", "-f", "-")
		Expect(err).NotTo(HaveOccurred(), "stderr: %s", stderr)

		By("waiting for the CRD to become established")
		Eventually(func() error {
			stdout, stderr, err := kubectl("get", "crd", "testresources.mtest.cybozu.com", "-o", `jsonpath='{.status.conditions[?(@.type=="Established")].status}'`)
			if err != nil {
				return fmt.Errorf("%v: stderr=%s", err, stderr)
			}
			if !strings.Contains(string(stdout), "True") {
				return fmt.Errorf("CRD not yet established: %s", stdout)
			}
			return nil
		}).Should(Succeed())

		By("verifying the custom resource is eventually created after CRD becomes available")
		Eventually(func() error {
			stdout, stderr, err := kubectl("get", "testresources.mtest.cybozu.com", "test-cr", "-n", "default", "-o", "json")
			if err != nil {
				return fmt.Errorf("%v: stderr=%s", err, stderr)
			}
			var obj unstructured.Unstructured
			if err := json.Unmarshal(stdout, &obj); err != nil {
				return err
			}
			spec, ok := obj.Object["spec"].(map[string]interface{})
			if !ok {
				return fmt.Errorf("spec not found")
			}
			msg, ok := spec["message"].(string)
			if !ok || msg != "hello" {
				return fmt.Errorf("unexpected spec.message: %v", spec["message"])
			}
			return nil
		}).Should(Succeed())

		By("verifying the CKE revision annotation is set")
		stdout, stderr, err := kubectl("get", "testresources.mtest.cybozu.com", "test-cr", "-n", "default", "-o", "json")
		Expect(err).NotTo(HaveOccurred(), "stderr: %s", stderr)
		var obj unstructured.Unstructured
		err = json.Unmarshal(stdout, &obj)
		Expect(err).NotTo(HaveOccurred())
		ann := obj.GetAnnotations()
		Expect(ann).To(HaveKey("cke.cybozu.com/revision"))

		By("cleaning up the custom resource and CRD")
		ckecliWithInput(trustedRESTMappingCRYAML, "resource", "delete", "-")
		waitServerStatusCompletion()

		_, stderr, err = kubectl("delete", "crd", "testresources.mtest.cybozu.com")
		Expect(err).NotTo(HaveOccurred(), "stderr: %s", stderr)

		By("removing trusted REST mappings from the cluster config")
		cluster = getCluster(0, 1, 2)
		cluster.TrustedRESTMappings = nil
		clusterSetAndWait(cluster)
	})
}
