Kubernetes specifications
=========================

This document describes the specifications and features of
a Kubernetes cluster bootstrapped by CKE.

- [High availability](#high-availability)
- [Control plane](#control-plane)
- [Node lifecycle](#node-lifecycle)
- [DNS resolution](#dns-resolution)
- [Certificates for admission webhooks](#certificates-for-admission-webhooks)
- [Data encryption at rest](#data-encryption-at-rest)
  - [Rationale for not using `kms`](#rationale-for-not-using-kms)
- [Pre-installed Kubernetes resources](#pre-installed-kubernetes-resources)
  - [Service accounts](#service-accounts)
  - [RBAC roles](#rbac-roles)
  - [Kubernetes Endpoints](#kubernetes-endpoints)
  - [Etcd Endpoints](#etcd-endpoints)
- [Unchangeable features](#unchangeable-features)
- [Default settings](#default-settings)

## High availability

Every node in Kubernetes cluster runs a TCP reverse proxy called [rivers](../tools/rivers)
for load-balancing requests to API servers.  It will implicitly retry
connection attempts when some API servers are down.

## Control plane

In CKE, the control plane of Kubernetes consists of Nodes.  To isolate
control plane nodes from others, CKE automatically adds `cke.cybozu.com/master: "true"` label.

If `taint_control_plane` is true in `cluster.yml`, CKE taints control
plane nodes with `cke.cybozu.com/master: PreferNoSchedule`.

## Node lifecycle

CKE simply removes a `Node` resource from Kubernetes API server when the
corresponding entry in `cluster.yml` disappears.  The administrator is
responsible to safely drain nodes in advance by using `kubectl drain` or
`kubectl taint nodes`.

- [Safely Drain a Node while Respecting Application SLOs](https://kubernetes.io/docs/tasks/administer-cluster/safely-drain-node/)
- [Taints and Tolerations](https://kubernetes.io/docs/concepts/configuration/taint-and-toleration/)

## DNS resolution

CKE deploys [unbound][] DNS server on each node by DaemonSet.
This DNS server is called "node-local DNS cache server"; at its name shown, pods send DNS
queries to a node-local DNS cache server running on the same node.

CKE also deploys [CoreDNS][] as Deployment.  Node-local DNS cache servers are configured to
send queries for Kubernetes domain names such as `kubernetes.default.svc.cluster.local` to
CoreDNS.

For other domain names such as `www.google.com`, node-local DNS cache servers can be
configured to send queries to upstream DNS servers defined in [cluster.yml](./cluster.md).
CKE validates the integrity of the replies using DNSSEC validation.

## Certificates for admission webhooks

[Admission webhooks][webhook] are extensions of Kubernetes to validate or mutate API resources.
To run webhooks, self-signed certificates need to be issued.

CKE can work as a certificate authority for these webhook servers and issue certificates.

To embed CA certificate in [ValidatingWebhookConfiguration][] or [MutatingWebhookConfiguration][],
define a webhook configuration annotated with `cke.cybozu.com/inject-cacert=true` as
a [user-defined resource](user-resources.md) like:

```yaml
apiVersion: admissionregistration.k8s.io/v1
kind: ValidatingWebhookConfiguration
metadata:
  name: "example"
  annotations:
    cke.cybozu.com/inject-cacert: "true"
webhooks:
  ...
```

To issue certificates, define a [Secret][] annotated with `cke.cybozu.com/issue-cert=<service name>`
as a [user-defined resource](user-resources.md) like:

```yaml
apiVersion: v1
kind: Secret
metadata:
  namespace: example
  name: webhook-cert
  annotations:
    cke.cybozu.com/issue-cert: webhook-service
type: kubernetes.io/tls
```

## Data encryption at rest

Kubernetes can encrypt data at rest, i.e. data stored in [etcd][].
For details, take a look at [Encrypting Secret Data at Rest](https://kubernetes.io/docs/tasks/administer-cluster/encrypt-data/).

CKE automatically encrypts [Secret][] resource data.  The encryption key is generated and
stored in Vault.  The secret provider is currently `aescbc`.  `kms` provider is not used
because it does not add extra security compared to other providers.

### Rationale for not using `kms`

`kms` provider delegates encryption key management to a remote key-management service (KMS).
However, `kms` provider itself connects only to a service that runs on the same host using
a UNIX domain socket.  The local service then connects to a remote KMS such as Vault.

This means that the local service has credentials to authenticate with remote KMS, and the
credentials need to be protected from malicious users / programs.

Other providers such as `aescbc` read an encryption key from a configuration file.
This file need to be protected just as same as KMS credentials.  There are no difference
with regard to security.

With these in mind, `kms` does not improve security but introduces an extra component.
As CKE can automatically generate and protect configuration files for `aescbs`, there
are no reasons to choose `kms`.

## Pre-installed Kubernetes resources

CKE installs and maintains following Kubernetes resources other than DNS ones.

### Service accounts

- `cke-node-dns` in `kube-system` is the service account for node-local DNS cache servers.
- `cke-cluster-dns` in `kube-system` is the service account for CoreDNS.

### RBAC roles

`system:kube-apiserver-to-kubelet` is a [ClusterRole](https://kubernetes.io/docs/reference/access-authn-authz/rbac/#role-and-clusterrole) to authorize API servers by kubelet.

`system:kube-apiserver` is a [ClusterRoleBinding](https://kubernetes.io/docs/reference/access-authn-authz/rbac/#rolebinding-and-clusterrolebinding) to bind `system:kube-apiserver-to-kubelet` to user `kubernetes`.  Note that API server is authenticated as `kubernetes` user.

`psp:node-dns` is a Role in `kube-system` to associate pod security policy with `node-dns` service account.

`psp:cluster-dns` is a Role in `kube-system` to associate pod security policy with `cluster-dns` service account.

### Kubernetes Endpoints

`kubernetes` Endpoints object and `kubernetes` EndpointSlice object, both in `default` namespace, represent the endpoints of the API servers.
CKE maintains these objects on behalf of the API servers.

### Etcd Endpoints

`cke-etcd` in `kube-system` namespace is a headless [Service](https://kubernetes.io/docs/concepts/services-networking/service/), [Endpoints](https://kubernetes.io/docs/concepts/services-networking/service/#services-without-selectors) and [EndpointSlice](https://kubernetes.io/docs/concepts/services-networking/endpoint-slices/) to help applications find endpoints of CKE maintained etcd cluster.

## Unchangeable features

- [CNI][] is enabled in `kubelet`.
- [The latest standard CNI plugins][CNI plugins] are installed and available.
- [RBAC][] is enabled.
- [CoreDNS][] is installed.
- [Secret][] data at rest are encrypted.
- [Aggregation layer](https://kubernetes.io/docs/tasks/access-kubernetes-api/configure-aggregation-layer/) is configured.

## Default settings

- `kube-apiserver` runs with coordinated leader election enabled.
- `kube-proxy` runs in IPVS mode.

[unbound]: https://www.nlnetlabs.nl/projects/unbound/
[webhook]: https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/
[ValidatingWebhookConfiguration]: https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#validatingwebhookconfiguration-v1-admissionregistration-k8s-io
[MutatingWebhookConfiguration]: https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#mutatingwebhookconfiguration-v1-admissionregistration-k8s-io
[Secret]: https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#secret-v1-core
[etcd]: http://etcd.io/
[RBAC]: https://kubernetes.io/docs/reference/access-authn-authz/rbac/
[CoreDNS]: https://github.com/coredns/coredns
[Secret]: https://kubernetes.io/docs/concepts/configuration/secret/
[CNI]: https://github.com/containernetworking/cni
[CNI plugins]: https://github.com/containernetworking/plugins
