Container Runtime support
=========================

CKE deployed containers
-----------------------

These deployment supports only [Docker] container runtime.

- `etcd`
- `kube-apiserver`
- `kube-controller-manager`
- `kube-scheduler`
- `kubelet`
- [rivers](https://github.com/cybozu/neco-containers/tree/master/cke-tools/src/cmd/rivers)

Kubernetes Pods
---------------

CKE project is testing Kubernetes deployment with container runtime as follows.

- [Docker]

  If Docker socket path is not default, add runtime options to `cluster.yml`.

```yaml
options:
  kubelet:
    container_runtime: docker
    container_runtime_endpoint: /path/to/docker/socket
```

  `container_runtime_endpoint` is `/var/run/dockershim.sock` by default.

- [containerd] v1.2

  To use containerd as container runtime for the Pods, add runtime option to `cluster.yml`.

```yaml
options:
  kubelet:
    extra_binds:
    # The root directory for containerd metadata. (Default: "/var/lib/containerd")
    - source: /var/lib/containerd
      destination: /var/lib/containerd
      read_only: false
    container_runtime: remote
    container_runtime_endpoint: /path/to/containerd/socket
```

[Docker]: https://www.docker.com/
[containerd]: https://containerd.io/
