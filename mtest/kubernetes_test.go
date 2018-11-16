package mtest

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/cybozu-go/cke/op"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

var _ = Describe("Kubernetes", func() {
	BeforeEach(func() {
		_, stderr, err := kubectl("create", "namespace", "mtest")
		Expect(err).NotTo(HaveOccurred(), "stderr: %s", stderr)
	})

	AfterEach(func() {
		_, stderr, err := kubectl("delete", "namespace", "mtest")
		Expect(err).NotTo(HaveOccurred(), "stderr: %s", stderr)
	})

	It("can run Pods", func() {
		By("waiting the default service account gets created")
		Eventually(func() error {
			_, stderr, err := kubectl("get", "sa/default", "-o", "json")
			if err != nil {
				return fmt.Errorf("%v: stderr=%s", err, stderr)
			}
			return nil
		}).Should(Succeed())

		By("running nginx")
		_, stderr, err := kubectl("run", "nginx", "-n=mtest", "--image=nginx",
			`--overrides={"spec": {"hostNetwork": true}}`,
			"--generator=run-pod/v1")
		Expect(err).NotTo(HaveOccurred(), "stderr=%s", stderr)

		By("checking nginx pod status")
		Eventually(func() error {
			stdout, stderr, err := kubectl("get", "pods/nginx", "-n=mtest", "-o", "json")
			if err != nil {
				return fmt.Errorf("%v: stderr=%s", err, stderr)
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

	It("has cluster DNS resources", func() {
		for resource, name := range map[string]string{
			"serviceaccounts":     "cluster-dns",
			"clusterroles":        "system:cluster-dns",
			"clusterrolebindings": "system:cluster-dns",
			"configmaps":          "cluster-dns",
			"deployments":         "cluster-dns",
			"services":            "cluster-dns",
		} {
			_, stderr, err := kubectl("-n", "kube-system", "get", resource+"/"+name)
			Expect(err).NotTo(HaveOccurred(), "stderr=%s", stderr)
		}

		stdout, stderr, err := kubectl("-n", "kube-system", "get", "configmaps/cluster-dns", "-o=json")
		Expect(err).NotTo(HaveOccurred(), "stderr=%s", stderr)

		configMap := new(corev1.ConfigMap)
		err = json.Unmarshal(stdout, configMap)
		Expect(err).ShouldNot(HaveOccurred())

		domain, ok := configMap.ObjectMeta.Labels[op.ClusterDNSLabelDomain]
		Expect(ok).Should(BeTrue())
		Expect(domain).Should(Equal("cluster.local"))

		dnsServers, ok := configMap.ObjectMeta.Labels[op.ClusterDNSLabelDNSServers]
		Expect(ok).Should(BeTrue())
		Expect(dnsServers).Should(Equal("8.8.8.8_1.1.1.1"))
	})

	It("resolves Service IP", func() {
		By("getting CoreDNS Pods")
		stdout, stderr, err := kubectl("get", "-n=kube-system", "pods", "--selector=k8s-app=cluster-dns", "-o=json")
		Expect(err).NotTo(HaveOccurred(), "stderr: %s", stderr)

		var pods corev1.PodList
		err = json.Unmarshal(stdout, &pods)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(pods.Items).To(HaveLen(2))

		node := pods.Items[0].Spec.NodeName

		By("deploying Service resource")
		_, stderr, err = kubectl("run", "nginx", "-n=mtest", "--image=nginx")
		Expect(err).NotTo(HaveOccurred(), "stderr=%s", stderr)

		_, stderr, err = kubectl("expose", "-n=mtest", "deployments", "nginx", "--port=80")
		Expect(err).NotTo(HaveOccurred(), "stderr=%s", stderr)

		overrides := fmt.Sprintf(`{ "apiVersion": "v1", "spec": { "nodeSelector": { "kubernetes.io/hostname": "%s" }}}`, node)
		_, stderr, err = kubectl("run",
			"-n=mtest", "--image=quay.io/cybozu/ubuntu:18.04", "--overrides="+overrides+"", "--restart=Never", "client", "--",
			"sleep", "infinity")
		Expect(err).NotTo(HaveOccurred(), "stderr: %s", stderr)

		By("waiting pods are ready")
		Eventually(func() error {
			_, stderr, err = kubectl("exec", "-n=mtest", "client", "true")
			if err != nil {
				return fmt.Errorf("%v: stderr=%s", err, stderr)
			}
			return nil
		}).Should(Succeed())

		By("resolving domain names")
		_, stderr, err = kubectl("exec", "-n=mtest", "client", "getent", "hosts", "nginx")
		Expect(err).NotTo(HaveOccurred(), "stderr: %s", stderr)

		_, stderr, err = kubectl("exec", "-n=mtest", "client", "getent", "hosts", "nginx.mtest.svc.cluster.local")
		Expect(err).NotTo(HaveOccurred(), "stderr: %s", stderr)
	})

	It("updates unbound config", func() {
		stdout, stderr, err := kubectl("get", "-n=kube-system", "pods", "--selector=k8s-app=cluster-dns", "-o=json")
		Expect(err).NotTo(HaveOccurred(), "stderr: %s", stderr)

		var pods corev1.PodList
		err = json.Unmarshal(stdout, &pods)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(pods.Items).To(HaveLen(2))

		node := pods.Items[0].Spec.NodeName

		cluster := getCluster()
		for i := 0; i < 3; i++ {
			cluster.Nodes[i].ControlPlane = true
		}

		_, stderr, err = kubectl("run", "nginx", "-n=mtest", "--image=nginx")
		Expect(err).NotTo(HaveOccurred(), "stderr=%s", stderr)

		_, stderr, err = kubectl("expose", "-n=mtest", "deployments", "nginx", "--port=80")
		Expect(err).NotTo(HaveOccurred(), "stderr=%s", stderr)

		By("updating domain name to neco-cluster.local")
		before := cluster.Options.Kubelet.Domain
		cluster.Options.Kubelet.Domain = "neco.local"
		ckecliClusterSet(cluster)
		Eventually(func() error {
			return checkCluster(cluster)
		}).Should(Succeed())

		overrides := fmt.Sprintf(`{ "apiVersion": "v1", "spec": { "nodeSelector": { "kubernetes.io/hostname": "%s" }}}`, node)
		_, stderr, err = kubectl("run",
			"-n=mtest", "--image=quay.io/cybozu/ubuntu:18.04", "--overrides="+overrides+"", "--restart=Never", "client1", "--",
			"sleep", "infinity")
		Expect(err).NotTo(HaveOccurred(), "stderr: %s", stderr)

		By("waiting unbound confid is updated")
		Eventually(func() error {
			_, stderr, err = kubectl("exec", "-n=mtest", "client1", "getent", "hosts", "nginx.mtest.svc.neco.local")
			if err != nil {
				return fmt.Errorf("%v: stderr=%s", err, stderr)
			}
			return nil
		}).Should(Succeed())

		cluster.Options.Kubelet.Domain = before
		ckecliClusterSet(cluster)
		Eventually(func() error {
			return checkCluster(cluster)
		}).Should(Succeed())

		overrides = fmt.Sprintf(`{ "apiVersion": "v1", "spec": { "nodeSelector": { "kubernetes.io/hostname": "%s" }}}`, node)
		_, stderr, err = kubectl("run",
			"-n=mtest", "--image=quay.io/cybozu/ubuntu:18.04", "--overrides="+overrides+"", "--restart=Never", "client2", "--",
			"sleep", "infinity")
		Expect(err).NotTo(HaveOccurred(), "stderr: %s", stderr)

		By("waiting unbound confid is updated")
		Eventually(func() error {
			_, stderr, err = kubectl("exec", "-n=mtest", "client2", "getent", "hosts", "nginx.mtest.svc."+before)
			if err != nil {
				return fmt.Errorf("%v: stderr=%s", err, stderr)
			}
			return nil
		}).Should(Succeed())
	})

	It("has node DNS resources", func() {
		for _, name := range []string{"configmaps/node-dns", "daemonsets/node-dns"} {
			_, stderr, err := kubectl("-n", "kube-system", "get", name)
			Expect(err).NotTo(HaveOccurred(), "stderr=%s", stderr)
		}

		By("checking node DNS pod status")
		Eventually(func() error {
			stdout, stderr, err := kubectl("-n", "kube-system", "get", "daemonsets/node-dns", "-o", "json")
			if err != nil {
				return fmt.Errorf("%v: stderr=%s", err, stderr)
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

		By("querying www.google.com using node DNS from ubuntu pod")
		_, stderr, err := kubectl("run", "-it", "--rm", "-n=mtest", "ubuntu",
			"--image=quay.io/cybozu/ubuntu-debug:18.04", "--generator=run-pod/v1",
			"--restart=Never", "--command", "--", "host", "-N", "0", "www.google.com")
		Expect(err).NotTo(HaveOccurred(), "stderr=%s", stderr)
	})

	It("has kube-system/cke-etcd Service and Endpoints", func() {
		_, stderr, err := kubectl("-n", "kube-system", "get", "services/cke-etcd")
		Expect(err).NotTo(HaveOccurred(), "stderr=%s", stderr)
		_, stderr, err = kubectl("-n", "kube-system", "get", "endpoints/cke-etcd")
		Expect(err).NotTo(HaveOccurred(), "stderr=%s", stderr)
	})
})
