Cluster configuration
=====================

CKE deploys and maintains a Kubernetes cluster and an etcd cluster solely for
the Kubernetes cluster.  The configurations of the clusters can be defined by
a YAML or JSON object with these fields:

Name            | Required | Type      | Description
--------------- | -------- | --------- | -----------
`name`          | true     | string    | The k8s cluster name.
`nodes`         | true     | array     | `Node` list.
`ssh_key`       | false    | string    | Cluster wide SSH private key.
`service_subnet`| true     | string    | CIDR subnet for k8s `Service`.
`pod_subnet`    | true     | string    | CIDR subnet for k8s `Pod`.
`dns_servers`   | false    | array     | List of upstream DNS server IP addresses.
`options`       | false    | `Options` | See options.

* IP addresses in `pod_subnet` are only used for host-local communication
  as a fallback CNI plugin.  They are never seen from outside of the cluster.

Node
----

A `Node` has these fields:

Name            | Required | Type      | Description
--------------- | -------- | --------- | -----------
`address`       | true     | string    | IP address of the node.
`hostname`      | false    | string    | Override the real hostname of the node in k8s.
`user`          | true     | string    | SSH user name.
`ssh_key`       | false    | string    | SSH private key of the user.
`control_plane` | false    | bool      | If true, the node will be used for k8s control plane and etcd.
`annotations`   | false    | object    | Node annotations.
`labels`        | false    | object    | Node labels.
`taints`        | false    | `[]Taint` | Node taints.

`annotations`, `labels`, and `taints` are added or updated, but not removed.
This is because other applications may edit their own annotations, labels, or taints.

Taint
-----

Name     | Required | Type   | Description
-------- | -------- | ------ | -----------
`key`    | true     | string | The taint key to be applied to a node.
`value`  | false    | string | The taint value corresponding to the taint key.
`effect` | true     | string | The effect of the taint on pods that do not tolerate the taint. Valid effects are `NoSchedule`, `PreferNoSchedule` and `NoExecute`.

Options
-------

`Option` is a set of optional parameters for k8s components.

Name              | Required | Type            | Description
----------------- | -------- | --------------- | -----------
`etcd`            | false    | `EtcdParams`    | Extra arguments for etcd.
`rivers`          | false    | `ServiceParams` | Extra arguments for Rivers.
`kube-api`        | false    | `ServiceParams` | Extra arguments for API server.
`kube-controller` | false    | `ServiceParams` | Extra arguments for controller manager.
`kube-scheduler`  | false    | `ServiceParams` | Extra arguments for scheduler.
`kube-proxy`      | false    | `ServiceParams` | Extra arguments for kube-proxy.
`kubelet`         | false    | `KubeletParams` | Extra arguments for kubelet.

### ServiceParams

Name              | Required | Type   | Description
----------------- | -------- | ------ | -----------
`extra_args`      | false    | array  | Extra command-line arguments.  List of strings.
`extra_binds`     | false    | array  | Extra bind mounts.  List of `Mount`.
`extra_env`       | false    | object | Extra environment variables.

### Mount

Name              | Required | Type   | Description
----------------- | -------- | ------ | -----------
`source`          | true     | string | Path in a host to a directory or a file.
`destination`     | true     | string | Path in the container filesystem.
`read_only`       | false    | bool   | True to mount the directory or file as read-only.
`propagation`     | false    | string | Whether mounts can be propagated to replicas.
`selinux_label`   | false    | string | Relabel the SELinux label of the host directory.

`selinux-label`:
- "z":  The mount content is shared among multiple containers.
- "Z":  The mount content is private and unshared.
- This label should not be specified to system directories.

### EtcdParams

Name              | Required | Type   | Description
----------------- | -------- | ------ | -----------
`volume_name`     | false    | string | Docker volume name for data. Default: `etcd-cke`.
`extra_args`      | false    | array  | Extra command-line arguments.  List of strings.
`extra_binds`     | false    | array  | Extra bind mounts.  List of `Mount`.
`extra_env`       | false    | object | Extra environment variables.

### KubeletParams

Name              | Required | Type      | Description
----------------- | -------- | --------- | -----------
`domain`          | false    | string    | The base domain for the cluster.  Default: `cluster.local`.
`allow_swap`      | false    | bool      | Do not fail even when swap is on.
`boot_taints`     | false    | `[]Taint` | Bootstrap node taints.
`extra_args`      | false    | array     | Extra command-line arguments.  List of strings.
`extra_binds`     | false    | array     | Extra bind mounts.  List of `Mount`.
`extra_env`       | false    | object    | Extra environment variables.

Taints in `boot_taints` are added to a Node in the following cases:
(1) when that Node is registered with Kubernetes by kubelet, or
(2) when the kubelet on that Node is being booted while the Node resource is already registered.
Those taints can be removed manually when they are no longer needed.
Note that the second case happens when the physical node is rebooted without resource manipulation.
If you want to add taints only at Node registration, use kubelet's `--register-with-taints` options in `extra_args`.
