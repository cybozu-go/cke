package main

import (
	"fmt"
	"os"
	"text/template"
)

var tmpl = template.Must(template.New("").Parse(`// Code generated by apply_gen.go. DO NOT EDIT.
//go:generate go run ./pkg/apply_gen

package cke

import (
	"strconv"

	"github.com/cybozu-go/log"
	appsv1 "k8s.io/api/apps/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
)

func annotate(meta *metav1.ObjectMeta, rev int64) {
	if meta.Annotations == nil {
		meta.Annotations = make(map[string]string)
	}
	meta.Annotations[AnnotationResourceRevision] = strconv.FormatInt(rev, 10)
}
{{- range . }}

func apply{{ .Kind }}(o *{{ .API }}.{{ .Kind }}, rev int64, client rest.Interface, isNamespaced bool) error {
	annotate(&o.ObjectMeta, rev)
	modified, err := encodeToJSON(o)
	if err != nil {
		return err
	}

	_, err = client.
		Patch(types.ApplyPatchType).
		Resource("{{ .Resource }}").
		Name(o.Name).
		NamespaceIfScoped(o.Namespace, isNamespaced).
		Param("force", "true").
		Param("fieldManager", "cke").
		Body(modified).
		Do().
		Get()
	if err != nil {
		log.Error("failed to apply patch", map[string]interface{}{
			"kind":      o.Kind,
			"namespace": o.Namespace,
			"name":      o.Name,
			log.FnError: err,
		})
		return err
	}

	return nil
}
{{- end }}
`))

func main() {
	err := subMain()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func subMain() error {
	f, err := os.OpenFile("resource_apply.go", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	err = tmpl.Execute(f, []struct {
		API             string
		Kind            string
		Resource        string
		GracefulSeconds int64
	}{
		{"corev1", "Namespace", "namespaces", 60},
		{"corev1", "ServiceAccount", "serviceaccounts", 0},
		{"corev1", "ConfigMap", "configmaps", 0},
		{"corev1", "Service", "services", 0},
		{"policyv1beta1", "PodSecurityPolicy", "podsecuritypolicies", 0},
		{"networkingv1", "NetworkPolicy", "networkpolicies", 0},
		{"rbacv1", "Role", "roles", 0},
		{"rbacv1", "RoleBinding", "rolebindings", 0},
		{"rbacv1", "ClusterRole", "clusterroles", 0},
		{"rbacv1", "ClusterRoleBinding", "clusterrolebindings", 0},
		{"appsv1", "Deployment", "deployments", 60},
		{"appsv1", "DaemonSet", "daemonsets", 60},
		{"batchv1beta1", "CronJob", "cronjobs", 60},
		{"policyv1beta1", "PodDisruptionBudget", "poddisruptionbudgets", 0},
	})
	if err != nil {
		return err
	}
	return f.Sync()
}
