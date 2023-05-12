package cke

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"

	"github.com/cybozu-go/log"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
)

// Annotations for CKE-managed resources.
const (
	AnnotationResourceImage     = "cke.cybozu.com/image"
	AnnotationResourceRevision  = "cke.cybozu.com/revision"
	AnnotationResourceInjectCA  = "cke.cybozu.com/inject-cacert"
	AnnotationResourceIssueCert = "cke.cybozu.com/issue-cert"
	AnnotationResourceRank      = "cke.cybozu.com/rank"
)

// kinds
const (
	KindDeployment                     = "Deployment"
	KindMutatingWebhookConfiguration   = "MutatingWebhookConfiguration"
	KindSecret                         = "Secret"
	KindValidatingWebhookConfiguration = "ValidatingWebhookConfiguration"
)

// rank
const (
	RankNamespace                      = 10
	RankServiceAccount                 = 20
	RankCustomResourceDefinition       = 30
	RankClusterRole                    = 40
	RankClusterRoleBinding             = 50
	RankClusterScopedResourceDefault   = 1000
	RankRole                           = 2000
	RankRoleBinding                    = 2010
	RankNetworkPolicy                  = 2020
	RankSecret                         = 2030
	RankConfigMap                      = 2040
	RankNamespaceScopedResourceDefault = 3000
)

var decUnstructured = yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)

// ApplyResource creates or updates given resource using server-side-apply.
func ApplyResource(ctx context.Context, dynclient dynamic.Interface, mapper meta.RESTMapper, inf Infrastructure, data []byte, rev int64, forceConflicts bool) error {
	obj := &unstructured.Unstructured{}
	_, gvk, err := decUnstructured.Decode(data, nil, obj)
	if err != nil {
		return fmt.Errorf("failed to decode data into *Unstructured: %w", err)
	}
	ann := obj.GetAnnotations()
	if ann == nil {
		ann = make(map[string]string)
	}
	ann[AnnotationResourceRevision] = strconv.FormatInt(rev, 10)
	obj.SetAnnotations(ann)

	mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return fmt.Errorf("failed to find REST mapping for %s: %w", gvk.String(), err)
	}

	if gvk.Kind == KindValidatingWebhookConfiguration || gvk.Kind == KindMutatingWebhookConfiguration {
		if ann[AnnotationResourceInjectCA] == "true" {
			if err := injectCA(ctx, inf.Storage(), obj, gvk); err != nil {
				return fmt.Errorf("failed to inject CA certificate: %w", err)
			}
		}
	}

	if gvk.Kind == KindSecret {
		if svc := ann[AnnotationResourceIssueCert]; svc != "" {
			if err := issueCert(ctx, inf, obj, svc); err != nil {
				return fmt.Errorf("failed to issue cert for webhook: %w", err)
			}
		}
	}

	var dr dynamic.ResourceInterface
	if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
		dr = dynclient.Resource(mapping.Resource).Namespace(obj.GetNamespace())
	} else {
		dr = dynclient.Resource(mapping.Resource)
	}
	buf := &bytes.Buffer{}
	if err := unstructured.UnstructuredJSONScheme.Encode(obj, buf); err != nil {
		return err
	}
	if log.Enabled(log.LvDebug) {
		log.Debug("resource-apply", map[string]interface{}{
			"gvk":       gvk.String(),
			"gvr":       mapping.Resource.String(),
			"namespace": obj.GetNamespace(),
			"name":      obj.GetName(),
			"data":      buf.String(),
		})
	}

	_, err = dr.Patch(ctx, obj.GetName(), types.ApplyPatchType, buf.Bytes(), metav1.PatchOptions{
		FieldManager: "cke",
		Force:        &forceConflicts,
	})
	return err
}

func injectCA(ctx context.Context, st Storage, obj *unstructured.Unstructured, gvk *schema.GroupVersionKind) error {
	cacert, err := st.GetCACertificate(ctx, CAWebhook)
	if err != nil {
		return fmt.Errorf("failed to get CA cert for webhook: %w", err)
	}
	certData := []byte(cacert)

	cvt := runtime.DefaultUnstructuredConverter
	switch {
	case gvk.Version == "v1" && gvk.Kind == KindValidatingWebhookConfiguration:
		typed := &admissionregistrationv1.ValidatingWebhookConfiguration{}
		if err := cvt.FromUnstructured(obj.UnstructuredContent(), typed); err != nil {
			return fmt.Errorf("failed to convert validating webhook %s: %w", obj.GetName(), err)
		}
		for i := range typed.Webhooks {
			typed.Webhooks[i].ClientConfig.CABundle = certData
		}
		mutated, err := cvt.ToUnstructured(typed)
		if err != nil {
			return err
		}
		obj.UnstructuredContent()["webhooks"] = mutated["webhooks"]
		return nil

	case gvk.Version == "v1" && gvk.Kind == KindMutatingWebhookConfiguration:
		typed := &admissionregistrationv1.MutatingWebhookConfiguration{}
		if err := cvt.FromUnstructured(obj.UnstructuredContent(), typed); err != nil {
			return fmt.Errorf("failed to convert mutating webhook %s: %w", obj.GetName(), err)
		}
		for i := range typed.Webhooks {
			typed.Webhooks[i].ClientConfig.CABundle = certData
		}
		mutated, err := cvt.ToUnstructured(typed)
		if err != nil {
			return err
		}
		obj.UnstructuredContent()["webhooks"] = mutated["webhooks"]
		return nil

	case gvk.Version == "v1beta1" && gvk.Kind == KindValidatingWebhookConfiguration:
		typed := &admissionregistrationv1beta1.ValidatingWebhookConfiguration{}
		if err := cvt.FromUnstructured(obj.UnstructuredContent(), typed); err != nil {
			return fmt.Errorf("failed to convert validating webhook %s: %w", obj.GetName(), err)
		}
		for i := range typed.Webhooks {
			typed.Webhooks[i].ClientConfig.CABundle = certData
		}
		mutated, err := cvt.ToUnstructured(typed)
		if err != nil {
			return err
		}
		obj.UnstructuredContent()["webhooks"] = mutated["webhooks"]
		return nil

	case gvk.Version == "v1beta1" && gvk.Kind == KindMutatingWebhookConfiguration:
		typed := &admissionregistrationv1beta1.MutatingWebhookConfiguration{}
		if err := cvt.FromUnstructured(obj.UnstructuredContent(), typed); err != nil {
			return fmt.Errorf("failed to convert mutating webhook %s: %w", obj.GetName(), err)
		}
		for i := range typed.Webhooks {
			typed.Webhooks[i].ClientConfig.CABundle = certData
		}
		mutated, err := cvt.ToUnstructured(typed)
		if err != nil {
			return err
		}
		obj.UnstructuredContent()["webhooks"] = mutated["webhooks"]
		return nil
	}

	panic(gvk)
}

