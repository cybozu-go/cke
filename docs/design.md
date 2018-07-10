Notes on CKE design
===================

Scope and Non-scope
-------------------

CKE should cover:

* Management of Kubernetes and etcd for Kubernetes including:

    * Bootstrapping from scratch.
    * Boot from fully stopped status.
    * Adding/Removing nodes.
    * Upgrading Kubernetes and etcd.
    * HA control planes.
    * Host-local DNS cache service.

* Deployment of add-ons:

    * CoreDNS
    * Custom add-ons by using custom manifests

* Integration with sabakan

    * Sabakan integration *must not* be requisite.

CKE should **NOT** cover:

* Asset provisioning

    * Assets such as `kubelet` are compiled into Docker images.  
      The images should be able to be pulled from Internet or
      pre-loaded by users.

* Management of Ceph or MySQL clusters

    * They should be managed by other tools.

Implementation policies
-----------------------

* Use etcd to:

    * persist data,
    * share data between multiple instances for high-availability,
    * and choose a leader instance that controls the cluster.

* Support only a single version of Kubernetes

    * A CKE version corresponds to a specific version of Kubernetes.
    * Upgrading from older CKE releases should anyway be supported.

* Cluster configuration can be supplied externally as YAML or JSON data.

* Users can define constraints on cluster configuration, for example:

    * the exact number of control plane nodes,
    * the minimum and maximum number of worker nodes,
    * etc.

* Sabakan integration, if enabled, generates cluster configuration that
    satisfies given constraints.

* CKE should periodically check the cluster status and compare it with
    the given configuration.  If anything is different, `cke` will updates
    the cluster.

* Assets are compiled into Docker images.

    * Third-party docker images should be mirrored on `quay.io/cybozu`.

* CKE does not install any tools onto node OS other than containers.

    * `kubelet` or other system services run by `docker run --restart=always`.

* CKE does not depend on specific network add-ons.

    * All its containers should run with `docker run --network=host`.
    * Users can install network add-ons by custom manifests.

* CKE uses `docker` as a container runtime of Kubernetes

    * Other CRI-conforming runtimes will be supported by allowing users
      to customize `kubelet` arguments.

* `cke` and `ckecli` does not communicate directly; they communicate through etcd.

How it works
------------

`cke` is a system service that watches Kubernetes cluster and configuration
changes in etcd.  If it detects differences between the cluster and configuration,
it updates Kubernetes cluster as follows.

1. If the instance has been elected as a leader, go forward.  Otherwise, do nothing.
2. Prepare a single operation for k8s to resolve a difference, and record it on etcd.
3. Update the operation record with the command to be executed.
4. Execute a single command for the operation.
5. Update the operation record with the result of the command.
6. Repeat 3, 4, 5 until the operation completes.
7. Update the operation record to mark as completed.
8. If there are no more differences, done.  Otherwise, go to 1.

For example, if the current Kubernetes cluster has the following differences from
the desired configuration:

* A control plane node exists only in the configuration.
* Two worker nodes is running in the cluster but not defined in the configuration.

`cke` updates the cluster with these three operations.
Note that each operation may invoke several commands.

1. Configure a control plane.
2. Remove an extra worker node.
3. Remove another extra worker node.

Handling failures
-----------------

### Leader death

`cke` leader may suddenly die while it is executing an operation.
In this case, another instance of `cke` will be elected as a new leader.

The new leader first checks that the last operation has completed by examining
the last operation record.  It the last operation has been completed, the new
leader works normally.

If the last operation has *not* been completed, the new leader need to kill
the last command, then update the operation record as canceled.

### Command failure

Commands may fail for miscellaneous reasons.  If a command for an operation
fails, `cke` simply cancels the operation.  Next time it checks the configuration
and the cluster, the situation may or may not change, who knows.

Sabakan integration
-------------------

Sabakan integration is an optional feature.  If enabled, `cke` periodically
queries sabakan to obtain list of machines.  The search query can be configured
by users.

`cke` then creates or updates the cluster configuration using the list of
available machines and given constraints, and stores the new cluster
configuration in etcd.

### Control plane node

When `cke` chooses a node for a control plane, it avoids nodes in the same
racks of other control plane nodes.

### Node labels

Generated cluster configuration will automatically label nodes with
[properties of sabakan machine struct][machine]

### Health check

`cke` chooses *healthy* nodes only.
The status of a node is given as `status` in [machine struct][machine].

Node selection can be further tuned by giving the shortest period of
healthy state as a constraint.

[machine]: https://github.com/cybozu-go/sabakan/blob/master/docs/machine.md#machine-struct
