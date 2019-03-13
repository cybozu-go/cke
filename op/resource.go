package op

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/log"
	appsv1 "k8s.io/api/apps/v1"
	batchv2alpha1 "k8s.io/api/batch/v2alpha1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type resourceCreateOp struct {
	apiserver *cke.Node
	resource  cke.ResourceDefinition
	finished  bool
}

// ResourceCreateOp creates a new resource.
func ResourceCreateOp(apiServer *cke.Node, resource cke.ResourceDefinition) cke.Operator {
	return &resourceCreateOp{
		apiserver: apiServer,
		resource:  resource,
	}
}

func (o *resourceCreateOp) Name() string {
	return "resource-create"
}

func (o *resourceCreateOp) NextCommand() cke.Commander {
	if o.finished {
		return nil
	}
	o.finished = true
	return o
}

func (o *resourceCreateOp) Run(ctx context.Context, inf cke.Infrastructure) error {
	cs, err := inf.K8sClient(ctx, o.apiserver)
	if err != nil {
		return err
	}

	switch o.resource.Kind {
	case "Namespace":
		var obj corev1.Namespace
		err := json.Unmarshal(o.resource.Definition, &obj)
		if err != nil {
			return err
		}
		setAnnotations(&obj.ObjectMeta, o.resource)
		_, err = cs.CoreV1().Namespaces().Create(&obj)
		if err != nil {
			return err
		}
	case "ServiceAccount":
		var obj corev1.ServiceAccount
		err := json.Unmarshal(o.resource.Definition, &obj)
		if err != nil {
			return err
		}
		setAnnotations(&obj.ObjectMeta, o.resource)
		_, err = cs.CoreV1().ServiceAccounts(o.resource.Namespace).Create(&obj)
		if err != nil {
			return err
		}
	case "ConfigMap":
		var obj corev1.ConfigMap
		err := json.Unmarshal(o.resource.Definition, &obj)
		if err != nil {
			return err
		}
		setAnnotations(&obj.ObjectMeta, o.resource)
		_, err = cs.CoreV1().ConfigMaps(o.resource.Namespace).Create(&obj)
		if err != nil {
			return err
		}
	case "Service":
		var obj corev1.Service
		err := json.Unmarshal(o.resource.Definition, &obj)
		if err != nil {
			return err
		}
		setAnnotations(&obj.ObjectMeta, o.resource)
		_, err = cs.CoreV1().Services(o.resource.Namespace).Create(&obj)
		if err != nil {
			return err
		}
	case "PodSecurityPolicy":
		var obj policyv1beta1.PodSecurityPolicy
		err := json.Unmarshal(o.resource.Definition, &obj)
		if err != nil {
			return err
		}
		setAnnotations(&obj.ObjectMeta, o.resource)
		_, err = cs.PolicyV1beta1().PodSecurityPolicies().Create(&obj)
		if err != nil {
			return err
		}
	case "NetworkPolicy":
		var obj networkingv1.NetworkPolicy
		err := json.Unmarshal(o.resource.Definition, &obj)
		if err != nil {
			return err
		}
		setAnnotations(&obj.ObjectMeta, o.resource)
		_, err = cs.NetworkingV1().NetworkPolicies(o.resource.Namespace).Create(&obj)
		if err != nil {
			return err
		}
	case "Role":
		var obj rbacv1.Role
		err := json.Unmarshal(o.resource.Definition, &obj)
		if err != nil {
			return err
		}
		setAnnotations(&obj.ObjectMeta, o.resource)
		_, err = cs.RbacV1().Roles(o.resource.Namespace).Create(&obj)
		if err != nil {
			return err
		}
	case "RoleBinding":
		var obj rbacv1.RoleBinding
		err := json.Unmarshal(o.resource.Definition, &obj)
		if err != nil {
			return err
		}
		setAnnotations(&obj.ObjectMeta, o.resource)
		_, err = cs.RbacV1().RoleBindings(o.resource.Namespace).Create(&obj)
		if err != nil {
			return err
		}
	case "ClusterRole":
		var obj rbacv1.ClusterRole
		err := json.Unmarshal(o.resource.Definition, &obj)
		if err != nil {
			return err
		}
		setAnnotations(&obj.ObjectMeta, o.resource)
		_, err = cs.RbacV1().ClusterRoles().Create(&obj)
		if err != nil {
			return err
		}
	case "ClusterRoleBinding":
		var obj rbacv1.ClusterRoleBinding
		err := json.Unmarshal(o.resource.Definition, &obj)
		if err != nil {
			return err
		}
		setAnnotations(&obj.ObjectMeta, o.resource)
		_, err = cs.RbacV1().ClusterRoleBindings().Create(&obj)
		if err != nil {
			return err
		}
	case "Deployment":
		var obj appsv1.Deployment
		err := json.Unmarshal(o.resource.Definition, &obj)
		if err != nil {
			return err
		}
		setAnnotations(&obj.ObjectMeta, o.resource)
		_, err = cs.AppsV1().Deployments(o.resource.Namespace).Create(&obj)
		if err != nil {
			return err
		}
	case "DaemonSet":
		var obj appsv1.DaemonSet
		err := json.Unmarshal(o.resource.Definition, &obj)
		if err != nil {
			return err
		}
		setAnnotations(&obj.ObjectMeta, o.resource)
		_, err = cs.AppsV1().DaemonSets(o.resource.Namespace).Create(&obj)
		if err != nil {
			return err
		}
	case "CronJob":
		var obj batchv2alpha1.CronJob
		err := json.Unmarshal(o.resource.Definition, &obj)
		if err != nil {
			return err
		}
		setAnnotations(&obj.ObjectMeta, o.resource)
		_, err = cs.BatchV2alpha1().CronJobs(o.resource.Namespace).Create(&obj)
		if err != nil {
			return err
		}
	default:
		log.Warn("unknown resource kind", map[string]interface{}{
			"kind": o.resource.Kind,
		})
	}

	return nil
}

func setAnnotations(meta *metav1.ObjectMeta, resource cke.ResourceDefinition) {
	if meta.Annotations == nil {
		meta.Annotations = make(map[string]string)
	}
	meta.Annotations[cke.AnnotationResourceRevision] = strconv.FormatInt(resource.Revision, 10)
	meta.Annotations[cke.AnnotationResourceOriginal] = string(resource.Definition)
}

func (o *resourceCreateOp) Command() cke.Command {
	return cke.Command{
		Name:   "create-resource",
		Target: o.resource.String(),
	}
}

// ResourcePatchOp patches a resource using 3-way strategic merge patch.
func ResourcePatchOp(apiServer *cke.Node, resource cke.ResourceDefinition) cke.Operator {
	return nil
}
