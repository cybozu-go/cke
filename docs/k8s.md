Kubernetes specifications
=========================

This document describes the specifications and features of
a Kubernetes cluster bootstrapped by CKE.

High availability
-----------------

Every node in Kubernetes cluster runs a TCP reverse proxy called [rivers][]
for load-balancing requests to API servers.  It will implicitly retry
connection attempts when some API servers are down.

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

Kubernetes resources
--------------------

CKE installs and maintains following Kubernetes resources other than DNS ones.

### Pod security policies

Though CKE does not enable [PodSecurityPolicy][] by default, it prepares necessary policies
for resources managed by CKE to be ready for enabling [PodSecurityPolicy][].

### Service accounts

* `node-dns` in `kube-system` is the service account for node-local DNS cache servers.
* `cluster-dns` in `kube-system` is the service account for CoreDNS.

### RBAC roles

`system:kube-apiserver-to-kubelet` is a [ClusterRole](https://kubernetes.io/docs/reference/access-authn-authz/rbac/#role-and-clusterrole) to authorize API servers by kubelet.

`system:kube-apiserver` is a [ClusterRoleBinding](https://kubernetes.io/docs/reference/access-authn-authz/rbac/#rolebinding-and-clusterrolebinding) to bind `system:kube-apiserver-to-kubelet` to user `kubernetes`.  Note that API server is authenticated as `kubernetes` user.

`psp:node-dns` is a Role in `kube-system` to associate pod security policy with `node-dns` service account.

`psp:cluster-dns` is a Role in `kube-system` to associate pod security policy with `cluster-dns` service account.

### Etcd endpoints

`cke-etcd` in `kube-system` namespace is a headless [Service](https://kubernetes.io/docs/concepts/services-networking/service/) and [Endpoints](https://kubernetes.io/docs/concepts/services-networking/service/#services-without-selectors) to help applications find endpoints of CKE maintained etcd cluster.

Unchangeable features
---------------------

* [CNI][] is enabled in `kubelet`.
* [The latest standard CNI plugins][CNI plugins] are installed and available.
* [RBAC][] is enabled.
* [CoreDNS][] is installed.

Default settings
----------------

* `kube-proxy` runs in IPVS mode.
* [PodSecurityPolicy][] is not enabled.

[rivers]: https://github.com/cybozu-go/cke-tools/tree/master/cmd/rivers
[unbound]: https://www.nlnetlabs.nl/projects/unbound/
[RBAC]: https://kubernetes.io/docs/reference/access-authn-authz/rbac/
[CoreDNS]: https://github.com/coredns/coredns
[CNI]: https://github.com/containernetworking/cni
[CNI plugins]: https://github.com/containernetworking/plugins
[PodSecurityPolicy]: https://kubernetes.io/docs/concepts/policy/pod-security-policy/
