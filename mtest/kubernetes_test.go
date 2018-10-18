package mtest

import (
	"encoding/json"
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
)

var _ = Describe("Kubernetes", func() {
	It("can run Pods", func() {
		By("waiting the default service account gets created")
		Eventually(func() error {
			_, err := kubectl("get", "sa/default", "-o", "json")
			return err
		}).Should(Succeed())

		By("running nginx")
		_, err := kubectl("run", "nginx", "--image=nginx",
			`--overrides={"spec": {"hostNetwork": true}}`,
			"--generator=run-pod/v1")
		Expect(err).NotTo(HaveOccurred())

		By("checking nginx pod status")
		Eventually(func() error {
			stdout, err := kubectl("get", "pods/nginx", "-o", "json")
			if err != nil {
				return err
			}

			var pod corev1.Pod
			err = json.Unmarshal(stdout, &pod)
			if err != nil {
				return err
			}

			if !pod.Spec.HostNetwork {
				return errors.New("pod is not running in host network")
			}

			for _, cond := range pod.Status.Conditions {
				if cond.Type == corev1.PodReady && cond.Status == corev1.ConditionTrue {
					return nil
				}
			}
			return errors.New("pod is not yet ready")
		}).Should(Succeed())
	})
})
