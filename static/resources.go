// Code generated by compile_resources. DO NOT EDIT.
//go:generate go run ../pkg/compile_resources

package static

import (
	"github.com/cybozu-go/cke"
)

// Resources is the Kubernetes resource definitions embedded in CKE.
var Resources = []cke.ResourceDefinition{
	{
		Key:        "ServiceAccount/kube-system/cke-cluster-dns",
		Kind:       "ServiceAccount",
		Namespace:  "kube-system",
		Name:       "cke-cluster-dns",
		Revision:   1,
		Image:      "",
		Definition: []byte("{\"kind\":\"ServiceAccount\",\"apiVersion\":\"v1\",\"metadata\":{\"name\":\"cke-cluster-dns\",\"namespace\":\"kube-system\",\"creationTimestamp\":null,\"annotations\":{\"cke.cybozu.com/revision\":\"1\"}}}\n"),
	},
	{
		Key:        "ServiceAccount/kube-system/cke-etcdbackup",
		Kind:       "ServiceAccount",
		Namespace:  "kube-system",
		Name:       "cke-etcdbackup",
		Revision:   1,
		Image:      "",
		Definition: []byte("{\"kind\":\"ServiceAccount\",\"apiVersion\":\"v1\",\"metadata\":{\"name\":\"cke-etcdbackup\",\"namespace\":\"kube-system\",\"creationTimestamp\":null,\"annotations\":{\"cke.cybozu.com/revision\":\"1\"}}}\n"),
	},
	{
		Key:        "ServiceAccount/kube-system/cke-node-dns",
		Kind:       "ServiceAccount",
		Namespace:  "kube-system",
		Name:       "cke-node-dns",
		Revision:   1,
		Image:      "",
		Definition: []byte("{\"kind\":\"ServiceAccount\",\"apiVersion\":\"v1\",\"metadata\":{\"name\":\"cke-node-dns\",\"namespace\":\"kube-system\",\"creationTimestamp\":null,\"annotations\":{\"cke.cybozu.com/revision\":\"1\"}}}\n"),
	},
	{
		Key:        "PodSecurityPolicy/cke-node-dns",
		Kind:       "PodSecurityPolicy",
		Namespace:  "",
		Name:       "cke-node-dns",
		Revision:   1,
		Image:      "",
		Definition: []byte("{\"kind\":\"PodSecurityPolicy\",\"apiVersion\":\"policy/v1beta1\",\"metadata\":{\"name\":\"cke-node-dns\",\"creationTimestamp\":null,\"annotations\":{\"cke.cybozu.com/revision\":\"1\",\"seccomp.security.alpha.kubernetes.io/allowedProfileNames\":\"docker/default\",\"seccomp.security.alpha.kubernetes.io/defaultProfileName\":\"docker/default\"}},\"spec\":{\"allowedCapabilities\":[\"NET_BIND_SERVICE\"],\"volumes\":[\"configMap\",\"emptyDir\",\"projected\",\"secret\",\"downwardAPI\",\"persistentVolumeClaim\"],\"hostNetwork\":true,\"seLinux\":{\"rule\":\"RunAsAny\"},\"runAsUser\":{\"rule\":\"RunAsAny\"},\"supplementalGroups\":{\"rule\":\"RunAsAny\"},\"fsGroup\":{\"rule\":\"RunAsAny\"},\"readOnlyRootFilesystem\":true,\"allowPrivilegeEscalation\":false}}\n"),
	},
	{
		Key:        "PodSecurityPolicy/cke-restricted",
		Kind:       "PodSecurityPolicy",
		Namespace:  "",
		Name:       "cke-restricted",
		Revision:   1,
		Image:      "",
		Definition: []byte("{\"kind\":\"PodSecurityPolicy\",\"apiVersion\":\"policy/v1beta1\",\"metadata\":{\"name\":\"cke-restricted\",\"creationTimestamp\":null,\"annotations\":{\"cke.cybozu.com/revision\":\"1\",\"seccomp.security.alpha.kubernetes.io/allowedProfileNames\":\"docker/default\",\"seccomp.security.alpha.kubernetes.io/defaultProfileName\":\"docker/default\"}},\"spec\":{\"requiredDropCapabilities\":[\"ALL\"],\"volumes\":[\"configMap\",\"emptyDir\",\"projected\",\"secret\",\"downwardAPI\",\"persistentVolumeClaim\"],\"seLinux\":{\"rule\":\"RunAsAny\"},\"runAsUser\":{\"rule\":\"MustRunAsNonRoot\"},\"supplementalGroups\":{\"rule\":\"MustRunAs\",\"ranges\":[{\"min\":1,\"max\":65535}]},\"fsGroup\":{\"rule\":\"MustRunAs\",\"ranges\":[{\"min\":1,\"max\":65535}]},\"readOnlyRootFilesystem\":true,\"allowPrivilegeEscalation\":false}}\n"),
	},
	{
		Key:        "ClusterRole/system:cluster-dns",
		Kind:       "ClusterRole",
		Namespace:  "",
		Name:       "system:cluster-dns",
		Revision:   1,
		Image:      "",
		Definition: []byte("{\"kind\":\"ClusterRole\",\"apiVersion\":\"rbac.authorization.k8s.io/v1\",\"metadata\":{\"name\":\"system:cluster-dns\",\"creationTimestamp\":null,\"labels\":{\"kubernetes.io/bootstrapping\":\"rbac-defaults\"},\"annotations\":{\"cke.cybozu.com/revision\":\"1\",\"rbac.authorization.kubernetes.io/autoupdate\":\"true\"}},\"rules\":[{\"verbs\":[\"list\",\"watch\"],\"apiGroups\":[\"\"],\"resources\":[\"endpoints\",\"services\",\"pods\",\"namespaces\"]},{\"verbs\":[\"use\"],\"apiGroups\":[\"policy\"],\"resources\":[\"podsecuritypolicies\"],\"resourceNames\":[\"cke-restricted\"]}]}\n"),
	},
	{
		Key:        "ClusterRole/system:kube-apiserver-to-kubelet",
		Kind:       "ClusterRole",
		Namespace:  "",
		Name:       "system:kube-apiserver-to-kubelet",
		Revision:   1,
		Image:      "",
		Definition: []byte("{\"kind\":\"ClusterRole\",\"apiVersion\":\"rbac.authorization.k8s.io/v1\",\"metadata\":{\"name\":\"system:kube-apiserver-to-kubelet\",\"creationTimestamp\":null,\"labels\":{\"kubernetes.io/bootstrapping\":\"rbac-defaults\"},\"annotations\":{\"cke.cybozu.com/revision\":\"1\",\"rbac.authorization.kubernetes.io/autoupdate\":\"true\"}},\"rules\":[{\"verbs\":[\"*\"],\"apiGroups\":[\"\"],\"resources\":[\"nodes/proxy\",\"nodes/stats\",\"nodes/log\",\"nodes/spec\",\"nodes/metrics\"]}]}\n"),
	},
	{
		Key:        "Role/kube-system/system:etcdbackup",
		Kind:       "Role",
		Namespace:  "kube-system",
		Name:       "system:etcdbackup",
		Revision:   1,
		Image:      "",
		Definition: []byte("{\"kind\":\"Role\",\"apiVersion\":\"rbac.authorization.k8s.io/v1\",\"metadata\":{\"name\":\"system:etcdbackup\",\"namespace\":\"kube-system\",\"creationTimestamp\":null,\"labels\":{\"kubernetes.io/bootstrapping\":\"rbac-defaults\"},\"annotations\":{\"cke.cybozu.com/revision\":\"1\",\"rbac.authorization.kubernetes.io/autoupdate\":\"true\"}},\"rules\":[{\"verbs\":[\"use\"],\"apiGroups\":[\"policy\"],\"resources\":[\"podsecuritypolicies\"],\"resourceNames\":[\"cke-restricted\"]}]}\n"),
	},
	{
		Key:        "Role/kube-system/system:node-dns",
		Kind:       "Role",
		Namespace:  "kube-system",
		Name:       "system:node-dns",
		Revision:   1,
		Image:      "",
		Definition: []byte("{\"kind\":\"Role\",\"apiVersion\":\"rbac.authorization.k8s.io/v1\",\"metadata\":{\"name\":\"system:node-dns\",\"namespace\":\"kube-system\",\"creationTimestamp\":null,\"labels\":{\"kubernetes.io/bootstrapping\":\"rbac-defaults\"},\"annotations\":{\"cke.cybozu.com/revision\":\"1\",\"rbac.authorization.kubernetes.io/autoupdate\":\"true\"}},\"rules\":[{\"verbs\":[\"use\"],\"apiGroups\":[\"policy\"],\"resources\":[\"podsecuritypolicies\"],\"resourceNames\":[\"cke-node-dns\"]}]}\n"),
	},
	{
		Key:        "ClusterRoleBinding/system:cluster-dns",
		Kind:       "ClusterRoleBinding",
		Namespace:  "",
		Name:       "system:cluster-dns",
		Revision:   1,
		Image:      "",
		Definition: []byte("{\"kind\":\"ClusterRoleBinding\",\"apiVersion\":\"rbac.authorization.k8s.io/v1\",\"metadata\":{\"name\":\"system:cluster-dns\",\"creationTimestamp\":null,\"labels\":{\"kubernetes.io/bootstrapping\":\"rbac-defaults\"},\"annotations\":{\"cke.cybozu.com/revision\":\"1\",\"rbac.authorization.kubernetes.io/autoupdate\":\"true\"}},\"subjects\":[{\"kind\":\"ServiceAccount\",\"name\":\"cke-cluster-dns\",\"namespace\":\"kube-system\"}],\"roleRef\":{\"apiGroup\":\"rbac.authorization.k8s.io\",\"kind\":\"ClusterRole\",\"name\":\"system:cluster-dns\"}}\n"),
	},
	{
		Key:        "ClusterRoleBinding/system:kube-apiserver",
		Kind:       "ClusterRoleBinding",
		Namespace:  "",
		Name:       "system:kube-apiserver",
		Revision:   1,
		Image:      "",
		Definition: []byte("{\"kind\":\"ClusterRoleBinding\",\"apiVersion\":\"rbac.authorization.k8s.io/v1\",\"metadata\":{\"name\":\"system:kube-apiserver\",\"creationTimestamp\":null,\"labels\":{\"kubernetes.io/bootstrapping\":\"rbac-defaults\"},\"annotations\":{\"cke.cybozu.com/revision\":\"1\",\"rbac.authorization.kubernetes.io/autoupdate\":\"true\"}},\"subjects\":[{\"kind\":\"User\",\"name\":\"kubernetes\"}],\"roleRef\":{\"apiGroup\":\"rbac.authorization.k8s.io\",\"kind\":\"ClusterRole\",\"name\":\"system:kube-apiserver-to-kubelet\"}}\n"),
	},
	{
		Key:        "RoleBinding/kube-system/system:etcdbackup",
		Kind:       "RoleBinding",
		Namespace:  "kube-system",
		Name:       "system:etcdbackup",
		Revision:   1,
		Image:      "",
		Definition: []byte("{\"kind\":\"RoleBinding\",\"apiVersion\":\"rbac.authorization.k8s.io/v1\",\"metadata\":{\"name\":\"system:etcdbackup\",\"namespace\":\"kube-system\",\"creationTimestamp\":null,\"labels\":{\"kubernetes.io/bootstrapping\":\"rbac-defaults\"},\"annotations\":{\"cke.cybozu.com/revision\":\"1\",\"rbac.authorization.kubernetes.io/autoupdate\":\"true\"}},\"subjects\":[{\"kind\":\"ServiceAccount\",\"name\":\"cke-etcdbackup\",\"namespace\":\"kube-system\"}],\"roleRef\":{\"apiGroup\":\"rbac.authorization.k8s.io\",\"kind\":\"Role\",\"name\":\"system:etcdbackup\"}}\n"),
	},
	{
		Key:        "RoleBinding/kube-system/system:node-dns",
		Kind:       "RoleBinding",
		Namespace:  "kube-system",
		Name:       "system:node-dns",
		Revision:   1,
		Image:      "",
		Definition: []byte("{\"kind\":\"RoleBinding\",\"apiVersion\":\"rbac.authorization.k8s.io/v1\",\"metadata\":{\"name\":\"system:node-dns\",\"namespace\":\"kube-system\",\"creationTimestamp\":null,\"labels\":{\"kubernetes.io/bootstrapping\":\"rbac-defaults\"},\"annotations\":{\"cke.cybozu.com/revision\":\"1\",\"rbac.authorization.kubernetes.io/autoupdate\":\"true\"}},\"subjects\":[{\"kind\":\"ServiceAccount\",\"name\":\"cke-node-dns\",\"namespace\":\"kube-system\"}],\"roleRef\":{\"apiGroup\":\"rbac.authorization.k8s.io\",\"kind\":\"Role\",\"name\":\"system:node-dns\"}}\n"),
	},
	{
		Key:        "Deployment/kube-system/cluster-dns",
		Kind:       "Deployment",
		Namespace:  "kube-system",
		Name:       "cluster-dns",
		Revision:   1,
		Image:      "quay.io/cybozu/coredns:1.6.2.1",
		Definition: []byte("{\"kind\":\"Deployment\",\"apiVersion\":\"apps/v1\",\"metadata\":{\"name\":\"cluster-dns\",\"namespace\":\"kube-system\",\"creationTimestamp\":null,\"annotations\":{\"cke.cybozu.com/image\":\"quay.io/cybozu/coredns:1.6.2.1\",\"cke.cybozu.com/revision\":\"1\"}},\"spec\":{\"replicas\":2,\"selector\":{\"matchLabels\":{\"cke.cybozu.com/appname\":\"cluster-dns\"}},\"template\":{\"metadata\":{\"creationTimestamp\":null,\"labels\":{\"cke.cybozu.com/appname\":\"cluster-dns\",\"k8s-app\":\"coredns\"}},\"spec\":{\"volumes\":[{\"name\":\"config-volume\",\"configMap\":{\"name\":\"cluster-dns\",\"items\":[{\"key\":\"Corefile\",\"path\":\"Corefile\"}]}}],\"containers\":[{\"name\":\"coredns\",\"image\":\"quay.io/cybozu/coredns:1.6.2.1\",\"args\":[\"-conf\",\"/etc/coredns/Corefile\"],\"ports\":[{\"name\":\"dns\",\"containerPort\":1053,\"protocol\":\"UDP\"},{\"name\":\"dns-tcp\",\"containerPort\":1053,\"protocol\":\"TCP\"}],\"resources\":{\"limits\":{\"memory\":\"170Mi\"},\"requests\":{\"cpu\":\"100m\",\"memory\":\"70Mi\"}},\"volumeMounts\":[{\"name\":\"config-volume\",\"readOnly\":true,\"mountPath\":\"/etc/coredns\"}],\"livenessProbe\":{\"httpGet\":{\"path\":\"/health\",\"port\":8080,\"scheme\":\"HTTP\"},\"initialDelaySeconds\":60,\"timeoutSeconds\":5,\"successThreshold\":1,\"failureThreshold\":5},\"readinessProbe\":{\"httpGet\":{\"path\":\"/health\",\"port\":8080,\"scheme\":\"HTTP\"},\"timeoutSeconds\":5},\"imagePullPolicy\":\"IfNotPresent\",\"securityContext\":{\"capabilities\":{\"drop\":[\"all\"]},\"readOnlyRootFilesystem\":true,\"allowPrivilegeEscalation\":false}}],\"dnsPolicy\":\"Default\",\"serviceAccountName\":\"cke-cluster-dns\",\"tolerations\":[{\"key\":\"node-role.kubernetes.io/master\",\"effect\":\"NoSchedule\"},{\"key\":\"CriticalAddonsOnly\",\"operator\":\"Exists\"}],\"priorityClassName\":\"system-cluster-critical\"}},\"strategy\":{\"type\":\"RollingUpdate\",\"rollingUpdate\":{\"maxUnavailable\":1}}},\"status\":{}}\n"),
	},
	{
		Key:        "DaemonSet/kube-system/node-dns",
		Kind:       "DaemonSet",
		Namespace:  "kube-system",
		Name:       "node-dns",
		Revision:   1,
		Image:      "quay.io/cybozu/unbound:1.9.2.1",
		Definition: []byte("{\"kind\":\"DaemonSet\",\"apiVersion\":\"apps/v1\",\"metadata\":{\"name\":\"node-dns\",\"namespace\":\"kube-system\",\"creationTimestamp\":null,\"annotations\":{\"cke.cybozu.com/image\":\"quay.io/cybozu/unbound:1.9.2.1\",\"cke.cybozu.com/revision\":\"1\"}},\"spec\":{\"selector\":{\"matchLabels\":{\"cke.cybozu.com/appname\":\"node-dns\"}},\"template\":{\"metadata\":{\"creationTimestamp\":null,\"labels\":{\"cke.cybozu.com/appname\":\"node-dns\"}},\"spec\":{\"volumes\":[{\"name\":\"config-volume\",\"configMap\":{\"name\":\"node-dns\",\"items\":[{\"key\":\"unbound.conf\",\"path\":\"unbound.conf\"}]}}],\"containers\":[{\"name\":\"unbound\",\"image\":\"quay.io/cybozu/unbound:1.9.2.1\",\"args\":[\"-c\",\"/etc/unbound/unbound.conf\"],\"resources\":{},\"volumeMounts\":[{\"name\":\"config-volume\",\"mountPath\":\"/etc/unbound\"}],\"livenessProbe\":{\"tcpSocket\":{\"port\":53,\"host\":\"localhost\"},\"initialDelaySeconds\":1,\"periodSeconds\":1,\"failureThreshold\":6},\"readinessProbe\":{\"tcpSocket\":{\"port\":53,\"host\":\"localhost\"},\"periodSeconds\":1},\"securityContext\":{\"capabilities\":{\"add\":[\"NET_BIND_SERVICE\"],\"drop\":[\"all\"]},\"readOnlyRootFilesystem\":true,\"allowPrivilegeEscalation\":false}},{\"name\":\"reload\",\"image\":\"quay.io/cybozu/unbound:1.9.2.1\",\"command\":[\"/usr/local/bin/reload-unbound\"],\"resources\":{},\"volumeMounts\":[{\"name\":\"config-volume\",\"mountPath\":\"/etc/unbound\"}],\"securityContext\":{\"capabilities\":{\"drop\":[\"all\"]},\"readOnlyRootFilesystem\":true,\"allowPrivilegeEscalation\":false}}],\"terminationGracePeriodSeconds\":0,\"nodeSelector\":{\"kubernetes.io/os\":\"linux\"},\"serviceAccountName\":\"cke-node-dns\",\"hostNetwork\":true,\"tolerations\":[{\"operator\":\"Exists\",\"effect\":\"NoSchedule\"},{\"key\":\"CriticalAddonsOnly\",\"operator\":\"Exists\"},{\"operator\":\"Exists\",\"effect\":\"NoExecute\"}],\"priorityClassName\":\"system-node-critical\"}},\"updateStrategy\":{\"type\":\"RollingUpdate\",\"rollingUpdate\":{\"maxUnavailable\":1}}},\"status\":{\"currentNumberScheduled\":0,\"numberMisscheduled\":0,\"desiredNumberScheduled\":0,\"numberReady\":0}}\n"),
	},
	{
		Key:        "Service/kube-system/cluster-dns",
		Kind:       "Service",
		Namespace:  "kube-system",
		Name:       "cluster-dns",
		Revision:   1,
		Image:      "",
		Definition: []byte("{\"kind\":\"Service\",\"apiVersion\":\"v1\",\"metadata\":{\"name\":\"cluster-dns\",\"namespace\":\"kube-system\",\"creationTimestamp\":null,\"labels\":{\"cke.cybozu.com/appname\":\"cluster-dns\"},\"annotations\":{\"cke.cybozu.com/revision\":\"1\"}},\"spec\":{\"ports\":[{\"name\":\"dns\",\"protocol\":\"UDP\",\"port\":53,\"targetPort\":1053},{\"name\":\"dns-tcp\",\"protocol\":\"TCP\",\"port\":53,\"targetPort\":1053}],\"selector\":{\"cke.cybozu.com/appname\":\"cluster-dns\"}},\"status\":{\"loadBalancer\":{}}}\n"),
	},
	{
		Key:        "PodDisruptionBudget/kube-system/cluster-dns-pdb",
		Kind:       "PodDisruptionBudget",
		Namespace:  "kube-system",
		Name:       "cluster-dns-pdb",
		Revision:   1,
		Image:      "",
		Definition: []byte("{\"kind\":\"PodDisruptionBudget\",\"apiVersion\":\"policy/v1beta1\",\"metadata\":{\"name\":\"cluster-dns-pdb\",\"namespace\":\"kube-system\",\"creationTimestamp\":null,\"annotations\":{\"cke.cybozu.com/revision\":\"1\"}},\"spec\":{\"selector\":{\"matchLabels\":{\"cke.cybozu.com/appname\":\"cluster-dns\"}},\"maxUnavailable\":1},\"status\":{\"disruptionsAllowed\":0,\"currentHealthy\":0,\"desiredHealthy\":0,\"expectedPods\":0}}\n"),
	},
}
