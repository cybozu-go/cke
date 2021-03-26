# Change Log

All notable changes to this project will be documented in this file.
This project employs a versioning scheme described in [RELEASE.md](RELEASE.md#versioning).

## [Unreleased]

## [1.19.7-rc.3] - 2021-03-26

No changes. Only for fixing GitHub release workflow.

## [1.19.7-rc.2] - 2021-03-26

No changes. Only for fixing GitHub release workflow.

## [1.19.7-rc.1] - 2021-03-25

No changes. Only for fixing GitHub release workflow.

## [1.19.6] - 2021-03-22

### Added
- New optional service `cke-localproxy` (#433)

## [1.19.5] - 2021-03-13

### Added
- New option `audit_log_path` in `APIServerParams` (#429)

### Changed
- server: reflect sabakan machine status while rebooting nodes (#425)

## [1.19.4] - 2021-02-19

### Changed
- server: reflect sabakan machine status while rebooting nodes (#425)

## [1.19.3] - 2021-02-09

### Changed
- node-dns: update Pods in bulk. (#420)
- Skip reboot if etcd cluster is not responding and in-sync. (#417)
- Update cke-tools to 1.19.2
    - `rivers` now checks connectivity to the upstream servers periodically.

## [1.19.2] - 2021-01-28

### Added
- Proxy mode setting for kube-proxy (#413)

### Changed
- Cherry-pick [kubernetes/kubernetes#97081](https://github.com/kubernetes/kubernetes/pull/97081)

## [1.19.1] - 2021-01-26

### Changed
- A helper container image for CKE called `cke-tools` is now built from `scratch`. (#408)

### Removed
- `etcdbackup`, a feature to backup CKE-managed etcd automatically, is removed. (#410)

## [1.19.0] - 2021-01-20

### Added
- ckecli: prevent rebooting multiple control plane nodes. (#405)

## [1.19.0-rc.1] - 2021-01-15

### Added
- kube-scheduler can be configured with [KubeSchedulerConfiguration v1beta1](https://pkg.go.dev/k8s.io/kube-scheduler@v0.19.7/config/v1beta1#KubeSchedulerConfiguration).
- Fields in [KubeletConfiguration](https://pkg.go.dev/k8s.io/kubelet/config/v1beta1#KubeletConfiguration) that are unsafe to be changed are kept while the node is running.

### Changed
- Rename `container_runtime_endpoint` in clutser.yml to `cri_endpoint`
- Update images
  - Kubernetes 1.19.7
  - cke-tools 1.7.4
  - CoreDNS 1.8.0
  - Unbound 1.13.0

### Removed
- kube-scheduler v1alpha1 and v1alpha2 configurations
- legacy configuration options for kube-scheduler and kubelet

## Ancient changes

- See [release-1.18/CHANGELOG.md](https://github.com/cybozu-go/cke/blob/release-1.18/CHANGELOG.md) for changes in CKE 1.18.
- See [release-1.17/CHANGELOG.md](https://github.com/cybozu-go/cke/blob/release-1.17/CHANGELOG.md) for changes in CKE 1.17.
- See [release-1.16/CHANGELOG.md](https://github.com/cybozu-go/cke/blob/release-1.16/CHANGELOG.md) for changes in CKE 1.16.
- See [release-1.15/CHANGELOG.md](https://github.com/cybozu-go/cke/blob/release-1.15/CHANGELOG.md) for changes in CKE 1.15.
- See [release-1.14/CHANGELOG.md](https://github.com/cybozu-go/cke/blob/release-1.14/CHANGELOG.md) for changes in CKE 1.14.
- See [release-1.13/CHANGELOG.md](https://github.com/cybozu-go/cke/blob/release-1.13/CHANGELOG.md) for changes in CKE 1.13.
- See [release-1.12/CHANGELOG.md](https://github.com/cybozu-go/cke/blob/release-1.12/CHANGELOG.md) for changes in CKE 1.12.

[Unreleased]: https://github.com/cybozu-go/cke/compare/v1.19.7-rc.3...HEAD
[1.19.7-rc.3]: https://github.com/cybozu-go/cke/compare/v1.19.7-rc.2...v1.19.7-rc.3
[1.19.7-rc.2]: https://github.com/cybozu-go/cke/compare/v1.19.7-rc.1...v1.19.7-rc.2
[1.19.7-rc.1]: https://github.com/cybozu-go/cke/compare/v1.19.6...v1.19.7-rc.1
[1.19.6]: https://github.com/cybozu-go/cke/compare/v1.19.5...v1.19.6
[1.19.5]: https://github.com/cybozu-go/cke/compare/v1.19.4...v1.19.5
[1.19.4]: https://github.com/cybozu-go/cke/compare/v1.19.3...v1.19.4
[1.19.3]: https://github.com/cybozu-go/cke/compare/v1.19.2...v1.19.3
[1.19.2]: https://github.com/cybozu-go/cke/compare/v1.19.1...v1.19.2
[1.19.1]: https://github.com/cybozu-go/cke/compare/v1.19.0...v1.19.1
[1.19.0]: https://github.com/cybozu-go/cke/compare/v1.19.0-rc.1...v1.19.0
[1.19.0-rc.1]: https://github.com/cybozu-go/cke/compare/v1.18.8...v1.19.0-rc.1
