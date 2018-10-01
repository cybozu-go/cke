Kubernetes specifications
=========================

This document describes the specifications and features of
a Kubernetes cluster bootstrapped by CKE.

High availability
-----------------

Every node in Kubernetes cluster runs a TCP reverse proxy called [rivers][]
for load-balancing requests to API servers.  It will implicitly retry
connection attempts when some API servers are down.

Kubernetes resources
--------------------

CKE installs and maintains following Kubernetes resources:

- `system:kube-apiserver-to-kubelet` [RBAC][] role to authorize API servers by kubelet.
- (TODO) [CoreDNS][]

Unchangeable features
---------------------

* [CNI][] is enabled for kubelet.
* [RBAC][] is enabled.
* [CoreDNS][] is installed.
* [PodSecurity][] is enabled.
* `kube-proxy` runs in IPVS mode.
* CNI is enabled.

[rivers]: https://github.com/cybozu-go/cke-tools/tree/master/cmd/rivers
[RBAC]: https://kubernetes.io/docs/reference/access-authn-authz/rbac/
[CoreDNS]: https://github.com/coredns/coredns
[CNI]: https://github.com/containernetworking/cni
[PodSecurity]: https://kubernetes.io/docs/concepts/policy/pod-security-policy/
