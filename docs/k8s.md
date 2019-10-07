Kubernetes specifications
=========================

This document describes the specifications and features of
a Kubernetes cluster bootstrapped by CKE.

High availability
-----------------

Every node in Kubernetes cluster runs a TCP reverse proxy called [rivers][]
for load-balancing requests to API servers.  It will implicitly retry
connection attempts when some API servers are down.

Control plane
-------------

In CKE, the control plane of Kubernetes consists of Nodes.  To isolate
control plane nodes from others, CKE automatically adds `cke.cybozu.com/master: "true"` label.

If `taint_control_plane` is true in `cluster.yml`, CKE taints control
plane nodes with `cke.cybozu.com/master: PreferNoSchedule`.

Node maintenance
----------------

CKE simply removes a `Node` resource from Kubernetes API server when the
corresponding entry in `cluster.yml` disappears.  The administrator is
responsible to safely drain nodes in advance by using `kubectl drain` or
`kubectl taint nodes`.

* [Safely Drain a Node while Respecting Application SLOs](https://kubernetes.io/docs/tasks/administer-cluster/safely-drain-node/)
* [Taints and Tolerations](https://kubernetes.io/docs/concepts/configuration/taint-and-toleration/)

DNS resolution
--------------

CKE deploys [unbound][] DNS server on each node by DaemonSet.
This DNS server is called "node-local DNS cache server"; at its name shown, pods send DNS
queries to a node-local DNS cache server running on the same node.

CKE also deploys [CoreDNS][] as Deployment.  Node-local DNS cache servers are configured to
send queries for Kubernetes domain names such as `kubernetes.default.svc.cluster.local` to
CoreDNS.

For other domain names such as `www.google.com`, node-local DNS cache servers can be
configured to send queries to upstream DNS servers defined in [cluster.yml](./cluster.md).

Data encryption at rest
-----------------------

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

Kubernetes resources
--------------------

CKE installs and maintains following Kubernetes resources other than DNS ones.

### Pod security policies

Though CKE does not enable [PodSecurityPolicy][] by default, it prepares necessary policies
for resources managed by CKE to be ready for enabling [PodSecurityPolicy][].

### Service accounts

* `cke-node-dns` in `kube-system` is the service account for node-local DNS cache servers.
* `cke-cluster-dns` in `kube-system` is the service account for CoreDNS.

### RBAC roles

`system:kube-apiserver-to-kubelet` is a [ClusterRole](https://kubernetes.io/docs/reference/access-authn-authz/rbac/#role-and-clusterrole) to authorize API servers by kubelet.

`system:kube-apiserver` is a [ClusterRoleBinding](https://kubernetes.io/docs/reference/access-authn-authz/rbac/#rolebinding-and-clusterrolebinding) to bind `system:kube-apiserver-to-kubelet` to user `kubernetes`.  Note that API server is authenticated as `kubernetes` user.

`psp:node-dns` is a Role in `kube-system` to associate pod security policy with `node-dns` service account.

`psp:cluster-dns` is a Role in `kube-system` to associate pod security policy with `cluster-dns` service account.

### Kubernetes endpoints

`kubernetes` endpoint object in `default` namespace represents the endpoints of the API servers.
CKE maintains this endpoint object on behalf of the API servers.

### Etcd endpoints

`cke-etcd` in `kube-system` namespace is a headless [Service](https://kubernetes.io/docs/concepts/services-networking/service/) and [Endpoints](https://kubernetes.io/docs/concepts/services-networking/service/#services-without-selectors) to help applications find endpoints of CKE maintained etcd cluster.

Unchangeable features
---------------------

* [CNI][] is enabled in `kubelet`.
* [The latest standard CNI plugins][CNI plugins] are installed and available.
* [RBAC][] is enabled.
* [CoreDNS][] is installed.
* [Secret][] data at rest are encrypted.
* [Aggregation layer](https://kubernetes.io/docs/tasks/access-kubernetes-api/configure-aggregation-layer/) is configured.

Default settings
----------------

* `kube-proxy` runs in IPVS mode.
* [PodSecurityPolicy][] is not enabled.

[rivers]: https://github.com/cybozu/neco-containers/tree/master/cke-tools/src/cmd/rivers
[unbound]: https://www.nlnetlabs.nl/projects/unbound/
[etcd]: http://etcd.io/
[RBAC]: https://kubernetes.io/docs/reference/access-authn-authz/rbac/
[CoreDNS]: https://github.com/coredns/coredns
[Secret]: https://kubernetes.io/docs/concepts/configuration/secret/
[CNI]: https://github.com/containernetworking/cni
[CNI plugins]: https://github.com/containernetworking/plugins
[PodSecurityPolicy]: https://kubernetes.io/docs/concepts/policy/pod-security-policy/
