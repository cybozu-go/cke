Kubernetes specifications
=========================

This document describes the specifications and features of
a Kubernetes cluster bootstrapped by CKE.

Unchangeable features
---------------------

* [CoreDNS][] is installed.
* [PodSecurity][] is enabled.
* `kube-proxy` runs in IPVS mode.
* CNI is enabled.

[CoreDNS]: https://github.com/coredns/coredns
[PodSecurity]: https://kubernetes.io/docs/concepts/policy/pod-security-policy/
[RBAC]: https://kubernetes.io/docs/reference/access-authn-authz/rbac/
