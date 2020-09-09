Cluster configuration
=====================

CKE deploys and maintains a Kubernetes cluster and an etcd cluster solely for
the Kubernetes cluster.  The configurations of the clusters can be defined by
a YAML or JSON object with these fields:

- [Node](#node)
- [Taint](#taint)
- [EtcdBackup](#etcdbackup)
- [Options](#options)
  - [ServiceParams](#serviceparams)
  - [Mount](#mount)
  - [EtcdParams](#etcdparams)
  - [APIServerParams](#apiserverparams)
  - [KubeletParams](#kubeletparams)
  - [SchedulerParams](#schedulerparams)
  - [CNIConfFile](#cniconffile)

| Name                  | Required | Type         | Description                                                      |
| --------------------- | -------- | ------------ | ---------------------------------------------------------------- |
| `name`                | true     | string       | The k8s cluster name.                                            |
| `nodes`               | true     | array        | `Node` list.                                                     |
| `taint_control_plane` | false    | bool         | If true, taint contorl plane nodes.                              |
| `service_subnet`      | true     | string       | CIDR subnet for k8s `Service`.                                   |
| `dns_servers`         | false    | array        | List of upstream DNS server IP addresses.                        |
| `dns_service`         | false    | string       | Upstream DNS service name with namespace as `namespace/service`. |
| `etcd_backup`         | false    | `EtcdBackup` | See EtcdBackup.                                                  |
| `options`             | false    | `Options`    | See options.                                                     |

* Upstream DNS servers can be specified one of the following ways:
    * List server IP addresses in `dns_servers`.
    * Specify Kubernetes `Service` name in `dns_service` (e.g. `"kube-system/dns"`).  
      The service type must be `ClusterIP`.

Node
----

A `Node` has these fields:

| Name            | Required | Type      | Description                                                    |
| --------------- | -------- | --------- | -------------------------------------------------------------- |
| `address`       | true     | string    | IP address of the node.                                        |
| `hostname`      | false    | string    | Override the real hostname of the node in k8s.                 |
| `user`          | true     | string    | SSH user name.                                                 |
| `control_plane` | false    | bool      | If true, the node will be used for k8s control plane and etcd. |
| `annotations`   | false    | object    | Node annotations.                                              |
| `labels`        | false    | object    | Node labels.                                                   |
| `taints`        | false    | `[]Taint` | Node taints.                                                   |
|                 |

`annotations`, `labels`, and `taints` are added or updated, but not removed.
This is because other applications may edit their own annotations, labels, or taints.

Note that annotations, labels, and taints whose names contain `cke.cybozu.com/` or start with `node-role.kubernetes.io/` are
reserved for CKE internal usage, therefore should not be used.

Taint
-----

| Name     | Required | Type   | Description                                                                                                                         |
| -------- | -------- | ------ | ----------------------------------------------------------------------------------------------------------------------------------- |
| `key`    | true     | string | The taint key to be applied to a node.                                                                                              |
| `value`  | false    | string | The taint value corresponding to the taint key.                                                                                     |
| `effect` | true     | string | The effect of the taint on pods that do not tolerate the taint. Valid effects are `NoSchedule`, `PreferNoSchedule` and `NoExecute`. |

EtcdBackup
----------

| Name       | Required | Type   | Description                                                      |
| ---------- | -------- | ------ | ---------------------------------------------------------------- |
| `enabled`  | true     | bool   | If true, periodic etcd backup will be run.                       |
| `pvc_name` | true     | string | The name of `PersistentVolumeClaim` where backup data is stored. |
| `schedule` | true     | string | The schedule for etcd backup in Cron format.                     |
| `rotate`   | false    | int    | Keep a number of backup files. Default: 14.                      |

Options
-------

`Option` is a set of optional parameters for k8s components.

| Name                      | Required | Type              | Description                             |
| ------------------------- | -------- | ----------------- | --------------------------------------- |
| `etcd`                    | false    | `EtcdParams`      | Extra arguments for etcd.               |
| `etcd-rivers`             | false    | `ServiceParams`   | Extra arguments for EtcdRivers.         |
| `rivers`                  | false    | `ServiceParams`   | Extra arguments for Rivers.             |
| `kube-api`                | false    | `APIServerParams` | Extra arguments for API server.         |
| `kube-controller-manager` | false    | `ServiceParams`   | Extra arguments for controller manager. |
| `kube-scheduler`          | false    | `SchedulerParams` | Extra arguments for scheduler.          |
| `kube-proxy`              | false    | `ServiceParams`   | Extra arguments for kube-proxy.         |
| `kubelet`                 | false    | `KubeletParams`   | Extra arguments for kubelet.            |

### ServiceParams

| Name          | Required | Type   | Description                                     |
| ------------- | -------- | ------ | ----------------------------------------------- |
| `extra_args`  | false    | array  | Extra command-line arguments.  List of strings. |
| `extra_binds` | false    | array  | Extra bind mounts.  List of `Mount`.            |
| `extra_env`   | false    | object | Extra environment variables.                    |

### Mount

| Name            | Required | Type   | Description                                       |
| --------------- | -------- | ------ | ------------------------------------------------- |
| `source`        | true     | string | Path in a host to a directory or a file.          |
| `destination`   | true     | string | Path in the container filesystem.                 |
| `read_only`     | false    | bool   | True to mount the directory or file as read-only. |
| `propagation`   | false    | string | Whether mounts can be propagated to replicas.     |
| `selinux_label` | false    | string | Relabel the SELinux label of the host directory.  |

`selinux-label`:

- "z":  The mount content is shared among multiple containers.
- "Z":  The mount content is private and unshared.
- This label should not be specified to system directories.

### EtcdParams

| Name          | Required | Type   | Description                                       |
| ------------- | -------- | ------ | ------------------------------------------------- |
| `volume_name` | false    | string | Docker volume name for data. Default: `etcd-cke`. |
| `extra_args`  | false    | array  | Extra command-line arguments.  List of strings.   |
| `extra_binds` | false    | array  | Extra bind mounts.  List of `Mount`.              |
| `extra_env`   | false    | object | Extra environment variables.                      |

### APIServerParams

| Name                | Required | Type   | Description                                           |
| ------------------- | -------- | ------ | ----------------------------------------------------- |
| `extra_args`        | false    | array  | Extra command-line arguments.  List of strings.       |
| `extra_binds`       | false    | array  | Extra bind mounts.  List of `Mount`.                  |
| `extra_env`         | false    | object | Extra environment variables.                          |
| `audit_log_enabled` | false    | bool   | If true, audit log will be logged to standard output. |
| `audit_log_policy`  | false    | string | Audit policy configuration in yaml format.            |

### KubeletParams

| Name                         | Required | Type        | Description                                                                                                                                                         |
| ---------------------------- | -------- | ----------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `domain`                     | false    | string      | The base domain for the cluster.  Default: `cluster.local`.                                                                                                         |
| `allow_swap`                 | false    | bool        | Do not fail even when swap is on.                                                                                                                                   |
| `boot_taints`                | false    | `[]Taint`   | Bootstrap node taints.                                                                                                                                              |
| `extra_args`                 | false    | array       | Extra command-line arguments.  List of strings.                                                                                                                     |
| `extra_binds`                | false    | array       | Extra bind mounts.  List of `Mount`.                                                                                                                                |
| `extra_env`                  | false    | object      | Extra environment variables.                                                                                                                                        |
| `cgroup_driver`              | false    | string      | Driver that the kubelet uses to manipulate cgroups on the host. `cgroupfs` (default) or `systemd`.                                                                  |
| `container_runtime`          | false    | string      | Container runtime for Pod. Default: `docker`. You have to choose `docker` or `remote` which supports [CRI][].                                                       |
| `container_runtime_endpoint` | false    | string      | Path of the runtime socket. It is required when `container_runtime` is `remote`. Default: `/var/run/dockershim.sock`.                                               |
| `container_log_max_size`     | false    | string      | Equivalent to the [log rotation for CRI runtime]. Size of log file size. If the file size becomes bigger than given size, the log file is rotated. Default: `10Mi`. |
| `container_log_max_files`    | false    | int         | Equivalent to the [log rotation for CRI runtime]. Number of rotated log files for keeping in the storage. Default: `5`.                                             |
| `cni_conf_file`              | false    | CNIConfFile | CNI configuration file.                                                                                                                                             |

Taints in `boot_taints` are added to a Node in the following cases:
(1) when that Node is registered with Kubernetes by kubelet, or
(2) when the kubelet on that Node is being booted while the Node resource is already registered.
Those taints can be removed manually when they are no longer needed.
Note that the second case happens when the physical node is rebooted without resource manipulation.
If you want to add taints only at Node registration, use kubelet's `--register-with-taints` options in `extra_args`.

CNI configuration file specified by `cni_conf_file` will be put in `/etc/cni/net.d` directory
on all nodes.  The file is created only when `kubelet` starts on the node; it will *not* be
updated later on.

### SchedulerParams

| Name          | Required | Type       | Description                                     |
| ------------- | -------- | ---------- | ----------------------------------------------- |
| `extenders`   | false    | `[]string` | Extender parameters                             |
| `predicates`  | false    | `[]string` | Predicate parameters                            |
| `priorities`  | false    | `[]string` | Priority parameters                             |
| `extra_args`  | false    | array      | Extra command-line arguments.  List of strings. |
| `extra_binds` | false    | array      | Extra bind mounts.  List of `Mount`.            |
| `extra_env`   | false    | object     | Extra environment variables.                    |

Elements of `extenders`, `predicates` and `priorities` are contents of
[`Extender`](https://github.com/kubernetes/kube-scheduler/blob/release-1.18/config/v1/types.go#L190),
[`PredicatePolicy`](https://github.com/kubernetes/kube-scheduler/blob/release-1.18/config/v1/types.go#L50) and
[`PriorityPolicy`](https://github.com/kubernetes/kube-scheduler/blob/release-1.18/config/v1/types.go#L60)
in JSON format, respectively.

### CNIConfFile

| Name      | Required | Type   | Description                 |
| --------- | -------- | ------ | --------------------------- |
| `name`    | true     | string | file name                   |
| `content` | true     | string | file content in JSON format |

`name` is the filename of CNI configuration file.
It should end with either `.conf` or `.conflist`.


[CRI]: https://github.com/kubernetes/kubernetes/blob/242a97307b34076d5d8f5bbeb154fa4d97c9ef1d/docs/devel/container-runtime-interface.md
[log rotation for CRI runtime]: https://github.com/kubernetes/kubernetes/issues/58823
