Kubernetes specifications
=========================

This document describes the specifications and features of
a Kubernetes cluster bootstrapped by CKE.

Cluster bootstrap
-----------------

After CKE has deployed an [etcd cluster](etcd.md), CKE bootstraps a kubernetes cluster.
Control plane applications of kubernetes cluster (apiserver, controller-manager and scheduler)
do not deployed as self-hosted.
These applications are deployed and maintained by CKE for simplicity.

In order to construct HA control plane, CKE deploys a [reverse proxy][] daemon to each nodes.
Reverse proxy is responsible for forwarding to apiservers on the control plane nodes.
The controller-manager, schedulers, and kubelets refers localhost's reverse proxy.

CKE does the following steps to bootstrap kubernetes cluster.

1. Launch reverse proxy on all nodes.
1. Launch apiservers on each control plane node.
1. Launch controller-manager on each control plane node.
1. Launch scheduler on each control plane node.
1. Launch kubelet on all nodes.

Install kubernetes applications
-------------------------------

<!-- TODO -->

After kubernetes bootstrapped, CKE deploys the following applications:

- `kube-proxy` as a DaemonSet
- CoreDNS

Unchangeable features
---------------------

* [CoreDNS][] is installed.
* [PodSecurity][] is enabled.
* `kube-proxy` runs in IPVS mode.
* CNI is enabled.

[CoreDNS]: https://github.com/coredns/coredns
[PodSecurity]: https://kubernetes.io/docs/concepts/policy/pod-security-policy/
[RBAC]: https://kubernetes.io/docs/reference/access-authn-authz/rbac/
[Reverse Proxy]: https://github.com/cybozu-go/cke-tools/tree/master/cmd/rivers
