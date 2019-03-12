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
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/client-go/kubernetes/scheme"
)

// ParseResource parses YAML string.
func ParseResource(data []byte) (key string, jsonData []byte, obj runtime.Object, err error) {
	decode := scheme.Codecs.UniversalDeserializer().Decode
	obj, gvk, err := decode(data, nil, nil)
	if err != nil {
		return "", nil, nil, err
	}

	switch o := obj.(type) {
	case *corev1.Namespace:
		data, err := json.Marshal(o)
		return o.Kind + "/" + o.Name, data, obj, err
	case *corev1.ServiceAccount:
		data, err := json.Marshal(o)
		return o.Kind + "/" + o.Namespace + "/" + o.Name, data, obj, err
	case *corev1.ConfigMap:
		data, err := json.Marshal(o)
		return o.Kind + "/" + o.Namespace + "/" + o.Name, data, obj, err
	case *corev1.Service:
		data, err := json.Marshal(o)
		return o.Kind + "/" + o.Namespace + "/" + o.Name, data, obj, err
	case *policyv1beta1.PodSecurityPolicy:
		data, err := json.Marshal(o)
		return o.Kind + "/" + o.Name, data, obj, err
	case *networkingv1.NetworkPolicy:
		data, err := json.Marshal(o)
		return o.Kind + "/" + o.Namespace + "/" + o.Name, data, obj, err
	case *rbacv1.Role:
		data, err := json.Marshal(o)
		return o.Kind + "/" + o.Namespace + "/" + o.Name, data, obj, err
	case *rbacv1.RoleBinding:
		data, err := json.Marshal(o)
		return o.Kind + "/" + o.Namespace + "/" + o.Name, data, obj, err
	case *rbacv1.ClusterRole:
		data, err := json.Marshal(o)
		return o.Kind + "/" + o.Name, data, obj, err
	case *rbacv1.ClusterRoleBinding:
		data, err := json.Marshal(o)
		return o.Kind + "/" + o.Name, data, obj, err
	case *appsv1.Deployment:
		data, err := json.Marshal(o)
		return o.Kind + "/" + o.Namespace + "/" + o.Name, data, obj, err
	case *appsv1.DaemonSet:
		data, err := json.Marshal(o)
		return o.Kind + "/" + o.Namespace + "/" + o.Name, data, obj, err
	case *batchv2alpha1.CronJob:
		data, err := json.Marshal(o)
		return o.Kind + "/" + o.Namespace + "/" + o.Name, data, obj, err
	}

	return "", nil, obj, fmt.Errorf("unsupported type: %s", gvk.String())
}

// CreateResourceDiff calculate strategic merge patch between two resources.
func CreateResourceDiff(original, modified []byte, obj runtime.Object) ([]byte, error) {
	return strategicpatch.CreateTwoWayMergePatch(original, modified, obj)
}
