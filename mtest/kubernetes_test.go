package mtest

import (
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	bolt "github.com/coreos/bbolt"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
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
	})

	It("resolves Service IP", func() {
		By("getting CoreDNS Pods")
		stdout, stderr, err := kubectl("get", "-n=kube-system", "pods", "--selector=cke.cybozu.com/appname=cluster-dns", "-o=json")
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
		cluster := getCluster()
		for i := 0; i < 3; i++ {
			cluster.Nodes[i].ControlPlane = true
		}

		By("updating domain name to neco.local")
		before := cluster.Options.Kubelet.Domain
		cluster.Options.Kubelet.Domain = "neco.local"
		ckecliClusterSet(cluster)
		Eventually(func() error {
			return checkCluster(cluster)
		}).Should(Succeed())

		stdout, stderr, err := kubectl("get", "-n=kube-system", "pods", "--selector=cke.cybozu.com/appname=node-dns", "-o=json")
		Expect(err).NotTo(HaveOccurred(), "stderr: %s", stderr)

		var pods corev1.PodList
		err = json.Unmarshal(stdout, &pods)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(pods.Items).NotTo(BeEmpty())
		pod := pods.Items[0]

		Eventually(func() error {
			stdout, stderr, err := kubectl("exec", "-n=kube-system", pod.Name, "-c=unbound",
				"/usr/local/unbound/sbin/unbound-control", "--",
				"-c", "/etc/unbound/unbound.conf", "list_stubs")
			if err != nil {
				return fmt.Errorf("%v: %s", err, string(stderr))
			}
			if strings.Contains(string(stdout), "neco.local. IN stub") {
				return nil
			}
			return errors.New("unbound.conf is not updated")
		}).Should(Succeed())

		cluster.Options.Kubelet.Domain = before
		ckecliClusterSet(cluster)
		Eventually(func() error {
			return checkCluster(cluster)
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

	It("can backup etcd snapshot", func() {
		By("deploying local persistent volume")
		_, stderr, err := kubectl("create", "-f", "local-pv.yml")
		Expect(err).NotTo(HaveOccurred(), "stderr=%s", stderr)

		By("enabling etcd backup")
		cluster := getCluster()
		for i := 0; i < 3; i++ {
			cluster.Nodes[i].ControlPlane = true
		}
		cluster.EtcdBackup.Enabled = true
		ckecliClusterSet(cluster)
		Eventually(func() error {
			return checkCluster(cluster)
		}).Should(Succeed())

		By("getting hostIP of etcdbackup Pod")
		var hostIP string
		Eventually(func() error {
			stdout, _, err := kubectl("-n", "kube-system", "get", "pods/etcdbackup", "-o", "json")
			if err != nil {
				return err
			}
			var pod corev1.Pod
			if err := json.Unmarshal(stdout, &pod); err != nil {
				return err
			}
			hostIP = pod.Status.HostIP
			if hostIP == "" {
				return errors.New("etcdbackup pod is not scheduled")
			}
			return nil
		}).Should(Succeed())

		By("deploying cluster-dns to etcdbackup Pod running hostIP")
		clusterDNSPatch := fmt.Sprintf(`{ "spec": { "template": { "spec": { "nodeSelector": { "kubernetes.io/hostname": "%s" } } } } } }`, hostIP)
		_, stderr, err = kubectl("patch", "deployment", "cluster-dns", "-n", "kube-system", "--patch="+clusterDNSPatch)
		Expect(err).NotTo(HaveOccurred(), "stderr=%s", stderr)

		By("deploying etcdbackup CronJob to etcdbackup Pod running hostIP")
		etcdbackupPatch := fmt.Sprintf(`{"spec": { "jobTemplate": { "spec": { "template": { "spec": { "nodeSelector": { "kubernetes.io/hostname": "%s" } } } } } } }`, hostIP)
		_, stderr, err = kubectl("patch", "cronjob", "etcdbackup", "-n", "kube-system", "--patch="+etcdbackupPatch)
		Expect(err).NotTo(HaveOccurred(), "stderr=%s", stderr)

		By("checking etcd backup job status")
		Eventually(func() error {
			stdout, stderr, err := kubectl("-n", "kube-system", "get", "job", "--sort-by=.metadata.creationTimestamp", "-o", "json")
			if err != nil {
				return fmt.Errorf("%v: stderr=%s", err, stderr)
			}

			var jobs batchv1.JobList
			err = json.Unmarshal(stdout, &jobs)
			if err != nil {
				return err
			}

			if len(jobs.Items) < 1 {
				return fmt.Errorf("no etcd backup jobs, JobList: %v", jobs)
			}
			if jobs.Items[0].Status.Succeeded != 1 {
				return fmt.Errorf(".Succeeded is not 1, JobList: %v", jobs)
			}

			return nil
		}).Should(Succeed())

		By("checking etcd snapshot is correct")
		stdout := ckecli("etcd", "backup", "list")
		var list []string
		err = json.Unmarshal(stdout, &list)
		Expect(err).NotTo(HaveOccurred(), "stderr=%s", stderr)
		Expect(list[0]).To(ContainSubstring("snapshot-"))

		ckecli("etcd", "backup", "get", list[0])
		gzfile, err := os.Open(list[0])
		Expect(err).ShouldNot(HaveOccurred())
		defer gzfile.Close()
		zr, err := gzip.NewReader(gzfile)
		Expect(err).ShouldNot(HaveOccurred())
		defer zr.Close()

		dbfile, err := os.Create("snapshot.db")
		Expect(err).ShouldNot(HaveOccurred())
		defer func() {
			dbfile.Close()
			os.Remove(dbfile.Name())
		}()
		_, err = io.Copy(dbfile, zr)
		Expect(err).ShouldNot(HaveOccurred())
		db, err := bolt.Open(dbfile.Name(), 0400, &bolt.Options{ReadOnly: true})
		Expect(err).ShouldNot(HaveOccurred())
		defer db.Close()

		By("confirming etcdbackup CronJob is removed when etcdbackup is disabled")
		cluster.EtcdBackup.Enabled = false
		ckecliClusterSet(cluster)
		Eventually(func() error {
			return checkCluster(cluster)
		}).Should(Succeed())
	})
})
