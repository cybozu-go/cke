User-defined resources
======================

CKE can automatically create or update user-defined resources on Kubernetes.
This can be considered as `kubectl apply --server-side=true --field-manager=cke` automatically executed by CKE.

## Supported resources

All the standard Kubernetes resources, including `CustomResourceDefinition`, are supported.

Custom resources (not `CustomResourceDefinition`s) are not supported by default,
but can be managed by pre-registering their REST mappings as `trusted_rest_mappings`
in the [cluster configuration](cluster.md#trustedrestmapping).

## Order of application

The resources are applied in the following order according to their kind.

- Namespace
- ServiceAccount
- CustomResourceDefinition
- ClusterRole
- ClusterRoleBinding
- (Other cluster-scope resources)
- Role
- RoleBinding
- NetworkPolicy
- Secret
- ConfigMap
- (Other namespace-scoped resources)

## Annotations

User-defined resources are automatically annotated as follows:

- `cke.cybozu.com/revision`: The last applied revision of this resource.

### Annotations for admission webhooks

By annotating ValidatingWebhookConfiguration or MutatingWebhookConfiguration
with `cke.cybozu.com/inject-cacert=true`, CKE automatically fill it with CA
certificates.

By annotating Secret with `cke.cybozu.com/issue-cert=<service name>`, CKE
automatically issues a new certificate for the named `Service` resource and
sets the certificate and private key in Secret data.

Read [k8s.md](k8s.md#certificates-for-admission-webhooks) for more details.

## Usage

Use `ckecli resource` subcommand to set, list, or delete user-defined resources.
