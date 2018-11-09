package mtest

import (
	"encoding/json"
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
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

	It("has cluster dns resources", func() {
		for resource, name := range map[string]string{
			"serviceaccounts":     "cluster-dns",
			"clusterroles":        "system:cluster-dns",
			"clusterrolebindings": "system:cluster-dns",
			"configmaps":          "cluster-dns",
			"deployments":         "cluster-dns",
			"services":            "cluster-dns",
		} {
			_, err := kubectl("-n", "kube-system", "get", resource+"/"+name)
			Expect(err).ShouldNot(HaveOccurred())
		}

		stdout, err := kubectl("-n", "kube-system", "get", "configmaps/cluster-dns", "-o=json")
		Expect(err).ShouldNot(HaveOccurred())
		configMap := new(corev1.ConfigMap)
		err = json.Unmarshal(stdout, configMap)
		Expect(err).ShouldNot(HaveOccurred())
		domain, ok := configMap.ObjectMeta.Labels["cke-domain"]
		Expect(ok).Should(BeTrue())
		Expect(domain).Should(Equal("neco"))
		dnsServers, ok := configMap.ObjectMeta.Labels["cke-dns-servers"]
		Expect(ok).Should(BeTrue())
		Expect(dnsServers).Should(Equal("8.8.8.8_1.1.1.1"))
	})

	It("has node dns resources", func() {
		for resource, name := range map[string]string{
			"configmaps": "node-dns",
			"daemonsets": "node-dns",
		} {
			_, err := kubectl("-n", "kube-system", "get", resource+"/"+name)
			Expect(err).ShouldNot(HaveOccurred())
		}

		By("checking node-dns pod status")
		Eventually(func() error {
			stdout, err := kubectl("-n", "kube-system", "get", "daemonsets/node-dns", "-o", "json")
			if err != nil {
				return err
			}

			var daemonSet appsv1.DaemonSet
			err = json.Unmarshal(stdout, &daemonSet)
			if err != nil {
				return err
			}

			if daemonSet.Status.NumberReady != 5 {
				return errors.New("NumberReady is not 5")
			}

			return nil
		}).Should(Succeed())
	})

	It("has kube-system/cke-etcd Service and Endpoints", func() {
		_, err := kubectl("-n", "kube-system", "get", "services/cke-etcd")
		Expect(err).ShouldNot(HaveOccurred())
		_, err = kubectl("-n", "kube-system", "get", "endpoints/cke-etcd")
		Expect(err).ShouldNot(HaveOccurred())
	})
})
