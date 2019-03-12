User-defined resources
======================

CKE can automatically create or update user-defined resources on Kubernetes.

## Supported resource types

- Namespace
- ServiceAccount
- PodSecurityPolicy
- NetworkPolicy
- ClusterRole, Role
- ClusterRoleBinding, RoleBinding
- ConfigMap
- Deployment
- DaemonSet
- CronJob
- Service

Resources are created in the order of this list.

## Annotations

User-defined resources are annotated as follows:

- `cke.cybozu.com/user-resource`: `true`

## Usage

Use `ckecli resource` subcommand to set, list, or delete user-defined resources.
