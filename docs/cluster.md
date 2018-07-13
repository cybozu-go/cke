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
`dns_servers`   | false    | array     | List of upstream DNS server IP addresses.
`options`       | false    | `Options` | See options.
`rbac`          | false    | bool      | true to enable [RBAC][].

Node
----

A `Node` has these fields:

Name            | Required | Type   | Description
--------------- | -------- | ------ | -----------
`address`       | true     | string | IP address of the node.
`hostname`      | false    | string | Override the real hostname of the node in k8s.
`user`          | true     | string | SSH user name.
`ssh_key`       | false    | string | SSH private key of the user.
`control_plane` | false    | bool   | If true, the node will be used for k8s control plane and etcd.
`labels`        | false    | object | Node labels for k8s.

Options
-------

`Option` is a set of optional parameters for k8s components.

Name              | Required | Type            | Description
----------------- | -------- | --------------- | -----------
`etcd`            | false    | `ServiceParams` | Extra arguments for etcd.
`kube-api`        | false    | `ServiceParams` | Extra arguments for API server.
`kube-controller` | false    | `ServiceParams` | Extra arguments for controller manager.
`kube-scheduler`  | false    | `ServiceParams` | Extra arguments for scheduler.
`kube-proxy`      | false    | `ServiceParams` | Extra arguments for kube-proxy.
`kubelet`         | false    | `KubeletParams` | Extra arguments for kubelet.
`coredns`         | false    | `ServiceParams` | Extra arguments for [CoreDNS][].

### ServiceParams

Name              | Required | Type   | Description
----------------- | -------- | ------ | -----------
`extra_args`      | false    | object | Extra command-line arguments.
`extra_binds`     | false    | object | Extra bind mounts.
`extra_env`       | false    | object | Extra environment variables.

### KubeletParams

Name              | Required | Type   | Description
----------------- | -------- | ------ | -----------
`domain`          | false    | string | The base domain for the cluster.  Default: `cluster.local`.
`allow_swap`      | false    | bool   | Do not fail even when swap is on.
`extra_args`      | false    | object | Extra command-line arguments.
`extra_binds`     | false    | object | Extra bind mounts.
`extra_env`       | false    | object | Extra environment variables.

[CoreDNS]: https://github.com/coredns/coredns
[RBAC]: https://kubernetes.io/docs/reference/access-authn-authz/rbac/
