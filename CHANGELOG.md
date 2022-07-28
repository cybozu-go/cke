# Change Log

All notable changes to this project will be documented in this file.
This project employs a versioning scheme described in [RELEASE.md](RELEASE.md#versioning).

## [Unreleased]

## [1.22.9] - 2022-07-27

### Fixed

- sonobuoy test: follow flatcar 3227.2.0 (#551)

## [1.22.8] - 2022-07-22

*This version was not actually released.*

**This version has a breaking change around reboot feature.** If you update from 1.22.7 or before,
- reboot configuration SHOULD be updated.
- reboot queue SHOULD be empty.

### Changed

- add inter-node distribution for cluster-dns (#549)
- parallel reboot feature (#540)

## [1.22.7] - 2022-07-15

### Changed

- Update Kubernetes to 1.22.12 (#547)

## [1.22.6] - 2022-06-30

### Changed

- Update Kubernetes to 1.22.11 (#542)

## [1.22.5] - 2022-05-19

### Added

- add unbound_exporter to node-dns Pods (#536)

### Changed

- use CoreDNS ready plugin (#537)

## [1.22.4] - 2022-04-25

### Changed

- Update etcd to 3.5.4 (#534)

## [1.22.3] - 2022-04-14

### Changed

- Update etcd to 3.5.3 (#531)

## [1.22.2] - 2022-04-07

### Changed

- surge update node-dns (#527)
- Add `--experimental-initial-corrupt-check` flag for etcd (#529)

## [1.22.1] - 2022-01-25

### Fixed

- add authn/authz kubeconfig options to controller-manager/scheduler (#524)

## [1.22.0] - 2022-01-04

### Changelog since 1.21.2, the latest version of 1.21.x

#### Changed

- Update images (#518)
  - Kubernetes 1.22.5
  - cke-tools 1.22.0
  - etcd 3.5.1
  - CoreDNS 1.8.6
  - Unbound 1.14.0

### Changelog since 1.22.0-rc.1, the latest rc version of 1.22.0

(nothing)

## [1.22.0-rc.1] - 2022-01-04

### Changed

- Update images (#518)
  - Kubernetes 1.22.5
  - cke-tools 1.22.0
  - etcd 3.5.1
  - CoreDNS 1.8.6
  - Unbound 1.14.0

## Ancient changes

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

[Unreleased]: https://github.com/cybozu-go/cke/compare/v1.22.9...HEAD
[1.22.9]: https://github.com/cybozu-go/cke/compare/v1.22.8...v1.22.9
[1.22.8]: https://github.com/cybozu-go/cke/compare/v1.22.7...v1.22.8
[1.22.7]: https://github.com/cybozu-go/cke/compare/v1.22.6...v1.22.7
[1.22.6]: https://github.com/cybozu-go/cke/compare/v1.22.5...v1.22.6
[1.22.5]: https://github.com/cybozu-go/cke/compare/v1.22.4...v1.22.5
[1.22.4]: https://github.com/cybozu-go/cke/compare/v1.22.3...v1.22.4
[1.22.3]: https://github.com/cybozu-go/cke/compare/v1.22.2...v1.22.3
[1.22.2]: https://github.com/cybozu-go/cke/compare/v1.22.1...v1.22.2
[1.22.1]: https://github.com/cybozu-go/cke/compare/v1.22.0...v1.22.1
[1.22.0]: https://github.com/cybozu-go/cke/compare/v1.21.2...v1.22.0
[1.22.0-rc.1]: https://github.com/cybozu-go/cke/compare/v1.21.2...v1.22.0-rc.1
