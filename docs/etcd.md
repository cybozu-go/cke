Etcd management
===============

This document describes how CKE bootstraps and manage life-cycle of the members for etcd cluster.
CKE bootstraps etcd cluster, and manage cluster members from the CKE cluster configurations, automatically.

Cluster bootstrap
-----------------

CKE first checks if a control plane node has etcd volumes, in order to identify whether the etcd cluster has been bootstrapped.
The etcd cluster bootstrap only when there are no control plane nodes with etcd volumes.
CKE does not do etcd bootstrap, when at-least one control plane nodes have etcd volumes. 

CKE does the following steps to bootstrap etcd cluster.

1. Pull etcd images on each control plane node.
2. Create volume on control plane nodes.
3. Starts etcd container on each node.  The initial cluster consist of the control plane nodes.

Scale-up and scale-down cluster
-------------------------------

When replacing an etcd node, it's important to remove the member first and then add its replacement ([etcd FAQ]()).
CKE also first remove non-healthy members, then add newly member nodes to the cluster.
Here *healthy* means that CKE can reach to the node's endpoint, and the node has no errors returned by via [Status API](Status API).

Note that CKE *does not* remove non-healthy nodes described as control plane from the cluster.
That situation can occurs when the node is on booting OS, temporary network unreachable, or other reasons.
If non-healthy status cause by failure on the machine or networks, they must be detected by external monitor and user (or external service such as sabakan integration) must be remove it.

CKE does the following steps on each iteration:

1. Remove unhealthy and non-cluster from etcd cluster members.
2. Remove unhealthy and non-control-plane from etcd cluster members. Then CKE stops etcd container.
3. Start etcd container on unstarted etcd member nodes.
4. Add new control-plane into etcd cluster member. Then CKE starts etcd container on the nodes.
5. Remove healthy and non-cluster member from etcd cluster members.
6. Remove non-control-plane member from etcd cluster members. Then CKE stops etcd container.

CKE processes nodes if applicable nodes are exists on each step, then re-evaluate the cluster from step 1.

<!-- TODO Version control -->

[etcd FAQ]: https://coreos.com/etcd/docs/latest/faq.html
[Status API]: https://godoc.org/github.com/coreos/etcd/clientv3#Maintenance
