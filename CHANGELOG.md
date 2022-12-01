# Change Log

All notable changes to this project will be documented in this file.
This project employs a versioning scheme described in [RELEASE.md](RELEASE.md#versioning).

## [Unreleased]

## [1.23.6] - 2022-12-02

### Changed

- Update etcd to 3.5.6 (#585)

## [1.23.5] - 2022-11-08

### Changed

- Update etcd to v3.5.5 (#576)

## [1.23.4] - 2022-10-21

### Changed

- **\[Action Required\]** Don't use tainted node as control plane node (#572) \
    Specify `control_plane_tolerations` in the [cluster template for sabakan integration](docs/sabakan-integration.md#cluster-template)
    if you or your system add taints to nodes and if you want to run
    control plane on tainted nodes.

## [1.23.3] - 2022-10-11

### Changed

- Update unbound and coredns (#571)

## [1.23.2] - 2022-09-22

### Changed

- Update unbound to 1.16.3.1 (#569)
- Update unbound to 1.16.1.2 and tune unbound.conf (#567)

## [1.23.1] - 2022-09-14

### Fixed

- Improve agent connection handling during rebooting (#565)

### Others

- Update CKE image for example (#561)
- Update product logo URL for sonobuoy (#562)

## [1.23.0] - 2022-08-12

### Changelog since 1.22.9, the latest version of 1.22.x

#### Changed

- Support Kubernetes 1.23 (#554)
    - Update Kubernetes to v1.23.9
    - Update some depencencies
    - Use KubeschedulerConfiguration v1beta3
    - Stop using deprecated --port option of kube-scheduler
- Minor changes to test (#555)
- Stop using deprecated --register-with-taints option (#557)

### Changelog since 1.23.0-rc.2, the latest rc version of 1.23.0

(nothing)

## [1.23.0-rc.2] - 2022-08-09

### Changed

- Stop using deprecated --register-with-taints option (#557)

## [1.23.0-rc.1] - 2022-08-04

### Changed

- Support Kubernetes 1.23 (#554)
    - Update Kubernetes to v1.23.9
    - Update some depencencies
    - Use KubeschedulerConfiguration v1beta3
    - Stop using deprecated --port option of kube-scheduler
- Minor changes to test (#555)

## Ancient changes

- See [release-1.22/CHANGELOG.md](https://github.com/cybozu-go/cke/blob/release-1.22/CHANGELOG.md) for changes in CKE 1.22.
- See [release-1.21/CHANGELOG.md](https://github.com/cybozu-go/cke/blob/release-1.21/CHANGELOG.md) for changes in CKE 1.21.
- See [release-1.20/CHANGELOG.md](https://github.com/cybozu-go/cke/blob/release-1.20/CHANGELOG.md) for changes in CKE 1.20.
- See [release-1.19/CHANGELOG.md](https://github.com/cybozu-go/cke/blob/release-1.19/CHANGELOG.md) for changes in CKE 1.19.
- See [release-1.18/CHANGELOG.md](https://github.com/cybozu-go/cke/blob/release-1.18/CHANGELOG.md) for changes in CKE 1.18.
- See [release-1.17/CHANGELOG.md](https://github.com/cybozu-go/cke/blob/release-1.17/CHANGELOG.md) for changes in CKE 1.17.
- See [release-1.16/CHANGELOG.md](https://github.com/cybozu-go/cke/blob/release-1.16/CHANGELOG.md) for changes in CKE 1.16.
- See [release-1.15/CHANGELOG.md](https://github.com/cybozu-go/cke/blob/release-1.15/CHANGELOG.md) for changes in CKE 1.15.
- See [release-1.14/CHANGELOG.md](https://github.com/cybozu-go/cke/blob/release-1.14/CHANGELOG.md) for changes in CKE 1.14.
- See [release-1.13/CHANGELOG.md](https://github.com/cybozu-go/cke/blob/release-1.13/CHANGELOG.md) for changes in CKE 1.13.
- See [release-1.12/CHANGELOG.md](https://github.com/cybozu-go/cke/blob/release-1.12/CHANGELOG.md) for changes in CKE 1.12.

[Unreleased]: https://github.com/cybozu-go/cke/compare/v1.23.6...HEAD
[1.23.6]: https://github.com/cybozu-go/cke/compare/v1.23.5...v1.23.6
[1.23.5]: https://github.com/cybozu-go/cke/compare/v1.23.4...v1.23.5
[1.23.4]: https://github.com/cybozu-go/cke/compare/v1.23.3...v1.23.4
[1.23.3]: https://github.com/cybozu-go/cke/compare/v1.23.2...v1.23.3
[1.23.2]: https://github.com/cybozu-go/cke/compare/v1.23.1...v1.23.2
[1.23.1]: https://github.com/cybozu-go/cke/compare/v1.23.0...v1.23.1
[1.23.0]: https://github.com/cybozu-go/cke/compare/v1.22.9...v1.23.0
[1.23.0-rc.2]: https://github.com/cybozu-go/cke/compare/v1.23.0-rc.1...v1.23.0-rc.2
[1.23.0-rc.1]: https://github.com/cybozu-go/cke/compare/v1.22.9...v1.23.0-rc.1