func issueCert(ctx context.Context, inf Infrastructure, obj *unstructured.Unstructured, svcName string) error {
	cvt := runtime.DefaultUnstructuredConverter
	secret := &corev1.Secret{}
	if err := cvt.FromUnstructured(obj.UnstructuredContent(), secret); err != nil {
		return fmt.Errorf("failed to convert secret %s/%s: %w", obj.GetNamespace(), obj.GetName(), err)
	}

	ca := WebhookCA{}
	cert, key, err := ca.IssueCertificate(ctx, inf, obj.GetNamespace(), svcName)
	if err != nil {
		return fmt.Errorf("failed to issue certificate for %s.%s.svc: %w", svcName, obj.GetNamespace(), err)
	}
	if secret.Data == nil {
		secret.Data = make(map[string][]byte)
	}
	secret.Data[corev1.TLSCertKey] = []byte(cert)
	secret.Data[corev1.TLSPrivateKeyKey] = []byte(key)
	mutated, err := cvt.ToUnstructured(secret)
	if err != nil {
		return err
	}

	obj.UnstructuredContent()["data"] = mutated["data"]
	obj.UnstructuredContent()["type"] = corev1.SecretTypeTLS
	return nil
}

// ParseResource parses YAML string.
func ParseResource(data []byte) (string, error) {
	obj := &unstructured.Unstructured{}
	_, _, err := decUnstructured.Decode(data, nil, obj)
	if err != nil {
		return "", err
	}

	name := obj.GetName()
	if name == "" {
		return "", errors.New("no name")
	}

	if obj.GetAPIVersion() == "" {
		return "", errors.New("no apiVersion")
	}

	if obj.GetNamespace() == "" {
		return obj.GetKind() + "/" + name, nil
	}
	return obj.GetKind() + "/" + obj.GetNamespace() + "/" + name, nil
}

func DecideRank(kind, namespace string, rank uint32) (uint32, error) {
	if rank > 0 {
		if namespace == "" && rank < 2000 {
			return rank, nil
		}
		if namespace != "" && rank > 1999 {
			return rank, nil
		}
		return 0, errors.New("invalid rank value")
	}
	// return default rank
	switch kind {
	case "Namespace":
		return RankNamespace, nil
	case "ServiceAccount":
		return RankServiceAccount, nil
	case "CustomResourceDefinition":
		return RankCustomResourceDefinition, nil
	case "ClusterRole":
		return RankClusterRole, nil
	case "ClusterRoleBinding":
		return RankClusterRoleBinding, nil
		// other cluster-scoped resources: 1000
	case "Role":
		return RankRole, nil
	case "RoleBinding":
		return RankRoleBinding, nil
	case "NetworkPolicy":
		return RankNetworkPolicy, nil
	case "Secret":
		return RankSecret, nil
	case "ConfigMap":
		return RankConfigMap, nil
		// other namespace scoped resources: 3000
	}

	if namespace == "" {
		return RankClusterScopedResourceDefault, nil
	}
	return RankNamespaceScopedResourceDefault, nil
}

// ResourceDefinition represents a CKE-managed kubernetes resource.
type ResourceDefinition struct {
	Key        string
	Kind       string
	Namespace  string
	Name       string
	Revision   int64
	Image      string // may contains multiple images; we should not use this whole string as an image name.
	Rank       uint32
	Definition []byte
}

// String implements fmt.Stringer.
func (d ResourceDefinition) String() string {
	return fmt.Sprintf("%s@%d", d.Key, d.Revision)
}

// NeedUpdate returns true if annotations of the current resource
// indicates need for update.
func (d ResourceDefinition) NeedUpdate(rs *ResourceStatus) bool {
	if rs == nil {
		return true
	}
	curRev, ok := rs.Annotations[AnnotationResourceRevision]
	if !ok {
		return true
	}
	if curRev != strconv.FormatInt(d.Revision, 10) {
		return true
	}

	if d.Image == "" {
		return false
	}

	curImage, ok := rs.Annotations[AnnotationResourceImage]
	if !ok {
		return true
	}
	return curImage != d.Image
}

// SortResources sort resources as defined order of creation.
func SortResources(res []ResourceDefinition) {
	less := func(i, j int) bool {
		a := res[i]
		b := res[j]
		aRank := a.Rank
		bRank := b.Rank

		if aRank == bRank {
			return a.Key < b.Key
		}
		return aRank < bRank
	}

	sort.Slice(res, less)
}
