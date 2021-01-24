Container Runtime support
=========================

CKE deployed containers
-----------------------

The following programs are run as Docker containers.

- `etcd`
- `kube-apiserver`
- `kube-controller-manager`
- `kube-scheduler`
- `kubelet`
- [rivers](../tools/rivers)

Kubernetes Pods
---------------

CKE has tested only with [containerd][].

To use containerd, add the following configurations to `cluster.yml`.

```yaml
options:
  kubelet:
    extra_binds:
    # The root directory for containerd metadata. (Default: "/var/lib/containerd")
    - source: /var/lib/containerd
      destination: /var/lib/containerd
      read_only: false
    cri_endpoint: /path/to/containerd/socket
```

[containerd]: https://containerd.io/
