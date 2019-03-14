User-defined resources
======================

CKE can automatically create or update user-defined resources on Kubernetes.

This can be considered as `kubectl apply` automatically executed by CKE.

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

- `cke.cybozu.com/revision`: The last applied revision of this resource.
- `cke.cybozu.com/last-applied-configuration`: The last applied resource definition in JSON.

## Usage

Use `ckecli resource` subcommand to set, list, or delete user-defined resources.
