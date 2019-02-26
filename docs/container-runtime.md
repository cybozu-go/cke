Container Runtime support
=========================

CKE deployed containers
-----------------------

These deployment supports only [Docker] container runtime.

- `etcd`
- `kube-apiserver`
- `kube-controller-manager`
- `kube-scheduler`
- [rivers](https://github.com/cybozu-go/cke-tools/tree/master/cmd/rivers)


Kubernetes Pods
---------------

CKE project is testing Kubernetes deployment with each container runtime as follows.

- [Docker]

  To use Docker as container runtime, add runtime option to `cluster.yml`.

```yaml
options:
  kubelet:
    container_runtime: docker
```

  `container_runtime_endpoint` is `/var/run/dockershim.sock` by default.

- [containerd] v1.2

  To use containerd as container runtime, add runtime option to `cluster.yml`.

```yaml
options:
  kubelet:
    container_runtime: remote
    container_runtime_endpoint: /path/to/containerd/docket
```


[Docker]: https://www.docker.com/
[containerd]: https://containerd.io/
