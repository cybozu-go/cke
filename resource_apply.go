// Code generated by apply_gen.go. DO NOT EDIT.
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

func applyNamespace(o *corev1.Namespace, rev int64, client rest.Interface, isNamespaced bool) error {
	annotate(&o.ObjectMeta, rev)
	modified, err := encodeToJSON(o)
	if err != nil {
		return err
	}

	_, err = client.
		Patch(types.ApplyPatchType).
		Resource("namespaces").
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

func applyServiceAccount(o *corev1.ServiceAccount, rev int64, client rest.Interface, isNamespaced bool) error {
	annotate(&o.ObjectMeta, rev)
	modified, err := encodeToJSON(o)
	if err != nil {
		return err
	}

	_, err = client.
		Patch(types.ApplyPatchType).
		Resource("serviceaccounts").
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

func applyConfigMap(o *corev1.ConfigMap, rev int64, client rest.Interface, isNamespaced bool) error {
	annotate(&o.ObjectMeta, rev)
	modified, err := encodeToJSON(o)
	if err != nil {
		return err
	}

	_, err = client.
		Patch(types.ApplyPatchType).
		Resource("configmaps").
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

func applyService(o *corev1.Service, rev int64, client rest.Interface, isNamespaced bool) error {
	annotate(&o.ObjectMeta, rev)
	modified, err := encodeToJSON(o)
	if err != nil {
		return err
	}

	_, err = client.
		Patch(types.ApplyPatchType).
		Resource("services").
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

func applyPodSecurityPolicy(o *policyv1beta1.PodSecurityPolicy, rev int64, client rest.Interface, isNamespaced bool) error {
	annotate(&o.ObjectMeta, rev)
	modified, err := encodeToJSON(o)
	if err != nil {
		return err
	}

	_, err = client.
		Patch(types.ApplyPatchType).
		Resource("podsecuritypolicies").
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

func applyNetworkPolicy(o *networkingv1.NetworkPolicy, rev int64, client rest.Interface, isNamespaced bool) error {
	annotate(&o.ObjectMeta, rev)
	modified, err := encodeToJSON(o)
	if err != nil {
		return err
	}

	_, err = client.
		Patch(types.ApplyPatchType).
		Resource("networkpolicies").
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

func applyRole(o *rbacv1.Role, rev int64, client rest.Interface, isNamespaced bool) error {
	annotate(&o.ObjectMeta, rev)
	modified, err := encodeToJSON(o)
	if err != nil {
		return err
	}

	_, err = client.
		Patch(types.ApplyPatchType).
		Resource("roles").
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

func applyRoleBinding(o *rbacv1.RoleBinding, rev int64, client rest.Interface, isNamespaced bool) error {
	annotate(&o.ObjectMeta, rev)
	modified, err := encodeToJSON(o)
	if err != nil {
		return err
	}

	_, err = client.
		Patch(types.ApplyPatchType).
		Resource("rolebindings").
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

func applyClusterRole(o *rbacv1.ClusterRole, rev int64, client rest.Interface, isNamespaced bool) error {
	annotate(&o.ObjectMeta, rev)
	modified, err := encodeToJSON(o)
	if err != nil {
		return err
	}

	_, err = client.
		Patch(types.ApplyPatchType).
		Resource("clusterroles").
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

func applyClusterRoleBinding(o *rbacv1.ClusterRoleBinding, rev int64, client rest.Interface, isNamespaced bool) error {
	annotate(&o.ObjectMeta, rev)
	modified, err := encodeToJSON(o)
	if err != nil {
		return err
	}

	_, err = client.
		Patch(types.ApplyPatchType).
		Resource("clusterrolebindings").
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

func applyDeployment(o *appsv1.Deployment, rev int64, client rest.Interface, isNamespaced bool) error {
	annotate(&o.ObjectMeta, rev)
	modified, err := encodeToJSON(o)
	if err != nil {
		return err
	}

	_, err = client.
		Patch(types.ApplyPatchType).
		Resource("deployments").
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

func applyDaemonSet(o *appsv1.DaemonSet, rev int64, client rest.Interface, isNamespaced bool) error {
	annotate(&o.ObjectMeta, rev)
	modified, err := encodeToJSON(o)
	if err != nil {
		return err
	}

	_, err = client.
		Patch(types.ApplyPatchType).
		Resource("daemonsets").
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

func applyCronJob(o *batchv1beta1.CronJob, rev int64, client rest.Interface, isNamespaced bool) error {
	annotate(&o.ObjectMeta, rev)
	modified, err := encodeToJSON(o)
	if err != nil {
		return err
	}

	_, err = client.
		Patch(types.ApplyPatchType).
		Resource("cronjobs").
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

func applyPodDisruptionBudget(o *policyv1beta1.PodDisruptionBudget, rev int64, client rest.Interface, isNamespaced bool) error {
	annotate(&o.ObjectMeta, rev)
	modified, err := encodeToJSON(o)
	if err != nil {
		return err
	}

	_, err = client.
		Patch(types.ApplyPatchType).
		Resource("poddisruptionbudgets").
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
