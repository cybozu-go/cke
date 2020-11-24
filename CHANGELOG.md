# Change Log

All notable changes to this project will be documented in this file.
This project employs a versioning scheme described in [RELEASE.md](RELEASE.md#versioning).

## [Unreleased]

## [1.18.4] - 2020-11-24

### Added

- metrics: `reboot_queue_entries` (#385)

## [1.18.3] - 2020-11-18

### Added

- Added node maintenance feature (#381, #382, #383).

### Fixed

- Fixed preStop command for cluster-dns (#378).

## [1.18.2] - 2020-10-18

### Added

- `ckecli etcd local-backup` to take backups of CKE-managed etcd (#375).

### Changed

- Change `terminationGracePeriodSeconds` for node DNS pods from 0 to 1 (#373).
- `ckecli vault init` outputs messages about what it has done (#374).

## [1.18.1] - 2020-10-12

### Added

- Issue and embed TLS certificates for admission webhooks (#371).

### Changed

- The user-defined resource feature is extended to accept all the standard Kubernetes resources (#371).

## [1.18.0] - 2020-10-05

### Changed

- Use Flatcar Container Linux (#365)

## [1.18.0-rc.1] - 2020-09-23

### Added
- New styles of configurations for k8s components are available.
  - Kubelet is now configurable by embedding [KubeletConfiguration v1beta1](https://pkg.go.dev/k8s.io/kubelet@v0.18.9/config/v1beta1#KubeletConfiguration) directly.
  - Kube-scheduler is now configurable by embedding [KubeSchedulerConfiguration v1alpha2](https://pkg.go.dev/k8s.io/kube-scheduler@v0.18.9/config/v1alpha2#KubeSchedulerConfiguration) directly.
  - See the [KEP in sig-cluster-lifecycle](https://github.com/kubernetes/enhancements/blob/master/keps/sig-cluster-lifecycle/wgs/0014-20180707-componentconfig-api-types-to-staging.md#migration-strategy-per-component-or-k8sio-repo) for more detail on the component configuration.

### Changed
- Update images.
  - etcd 3.3.25
  - Kubernetes 1.18.9
  - cke-tools 1.7.2
  - CoreDNS 1.7.0
  - Unbound 1.11.0
- Update manifests in example.

## Ancient changes

- See [release-1.17/CHANGELOG.md](https://github.com/cybozu-go/cke/blob/release-1.17/CHANGELOG.md) for changes in CKE 1.17.
- See [release-1.16/CHANGELOG.md](https://github.com/cybozu-go/cke/blob/release-1.16/CHANGELOG.md) for changes in CKE 1.16.
- See [release-1.15/CHANGELOG.md](https://github.com/cybozu-go/cke/blob/release-1.15/CHANGELOG.md) for changes in CKE 1.15.
- See [release-1.14/CHANGELOG.md](https://github.com/cybozu-go/cke/blob/release-1.14/CHANGELOG.md) for changes in CKE 1.14.
- See [release-1.13/CHANGELOG.md](https://github.com/cybozu-go/cke/blob/release-1.13/CHANGELOG.md) for changes in CKE 1.13.
- See [release-1.12/CHANGELOG.md](https://github.com/cybozu-go/cke/blob/release-1.12/CHANGELOG.md) for changes in CKE 1.12.

[Unreleased]: https://github.com/cybozu-go/cke/compare/v1.18.4...HEAD
[1.18.4]: https://github.com/cybozu-go/cke/compare/v1.18.3...v1.18.4
[1.18.3]: https://github.com/cybozu-go/cke/compare/v1.18.2...v1.18.3
[1.18.2]: https://github.com/cybozu-go/cke/compare/v1.18.1...v1.18.2
[1.18.1]: https://github.com/cybozu-go/cke/compare/v1.18.0...v1.18.1
[1.18.0]: https://github.com/cybozu-go/cke/compare/v1.18.0-rc.1...v1.18.0
[1.18.0-rc.1]: https://github.com/cybozu-go/cke/compare/v1.17.11...v1.18.0-rc.1
