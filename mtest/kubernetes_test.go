package mtest

import (
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func testKubernetes() {
	It("can run Pods", func() {
		namespace := fmt.Sprintf("mtest-%d", getRandomNumber().Int())
		By("creating namespace " + namespace)
		_, stderr, err := kubectl("create", "namespace", namespace)
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
		_, stderr, err = kubectlWithInput(nginxYAML, "apply", "-f", "-", "-n="+namespace)
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
		_, stderr, err = kubectlWithInput(nginxYAML, "apply", "-f", "-", "-n="+namespace)
		Expect(err).NotTo(HaveOccurred(), "stderr=%s", stderr)

		_, stderr, err = kubectl("expose", "-n="+namespace, "pod", "nginx", "--port=8000")
		Expect(err).NotTo(HaveOccurred(), "stderr=%s", stderr)

		overrides := fmt.Sprintf(`{
	"apiVersion": "v1",
	"spec": { "nodeSelector": { "kubernetes.io/hostname": "%s" }}
}`, node)
		overrideFile := remoteTempFile(overrides)
		_, stderr, err = kubectl("run",
			"-n="+namespace, "--image=quay.io/cybozu/ubuntu:20.04", "--overrides=\"$(cat "+overrideFile+")\"", "--restart=Never",
			"client", "--", "pause")
		Expect(err).NotTo(HaveOccurred(), "stderr: %s, err: %v", stderr, err)

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
		if cluster.Options.Kubelet.Config == nil {
			cluster.Options.Kubelet.Config = &unstructured.Unstructured{}
		}
		cluster.Options.Kubelet.Config.UnstructuredContent()["clusterDomain"] = "neco.local"
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

		cluster = getCluster()
		for i := 0; i < 3; i++ {
			cluster.Nodes[i].ControlPlane = true
		}
		clusterSetAndWait(cluster)
	})

	It("has node DNS resources", func() {
		namespace := fmt.Sprintf("mtest-%d", getRandomNumber().Int())
		By("creating namespace " + namespace)
		_, stderr, err := kubectl("create", "namespace", namespace)
		Expect(err).NotTo(HaveOccurred(), "stderr: %s", stderr)

		By("waiting the default service account gets created")
		Eventually(func() error {
			_, stderr, err := kubectl("get", "sa/default", "-o", "json", "-n="+namespace)
			if err != nil {
				return fmt.Errorf("%v: stderr=%s", err, stderr)
			}
			return nil
		}).Should(Succeed())

		for _, name := range []string{
			"configmaps/node-dns",
			"daemonsets/node-dns",
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
		_, stderr, err = kubectl("run", "-n="+namespace, "--image=quay.io/cybozu/ubuntu:20.04", "--restart=Never",
			"client", "--", "pause")
		Expect(err).NotTo(HaveOccurred(), "stderr: %s", stderr)
		Eventually(func() error {
			_, _, err := kubectl("exec", "-n="+namespace, "client", "getent", "hosts", "www.cybozu.com")
			return err
		}).Should(Succeed())

		By("getting metrics from unbound_exporter")
		Eventually(func() error {
			stdout, _, err := kubectl("exec", "-n=kube-system", "daemonset/node-dns", "-c", "unbound", "--", "curl", "-sSf", "http://127.0.0.1:9167/metrics")
			if err != nil {
				return err
			}
			if !strings.Contains(string(stdout), "unbound_up 1") {
				return errors.New("exporter does not return unbound_up=1")
			}
			if !strings.Contains(string(stdout), `unbound_memory_caches_bytes{cache="message"}`) {
				return errors.New("exporter does not return unbound_memory_caches_bytes")
			}
			return nil
		}).Should(Succeed())

		By("checking rollout restart of node DNS")
		getNodeDnsPodList := func() (*corev1.PodList, error) {
			stdout, stderr, err := kubectl("get", "pod", "-n", "kube-system", "-l", "cke.cybozu.com/appname=node-dns", "-o", "json")
			if err != nil {
				return nil, fmt.Errorf("stderr: %s, err: %w", stderr, err)
			}
			podList := corev1.PodList{}
			err = json.Unmarshal(stdout, &podList)
			if err != nil {
				return nil, err
			}
			return &podList, nil
		}
		beforeList, err := getNodeDnsPodList()
		Expect(err).NotTo(HaveOccurred())

		_, stderr, err = kubectl("rollout", "restart", "-n", "kube-system", "daemonsets/node-dns")
		Expect(err).NotTo(HaveOccurred(), "stderr: %s", stderr)

		Eventually(func() error {
			afterList, err := getNodeDnsPodList()
			if err != nil {
				return err
			}
			if len(afterList.Items) != len(beforeList.Items) {
				return fmt.Errorf("rollout is not completed: before=%d, after=%d", len(beforeList.Items), len(afterList.Items))
			}
			for _, apod := range afterList.Items {
				for _, bpod := range beforeList.Items {
					if apod.Name == bpod.Name && apod.CreationTimestamp == bpod.CreationTimestamp {
						return fmt.Errorf("rollout is not completed: pod %s is not restarted", apod.Name)
					}
				}
			}

			return nil
		}, time.Minute*5).Should(Succeed())
	})

	It("has kube-system/cke-etcd Service and Endpoints", func() {
		_, stderr, err := kubectl("-n", "kube-system", "get", "services/cke-etcd")
		Expect(err).NotTo(HaveOccurred(), "stderr=%s", stderr)
		_, stderr, err = kubectl("-n", "kube-system", "get", "endpoints/cke-etcd")
		Expect(err).NotTo(HaveOccurred(), "stderr=%s", stderr)
	})

	It("can output audit log to journal log", func() {
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
		cluster.Options.APIServer.AuditLogPath = ""
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

	It("can output audit log to a file", func() {
		By("confirming audit log file is not yet created")
		_, _, err := execAt(node1, "sudo", "ls", "/var/log/audit/audit.log")
		Expect(err).Should(HaveOccurred())

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
		cluster.Options.APIServer.AuditLogPath = "/var/log/audit/audit.log"
		clusterSetAndWait(cluster)
		logs, _, err := execAt(node1, "sudo", "cat", "/var/log/audit/audit.log")
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
---
apiVersion: apps/v1
kind: Deployment
metadata:
  namespace: foo
  name: test-deployment
  annotations:
    cke.cybozu.com/rank: "2200"
spec:
  replicas: 1
  selector:
    matchLabels:
      cke.cybozu.com/appname: test-deployment
  template:
    metadata:
      labels:
        cke.cybozu.com/appname: test-deployment
    spec:
      containers:
      - name: ubuntu
        image: quay.io/cybozu/ubuntu:20.04
        args:
        - pause
---
kind: DaemonSet
apiVersion: apps/v1
metadata:
  name: test-daemonset
  namespace: foo
  annotations:
    cke.cybozu.com/rank: "2100"
spec:
  selector:
    matchLabels:
      cke.cybozu.com/appname: test-daemonset
  template:
    metadata:
      labels:
        cke.cybozu.com/appname: test-daemonset
    spec:
      initContainers:
      - name: wait-1min
        image: quay.io/cybozu/ubuntu:20.04
        args:
        - sleep
        - "60"
      containers:
      - name: ubuntu
        image: quay.io/cybozu/ubuntu:20.04
        args:
        - pause
`

		By("checking the order of applying")
		_, _, err := ckecliWithInput([]byte(resources), "resource", "set", "-")
		Expect(err).NotTo(HaveOccurred())

		defer ckecliWithInput([]byte(resources), "resource", "delete", "-")
		ts := time.Now()

		By("waiting to complete creation")
		Eventually(func() error {
			stdout, _, err := kubectl("-n", "foo", "get", "daemonset", "test-daemonset", "-o", "json")
			if err != nil {
				return err
			}
			var ds appsv1.DaemonSet
			if err := json.Unmarshal(stdout, &ds); err != nil {
				return err
			}
			desired := ds.Status.DesiredNumberScheduled
			if desired == 0 {
				return fmt.Errorf("desired must be not 0")
			}
			available := ds.Status.NumberAvailable
			stdout, _, err = kubectl("-n", "foo", "get", "pods", "-l", "cke.cybozu.com/appname=test-deployment", "-o", "json")
			if err != nil {
				return err
			}
			podList := &corev1.PodList{}
			if err := json.Unmarshal(stdout, podList); err != nil {
				return err
			}

			if desired > available {
				// We expect that the array of pod.Items is empty
				if len(podList.Items) == 0 {
					return fmt.Errorf("should wait")
				}
				// If the array is not empty, test must fail and return
				Fail("Deployment(foo/test-deployment) resource is expected not to create before completing to create DaemonSet(test-daemonset)")
			}
			return nil
		}).WithTimeout(2 * time.Minute).Should(Succeed())

		By("getting deployment")
		Eventually(func() error {
			dp := &appsv1.Deployment{}
			stdout, _, err := kubectl("-n", "foo", "get", "deployment", "test-deployment", "-o", "json")
			if err != nil {
				return err
			}
			if err := json.Unmarshal(stdout, dp); err != nil {
				return err
			}
			return nil
		}).Should(Succeed())

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
---
apiVersion: apps/v1
kind: Deployment
metadata:
  namespace: foo
  name: test-deployment
  annotations:
    cke.cybozu.com/rank: "2200"
  labels:
    updated: "true"
spec:
  replicas: 1
  selector:
    matchLabels:
      cke.cybozu.com/appname: test-deployment
  template:
    metadata:
      labels:
        cke.cybozu.com/appname: test-deployment
        updated: "true"
    spec:
      containers:
      - name: ubuntu
        image: quay.io/cybozu/ubuntu:20.04
        args:
        - pause
---
kind: DaemonSet
apiVersion: apps/v1
metadata:
  name: test-daemonset
  namespace: foo
  annotations:
    cke.cybozu.com/rank: "2100"
  labels:
    updated: "true"
spec:
  selector:
    matchLabels:
      cke.cybozu.com/appname: test-daemonset
  template:
    metadata:
      labels:
        cke.cybozu.com/appname: test-daemonset
        updated: "true"
    spec:
      initContainers:
      - name: wait-1min
        image: quay.io/cybozu/ubuntu:20.04
        args:
        - sleep
        - "60"
      containers:
      - name: ubuntu
        image: quay.io/cybozu/ubuntu:20.04
        args:
        - pause
`
		ckecliWithInput([]byte(newResources), "resource", "set", "-")
		defer ckecliWithInput([]byte(newResources), "resource", "delete", "-")
		ts = time.Now()

		Eventually(func() error {
			stdout, _, err := kubectl("get", "namespaces/foo", "-o", "json")
			if err != nil {
				return err
			}
			var ns corev1.Namespace
			if err := json.Unmarshal(stdout, &ns); err != nil {
				return err
			}
			val, ok := ns.Labels["test"]
			if !ok || val != "value" {
				return fmt.Errorf("ns must have the label that key is test and value is value")
			}
			return nil
		}).Should(Succeed())

		By("waiting to complete the update of DaemonSet")
		Eventually(func() error {
			By("trying to get daemonset")
			stdout, _, err := kubectl("-n", "foo", "get", "daemonset", "-l", "updated=true", "-o", "json")
			if err != nil {
				return err
			}
			var dsList appsv1.DaemonSetList
			if err := json.Unmarshal(stdout, &dsList); err != nil {
				return err
			}
			if len(dsList.Items) == 0 {
				return fmt.Errorf("should wait to update")
			}
			// if the label is not exist, retry
			desired := dsList.Items[0].Status.DesiredNumberScheduled
			if desired == 0 {
				return fmt.Errorf("desired must be not 0")
			}
			available := dsList.Items[0].Status.NumberAvailable

			stdout, _, err = kubectl("-n", "foo", "get", "deployment", "-l", "updated=true", "-o", "json")
			if err != nil {
				return err
			}
			dpList := appsv1.DeploymentList{}
			if err := json.Unmarshal(stdout, &dpList); err != nil {
				return err
			}

			if desired > available {
				// We expect that the array of pod.Items is empty
				if len(dpList.Items) == 0 {
					return fmt.Errorf("should wait")
				}
				// If the array is not empty, test must fail and return
				Fail("Deployment(foo/test-deployment) resource is expected not to create before completing to create DaemonSet(test-daemonset)")
			}
			return nil
		}).Should(Succeed())

		By("getting deployment")
		dp := &appsv1.Deployment{}
		Eventually(func() error {
			stdout, _, err := kubectl("-n", "foo", "get", "deployment", "test-deployment", "-o", "json")
			if err != nil {
				return err
			}
			if err := json.Unmarshal(stdout, dp); err != nil {
				return err
			}
			l, ok := dp.Labels["update"]
			if !ok || l != "true" {
				return fmt.Errorf("test-deployment must has the label named update")
			}
			return nil
		}).Should(Succeed())
		Expect(dp.Labels).Should(HaveKeyWithValue("updated", "true"))

		Eventually(func() error {
			return checkCluster(cluster, ts)
		}).Should(Succeed())

	})

	It("embed certificates for webhooks", func() {
		By("set user-defined resource")
		_, _, err := ckecliWithInput(webhookYAML, "resource", "set", "-")
		Expect(err).NotTo(HaveOccurred())
		defer ckecliWithInput(webhookYAML, "resource", "delete", "-")
		ts := time.Now()

		cluster := getCluster()
		for i := 0; i < 3; i++ {
			cluster.Nodes[i].ControlPlane = true
		}
		Eventually(func() error {
			return checkCluster(cluster, ts)
		}).Should(Succeed())

		By("checking ValidatingWebhookConfiguration")
		stdout, _, err := kubectl("get", "validatingwebhookconfigurations/test", "-o", "json")
		Expect(err).ShouldNot(HaveOccurred())
		vwh := &admissionregistrationv1beta1.ValidatingWebhookConfiguration{}
		err = json.Unmarshal(stdout, vwh)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(vwh.Webhooks).Should(HaveLen(2))

		block, _ := pem.Decode(vwh.Webhooks[1].ClientConfig.CABundle)
		Expect(block).NotTo(BeNil())
		_, err = x509.ParseCertificate(block.Bytes)
		Expect(err).ShouldNot(HaveOccurred())

		By("checking MutatingWebhookConfiguration")
		stdout, _, err = kubectl("get", "mutatingwebhookconfigurations/test", "-o", "json")
		Expect(err).ShouldNot(HaveOccurred())
		mwh := &admissionregistrationv1beta1.MutatingWebhookConfiguration{}
		err = json.Unmarshal(stdout, mwh)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(mwh.Webhooks).Should(HaveLen(1))

		block, _ = pem.Decode(vwh.Webhooks[0].ClientConfig.CABundle)
		Expect(block).NotTo(BeNil())
		_, err = x509.ParseCertificate(block.Bytes)
		Expect(err).ShouldNot(HaveOccurred())

		By("checking Secret")
		stdout, _, err = kubectl("get", "secrets/webhook-cert", "-o", "json")
		Expect(err).ShouldNot(HaveOccurred())
		secret := &corev1.Secret{}
		err = json.Unmarshal(stdout, secret)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(secret.Data).To(HaveKey("tls.crt"))
		block, _ = pem.Decode(secret.Data["tls.crt"])
		Expect(block).NotTo(BeNil())
		cert, err := x509.ParseCertificate(block.Bytes)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(cert.DNSNames).Should(ContainElements(
			"webhook-service",
			"webhook-service.default",
			"webhook-service.default.svc",
		))
	})
}
