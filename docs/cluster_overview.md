Cluster Overview
================

How CKE works
-------------

![How CKE works](http://www.plantuml.com/plantuml/svg/PO-nQiGm38PtFOMWiuScwTAXf9HEXOxTLKi9eOvTRFdkzS--S11i3yRVzsEXVqvAKVFk88fLygiJ_FZwH4fe_mIVc9ToJk4FPQSrljG7C2dzKX8KjGnaDKHyvttpMz98bIWXLG7W0mj-rwku2i-z6derzchgrGj0NTZaV_Ds36zuQ7XiU6huCP33rPkZtecFzd1lXiR9ekLRoL_H1hziQuw2rkMa4c4MptbtDm00)

CKE constructs and maintains Kubernetes cluster from cluster configuration
defined by user (administrator).  CKE also deploys some components to nodes
and onto Kubernetes.

User defines two type of nodes to cluster configuration:

- Worker node: kubelet and kube-proxy
- Control plane node: Etcd, Kubernetes control plane components.

Worker node is a worker for Kubernetes cluster where containers are running.
Control plane node works as a master node for the cluster.  It also contains
etcd server used by kube-apiservers.  The number of the control plane nodes is
typically three or five.  Control plane node also has a feature of a worker
nodes.  Therefore, Kubernetes worker components (kubelet and kube-proxy) are
deployed to every node, and only a few nodes have control plane components.

Worker Nodes
------------

CKE deploys following components to worker nodes:

- kubelet
- kube-proxy
- rivers

CKE deploys *[rivers][rivers]* to all nodes to proxy kube-apiservers due to high
availability.  It works as a load balancer to the servers, and every Kubernetes
components connect to kube-apiservers via it
(see also [k8s.md](k8s.md#high-availability)).


![Worker Nodes](http://www.plantuml.com/plantuml/png/bP5FYuCm4CNl-HI3fzs31yVx8ko7sEEIwb2ACP4nwHzAltjD6a75XdfxlFVcyOEf1YlPkau9mLHRgO-A8Firsh9Hq2kfAGCvGFro_eDJxEZYZcufX3RDsFipt1A7mYN80ku2OBRKkWCfig4ITR5iz6okjv07vTElR-3JcNWOtQYyFTr3xlhyPnQ4mxNzU0k97q1Y4XAt8RqztIzeS89SsHuoyiPWzRz4YAcmZ26cPZ4rYzkpeYBTk4uz0G00)

Control Plane Nodes
-------------------

CKE deploys following components to control plane nodes:

- etcd
- etcd-rivers (works as a load balancer to etcd)
- kube-apiserver
- kube-scheduler
- kube-controller-manager
- revers (works as a load balancer to kube-apiserver)

CKE constructs etcd cluster before it construct Kubernetes cluster.  Then CKE
deploys Kubernetes components with rivers.

![Control Plane Nodes](http://www.plantuml.com/plantuml/svg/dP11RiCW44Ntd09brIuSYPbz5Qa7iQYDqaZOiG1tK_NkjJ4E43A9avs7__qOti4wQTpOQMPKusH_r8hlFi-zCsVD1orxjUFIycOvgVs9uB-CyrOw-INjL5UkQNrh_X1JbA3aSBBA_2ZZ2vVfgcMRRzMEEhJ2LBJ24bDGTRANnr2FnxK_NlvxUryMgynfkizUzgkNUQqaQGmOJtRWrJXK7p4jxoixyQ4Xog_-Oq_OXdksOPDjs6GRNhGDZsq3PHjOwXeoZt3JTTc9ponTmtEkyPvhtEGQDxd65rtZOzT8kSRCDMOUyQRhiXEVMRh6sVKy2xxV-m3y2Ek8Z2LraH045G0LO1e0XG8A1HGAAHHIACnGsQPHbw02e88L1HGAA1HGAAHGIA6mH1rKtuwT_WS0)

DNS
---

CKE deploys *[CoreDNS][]* as in-cluster DNS server to resolve names registered
by Kubernetes such as service name `xxx.default.svc.cluster.local`.  CKE also
deploys node-local DNS server to proxy CoreDNS, and each pod refer it as DNS
server.  Node-local DNS is responsible for caching names.  CKE deploys
*[unbound][]* as node-local DNS.  Node-local DNS also refer full resolver to
resolve domain from the internet.

Since CKE does not deploy full resolver on the cluster, you should deploy a
full resoluver by yourself, or set Public DNS such as `8.8.8.8` to `dns_servers`
in cluster config.

![DNS](http://www.plantuml.com/plantuml/svg/bPDDImCn48Rl-HN3djf32qq_3XwaK154A887Brwscq74TAOa6HMa_ztTR4BPfG7Tq-mpxtoyCDdwKBiWHwjKOraCL8zoG4SOq5Vmem0SDg6cDujGxQpuW0xkzi-lDDcnmpQQLb1xQFgK8JyikHThst_FzXDTLB8atLafOjDgNjXzfEHN31UZmKzikkI9pQB0TO4lMpwPGhLdWsbjeGCBcNvjQhaXlr0AOdkOoMbsUy6HwgjqEQPbFxhePrNWwmBV_CsFJdvMmnrrJzTNwMPCp_aa7YZ4Y-W6lATOgUmxbLqEu0Q-upTFQ6wvgMtMwt_gSt-MiPgFWvubZSeqYRA3eMYBPBfNy0i0)

[rivers]: https://github.com/cybozu/neco-containers/tree/master/cke-tools/src/cmd/rivers
[CoreDNS]: https://github.com/coredns/coredns
[unbound]: https://nlnetlabs.nl/projects/unbound/
