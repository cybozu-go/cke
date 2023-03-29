User-defined resources
======================

CKE can automatically create or update user-defined resources on Kubernetes.
This can be considered as `kubectl apply --server-side=true --field-manager=cke` automatically executed by CKE.

## Supported resources

All the standard Kubernetes resources, including `CustomResourceDefinition`, are supported.

Custom resources (not `CustomResourceDefinition`s) are not supported.

## Order of application

The resources are applied in the following order according to their kind.

- Namespace
  - rank: 10
- ServiceAccount
  - rank: 20
- CustomResourceDefinition
  - rank: 30
- ClusterRole
  - rank: 40
- ClusterRoleBinding
  - rank: 50
- (Other cluster-scope resources)
  - rank: 1000
- Role
  - rank: 2000
- RoleBinding
  - rank: 2010
- NetworkPolicy
  - rank: 2020
- Secret
  - rank: 2030
- ConfigMap
  - rank: 2040
- (Other namespace-scoped resources)
  - rank: 3000

### Custom order

Users can control the order of applying resources by annotating `cke.cybozu.com/rank`.
In the case of cluster-scope resources, a rank value must be 0 ~ 1999.
For namespace-scope resources, a rank value must be 2000 ~.

If `cke.cybozu.com/rank` is not set, the rank is assigned a value based on the abovementioned list.

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
