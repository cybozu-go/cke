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

// Annotations for user-defined resources.
const (
	AnnotationResourceRevision = "cke.cybozu.com/revision"
	AnnotationResourceOriginal = "cke.cybozu.com/last-applied-configuration"
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
		data, err := json.Marshal(o)
		return o.Kind + "/" + o.Namespace + "/" + o.Name, data, err
	case *corev1.ConfigMap:
		data, err := json.Marshal(o)
		return o.Kind + "/" + o.Namespace + "/" + o.Name, data, err
	case *corev1.Service:
		data, err := json.Marshal(o)
		return o.Kind + "/" + o.Namespace + "/" + o.Name, data, err
	case *policyv1beta1.PodSecurityPolicy:
		data, err := json.Marshal(o)
		return o.Kind + "/" + o.Name, data, err
	case *networkingv1.NetworkPolicy:
		data, err := json.Marshal(o)
		return o.Kind + "/" + o.Namespace + "/" + o.Name, data, err
	case *rbacv1.Role:
		data, err := json.Marshal(o)
		return o.Kind + "/" + o.Namespace + "/" + o.Name, data, err
	case *rbacv1.RoleBinding:
		data, err := json.Marshal(o)
		return o.Kind + "/" + o.Namespace + "/" + o.Name, data, err
	case *rbacv1.ClusterRole:
		data, err := json.Marshal(o)
		return o.Kind + "/" + o.Name, data, err
	case *rbacv1.ClusterRoleBinding:
		data, err := json.Marshal(o)
		return o.Kind + "/" + o.Name, data, err
	case *appsv1.Deployment:
		data, err := json.Marshal(o)
		return o.Kind + "/" + o.Namespace + "/" + o.Name, data, err
	case *appsv1.DaemonSet:
		data, err := json.Marshal(o)
		return o.Kind + "/" + o.Namespace + "/" + o.Name, data, err
	case *batchv2alpha1.CronJob:
		data, err := json.Marshal(o)
		return o.Kind + "/" + o.Namespace + "/" + o.Name, data, err
	}

	return "", nil, fmt.Errorf("unsupported type: %s", gvk.String())
}

// ResourceDefinition represents a user-defined resource in etcd.
type ResourceDefinition struct {
	Kind       string
	Namespace  string
	Name       string
	Revision   int64
	Definition []byte
}

// String implements fmt.Stringer.
func (d ResourceDefinition) String() string {
	if len(d.Namespace) == 0 {
		return fmt.Sprintf("%s/%s@%d", d.Kind, d.Name, d.Revision)
	}
	return fmt.Sprintf("%s/%s/%s@%d", d.Kind, d.Namespace, d.Name, d.Revision)
}
