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

Kubernetes resources
--------------------

CKE installs and maintains following Kubernetes resources.

### RBAC roles

`system:kube-apiserver-to-kubelet` is a [ClusterRole](https://kubernetes.io/docs/reference/access-authn-authz/rbac/#role-and-clusterrole) to authorize API servers by kubelet.

### Etcd endpoints

`cke-etcd` is a headless [Service](https://kubernetes.io/docs/concepts/services-networking/service/) and [Endpoints](https://kubernetes.io/docs/concepts/services-networking/service/#services-without-selectors) to help applications find endpoints of CKE maintained etcd cluster.

Unchangeable features
---------------------

* [CNI][] is enabled in `kubelet`.
* [The latest standard CNI plugins][CNI plugins] are installed and available.
* [RBAC][] is enabled.
* [CoreDNS][] is installed.
* [PodSecurity][] is enabled.
* `kube-proxy` runs in IPVS mode.

[rivers]: https://github.com/cybozu-go/cke-tools/tree/master/cmd/rivers
[RBAC]: https://kubernetes.io/docs/reference/access-authn-authz/rbac/
[CoreDNS]: https://github.com/coredns/coredns
[CNI]: https://github.com/containernetworking/cni
[CNI plugins]: https://github.com/containernetworking/plugins
[PodSecurity]: https://kubernetes.io/docs/concepts/policy/pod-security-policy/
