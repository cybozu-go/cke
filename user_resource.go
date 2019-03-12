package cke

import (
	"encoding/json"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	batchv2alpha1 "k8s.io/api/batch/v2alpha1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/client-go/kubernetes/scheme"
)

// Kind represents resource types in Kubernetes.
type Kind string

// List of supported Kinds
const (
	KindNamespace          = Kind("Namespace")
	KindServiceAccount     = Kind("ServiceAccount")
	KindPodSecurityPolicy  = Kind("PodSecurityPolicy")
	KindNetworkPolicy      = Kind("NetworkPolicy")
	KindRole               = Kind("Role")
	KindRoleBinding        = Kind("RoleBinding")
	KindClusterRole        = Kind("ClusterRole")
	KindClusterRoleBinding = Kind("ClusterRoleBinding")
	KindConfigMap          = Kind("ConfigMap")
	KindDeployment         = Kind("Deployment")
	KindDaemonSet          = Kind("DaemonSet")
	KindCronJob            = Kind("CronJob")
	KindService            = Kind("Service")
)

// ParseResource parses YAML string.
func ParseResource(data []byte) (key string, jsonData []byte, err error) {
	decode := scheme.Codecs.UniversalDeserializer().Decode
	obj, gvk, err := decode(data, nil, nil)
	if err != nil {
		return "", nil, err
	}

	switch o := obj.(type) {
	case *corev1.Namespace:
		data, err := json.Marshal(o)
		return o.Kind + "/" + o.Name, data, err
	case *corev1.ServiceAccount:
		fmt.Println("this is service account")
	case *corev1.ConfigMap:
		fmt.Println("this is config map")
	case *corev1.Service:
		fmt.Println("this is service")
	case *policyv1beta1.PodSecurityPolicy:
		fmt.Println("this is pod security policy")
	case *networkingv1.NetworkPolicy:
		fmt.Println("this is network policy")
	case *rbacv1.Role:
		fmt.Println("this is role")
	case *rbacv1.RoleBinding:
		fmt.Println("this is role binding")
	case *rbacv1.ClusterRole:
		fmt.Println("this is cluster role")
	case *rbacv1.ClusterRoleBinding:
		fmt.Println("this is cluster role binding")
	case *appsv1.Deployment:
		fmt.Println("this is deployment")
	case *appsv1.DaemonSet:
		fmt.Println("this is daemonset")
	case *batchv2alpha1.CronJob:
		fmt.Println("this is cron job")
	}

	return "", nil, fmt.Errorf("unsupported type: %s", gvk.String())
}
