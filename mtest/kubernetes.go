package mtest

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
)

// TestKubernetes tests kubernetes workloads on CKE
func TestKubernetes() {
	It("can run Pods", func() {
		namespace := fmt.Sprintf("mtest-%d", getRandomNumber().Int())
		By("creating namespace " + namespace)
		_, stderr, err := kubectl("create", "namespace", namespace)
		Expect(err).NotTo(HaveOccurred(), "stderr: %s", stderr)
		psp, err := ioutil.ReadFile(policyYAMLPath)
		Expect(err).ShouldNot(HaveOccurred())
		_, stderr, err = kubectlWithInput(psp, "apply", "-f", "-", "-n="+namespace)
		Expect(err).NotTo(HaveOccurred(), "stderr: %s", stderr)

		By("waiting the default service account gets created")
		Eventually(func() error {
			_, stderr, err := kubectl("get", "sa/default", "-o", "json", "-n="+namespace)
			if err != nil {
				return fmt.Errorf("%v: stderr=%s", err, stderr)
			}
			return nil
		}).Should(Succeed())

		By("running nginx")
		nginx, err := ioutil.ReadFile(nginxYAMLPath)
		Expect(err).ShouldNot(HaveOccurred())
		_, stderr, err = kubectlWithInput(nginx, "apply", "-f", "-", "-n="+namespace)
		Expect(err).NotTo(HaveOccurred(), "stderr=%s", stderr)

		By("checking nginx pod status")
		Eventually(func() error {
			stdout, stderr, err := kubectl("get", "pods/nginx", "-n="+namespace, "-o", "json")
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
			"serviceaccounts":     "cke-cluster-dns",
			"clusterroles":        "system:cluster-dns",
			"clusterrolebindings": "system:cluster-dns",
			"configmaps":          "cluster-dns",
			"deployments":         "cluster-dns",
			"services":            "cluster-dns",
		} {
			stdout, stderr, err := kubectl("-n", "kube-system", "get", "-v=8", resource+"/"+name)
			Expect(err).NotTo(HaveOccurred(), "resource=%s/%s, stdout=%s, stderr=%s", resource, name, stdout, stderr)
		}

		stdout, stderr, err := kubectl("-n", "kube-system", "get", "configmaps/cluster-dns", "-o=json")
		Expect(err).NotTo(HaveOccurred(), "stderr=%s", stderr)

		configMap := new(corev1.ConfigMap)
		err = json.Unmarshal(stdout, configMap)
		Expect(err).ShouldNot(HaveOccurred())
	})

	It("resolves Service IP", func() {
		namespace := fmt.Sprintf("mtest-%d", getRandomNumber().Int())
		By("creating namespace " + namespace)
		_, stderr, err := kubectl("create", "namespace", namespace)
		Expect(err).NotTo(HaveOccurred(), "stderr: %s", stderr)
		psp, err := ioutil.ReadFile(policyYAMLPath)
		Expect(err).ShouldNot(HaveOccurred())
		_, stderr, err = kubectlWithInput(psp, "apply", "-f", "-", "-n="+namespace)
		Expect(err).NotTo(HaveOccurred(), "stderr: %s", stderr)

		var node string
		By("getting CoreDNS Pods")
		Eventually(func() error {
			stdout, stderr, err := kubectl("get", "-n=kube-system", "pods", "--selector=cke.cybozu.com/appname=cluster-dns", "-o=json")
			if err != nil {
				return fmt.Errorf("%v: stderr=%s", err, stderr)
			}

			var pods corev1.PodList
			err = json.Unmarshal(stdout, &pods)
			if err != nil {
				return fmt.Errorf("%v: stdout=%s", err, stdout)
			}
			if len(pods.Items) != 2 {
				return fmt.Errorf("len(pods.Items) should be 2: %d", len(pods.Items))
			}
			node = pods.Items[0].Spec.NodeName
			return nil
		}).Should(Succeed())

		By("deploying Service resource")
		nginx, err := ioutil.ReadFile(nginxYAMLPath)
		Expect(err).ShouldNot(HaveOccurred())
		_, stderr, err = kubectlWithInput(nginx, "apply", "-f", "-", "-n="+namespace)
		Expect(err).NotTo(HaveOccurred(), "stderr=%s", stderr)

		_, stderr, err = kubectl("expose", "-n="+namespace, "pod", "nginx", "--port=8000")
		Expect(err).NotTo(HaveOccurred(), "stderr=%s", stderr)

		overrides := fmt.Sprintf(`{
	"apiVersion": "v1",
	"spec": { "nodeSelector": { "kubernetes.io/hostname": "%s" }}
}`, node)
		overrideFile := remoteTempFile(overrides)
		_, stderr, err = kubectl("run",
			"-n="+namespace, "--image=quay.io/cybozu/ubuntu:18.04", "--overrides=\"$(cat "+overrideFile+")\"", "--restart=Never",
			"client", "--", "pause")
		Expect(err).NotTo(HaveOccurred(), "stderr: %s", stderr)

		By("waiting pods are ready")
		Eventually(func() error {
			_, stderr, err = kubectl("exec", "-n="+namespace, "client", "true")
			if err != nil {
				return fmt.Errorf("%v: stderr=%s", err, stderr)
			}
			return nil
		}).Should(Succeed())

		By("resolving domain names")
		Eventually(func() error {
			_, stderr, err := kubectl("exec", "-n="+namespace, "client", "getent", "hosts", "nginx")
			if err != nil {
				return fmt.Errorf("%v: stderr=%s", err, stderr)
			}

			_, stderr, err = kubectl("exec", "-n="+namespace, "client", "getent", "hosts", "nginx."+namespace+".svc.cluster.local")
			if err != nil {
				return fmt.Errorf("%v: stderr=%s", err, stderr)
			}
			return nil
		}).Should(Succeed())
	})

	It("updates unbound config", func() {
		cluster := getCluster()
		for i := 0; i < 3; i++ {
			cluster.Nodes[i].ControlPlane = true
		}

		By("updating domain name to neco.local")
		before := cluster.Options.Kubelet.Domain
		if cluster.Options.Kubelet.Config == nil {
			cluster.Options.Kubelet.Domain = "neco.local"
		} else {
			cluster.Options.Kubelet.Config.Object["clusterDomain"] = "neco.local"
		}
		clusterSetAndWait(cluster)

		stdout, stderr, err := kubectl("get", "-n=kube-system", "pods", "--selector=cke.cybozu.com/appname=node-dns", "-o=json")
		Expect(err).NotTo(HaveOccurred(), "stderr: %s", stderr)

		var pods corev1.PodList
		err = json.Unmarshal(stdout, &pods)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(pods.Items).NotTo(BeEmpty())
		pod := pods.Items[0]

		Eventually(func() error {
			stdout, stderr, err := kubectl("exec", "-n=kube-system", pod.Name, "-c=unbound",
				"--", "/usr/local/unbound/sbin/unbound-control",
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
		clusterSetAndWait(cluster)
	})

	It("has node DNS resources", func() {
		namespace := fmt.Sprintf("mtest-%d", getRandomNumber().Int())
		By("creating namespace " + namespace)
		_, stderr, err := kubectl("create", "namespace", namespace)
		Expect(err).NotTo(HaveOccurred(), "stderr: %s", stderr)
		psp, err := ioutil.ReadFile(policyYAMLPath)
		Expect(err).ShouldNot(HaveOccurred())
		_, stderr, err = kubectlWithInput(psp, "apply", "-f", "-", "-n="+namespace)
		Expect(err).NotTo(HaveOccurred(), "stderr: %s", stderr)

		for _, name := range []string{
			"configmaps/node-dns",
			"daemonsets/node-dns",
			"serviceaccounts/cke-node-dns",
		} {
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
		_, stderr, err = kubectl("run", "-n="+namespace, "--image=quay.io/cybozu/ubuntu:18.04", "--restart=Never",
			"client", "--", "pause")
		Expect(err).NotTo(HaveOccurred(), "stderr: %s", stderr)
		Eventually(func() error {
			_, _, err := kubectl("exec", "-n="+namespace, "client", "getent", "hosts", "www.cybozu.com")
			return err
		}).Should(Succeed())
	})

	It("has kube-system/cke-etcd Service and Endpoints", func() {
		_, stderr, err := kubectl("-n", "kube-system", "get", "services/cke-etcd")
		Expect(err).NotTo(HaveOccurred(), "stderr=%s", stderr)
		_, stderr, err = kubectl("-n", "kube-system", "get", "endpoints/cke-etcd")
		Expect(err).NotTo(HaveOccurred(), "stderr=%s", stderr)
	})

	It("can backup etcd snapshot", func() {
		By("deploying local persistent volume")
		pv, err := ioutil.ReadFile(localPVYAMLPath)
		Expect(err).ShouldNot(HaveOccurred())
		_, stderr, err := kubectlWithInput(pv, "create", "-f", "-")
		Expect(err).NotTo(HaveOccurred(), "stderr=%s", stderr)

		By("enabling etcd backup")
		cluster := getCluster()
		for i := 0; i < 3; i++ {
			cluster.Nodes[i].ControlPlane = true
		}
		cluster.EtcdBackup.Enabled = true
		clusterSetAndWait(cluster)

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
				return fmt.Errorf("etcdbackup pod is not scheduled: %s", pod.String())
			}
			return nil
		}).Should(Succeed())

		By("deploying cluster-dns to etcdbackup Pod running hostIP")
		clusterDNSPatch := fmt.Sprintf(`{ "spec": { "template": { "spec": { "nodeSelector": { "kubernetes.io/hostname": "%s" } } } } } }`, hostIP)
		patchFile := remoteTempFile(clusterDNSPatch)
		_, stderr, err = kubectl("patch", "deployment", "cluster-dns", "-n", "kube-system", "--patch=\"$(cat "+patchFile+")\"")
		Expect(err).NotTo(HaveOccurred(), "stderr=%s", stderr)

		By("deploying etcdbackup CronJob to etcdbackup Pod running hostIP")
		etcdbackupPatch := fmt.Sprintf(`{"spec": { "jobTemplate": { "spec": { "template": { "spec": { "nodeSelector": { "kubernetes.io/hostname": "%s" } } } } } } }`, hostIP)
		patchFile = remoteTempFile(etcdbackupPatch)
		_, stderr, err = kubectl("patch", "cronjob", "etcdbackup", "-n", "kube-system", "--patch=\"$(cat "+patchFile+")\"")
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
			if jobs.Items[len(jobs.Items)-1].Status.Succeeded != 1 {
				return fmt.Errorf(".Succeeded is not 1, JobList: %v", jobs)
			}

			return nil
		}).Should(Succeed())

		By("checking etcd snapshot is correct")
		stdout := ckecliSafe("etcd", "backup", "list")
		var list []string
		err = json.Unmarshal(stdout, &list)
		Expect(err).NotTo(HaveOccurred(), "stderr=%s", stderr)
		Expect(list[0]).To(ContainSubstring("snapshot-"))

		ckecliSafe("etcd", "backup", "get", list[0])
		_, stderr, err = execAt(host1, "gunzip", "snapshot-*.db.gz")
		Expect(err).NotTo(HaveOccurred(), "stderr=%s", stderr)
		stdout, stderr, err = execAt(host1, "env", "ETCDCTL_API=3", "/opt/bin/etcdctl", "snapshot", "status", "snapshot-*.db")
		Expect(err).NotTo(HaveOccurred(), "stdout=%s, stderr=%s", stdout, stderr)

		By("confirming etcdbackup CronJob is removed when etcdbackup is disabled")
		cluster.EtcdBackup.Enabled = false
		clusterSetAndWait(cluster)
	})

	It("can output audit log", func() {
		By("confirming journald does not have audit log")
		logs, _, err := execAt(node1, "sudo", "journalctl", "CONTAINER_NAME=kube-apiserver", "-p", "6..6", "-q")
		Expect(err).ShouldNot(HaveOccurred())
		Expect(logs).Should(BeEmpty())

		By("enabling audit log")
		cluster := getCluster()
		for i := 0; i < 3; i++ {
			cluster.Nodes[i].ControlPlane = true
		}
		cluster.Options.APIServer.AuditLogEnabled = true
		cluster.Options.APIServer.AuditLogPolicy = `apiVersion: audit.k8s.io/v1
kind: Policy
rules:
- level: Metadata`
		clusterSetAndWait(cluster)
		logs, _, err = execAt(node1, "sudo", "journalctl", "CONTAINER_NAME=kube-apiserver", "-p", "6..6", "-q")
		Expect(err).ShouldNot(HaveOccurred())
		Expect(logs).ShouldNot(BeEmpty())
		status, _, err := getClusterStatus(cluster)
		Expect(err).ShouldNot(HaveOccurred())
		var policyFile string
		for _, v := range status.NodeStatuses[node1].APIServer.BuiltInParams.ExtraArguments {
			if strings.HasPrefix(v, "--audit-policy-file=") {
				policyFile = v
				break
			}
		}
		Expect(policyFile).ShouldNot(BeEmpty())

		By("changing audit policy")
		cluster.Options.APIServer.AuditLogPolicy = `apiVersion: audit.k8s.io/v1
kind: Policy
rules:
- level: Request`
		clusterSetAndWait(cluster)
		status, _, err = getClusterStatus(cluster)
		Expect(err).ShouldNot(HaveOccurred())
		var currentPolicyFile string
		for _, v := range status.NodeStatuses[node1].APIServer.BuiltInParams.ExtraArguments {
			if strings.HasPrefix(v, "--audit-policy-file=") {
				currentPolicyFile = v
				break
			}
		}
		Expect(currentPolicyFile).ShouldNot(BeEmpty())
		Expect(currentPolicyFile).ShouldNot(Equal(policyFile))

		By("disabling audit log")
		cluster.Options.APIServer.AuditLogEnabled = false
		cluster.Options.APIServer.AuditLogPolicy = ""
		clusterSetAndWait(cluster)
	})

	It("updates user-defined resources", func() {
		By("set user-defined resource")
		resources := `apiVersion: v1
kind: Namespace
metadata:
  name: foo
---
apiVersion: v1
kind: ServiceAccount
metadata:
  namespace: foo
  name: sa1
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  namespace: foo
  name: pod-reader
rules:
- apiGroups: [""]
  resources: ["pods"]
  verbs: ["get", "watch", "list"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: read-pods
  namespace: foo
subjects:
- kind: ServiceAccount
  name: sa1
  namespace: foo
roleRef:
  kind: Role
  name: pod-reader
  apiGroup: rbac.authorization.k8s.io
`
		ckecliWithInput([]byte(resources), "resource", "set", "-")
		defer ckecliWithInput([]byte(resources), "resource", "delete", "-")
		ts := time.Now()

		cluster := getCluster()
		for i := 0; i < 3; i++ {
			cluster.Nodes[i].ControlPlane = true
		}
		Eventually(func() error {
			return checkCluster(cluster, ts)
		}).Should(Succeed())

		By("updating user-defined resources")
		newResources := `apiVersion: v1
kind: Namespace
metadata:
  name: foo
  labels:
    test: value
`
		ckecliWithInput([]byte(newResources), "resource", "set", "-")
		defer ckecliWithInput([]byte(newResources), "resource", "delete", "-")
		ts = time.Now()
		Eventually(func() error {
			return checkCluster(cluster, ts)
		}).Should(Succeed())

		stdout, _, err := kubectl("get", "namespaces/foo", "-o", "json")
		Expect(err).ShouldNot(HaveOccurred())
		var ns corev1.Namespace
		err = json.Unmarshal(stdout, &ns)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(ns.Labels).Should(HaveKeyWithValue("test", "value"))
	})
}
