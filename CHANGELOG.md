# Change Log

All notable changes to this project will be documented in this file.
This project employs a versioning scheme described in [RELEASE.md](RELEASE.md#versioning).

## [Unreleased]

### Changed

- Update kubernetes 1.28.12 [#756](https://github.com/cybozu-go/cke/pull/756)

## [1.28.6]

### Changed

- Add operation to delete 'updateStrategy: OnDelete' daemonset pods before rebooting in [#746](https://github.com/cybozu-go/cke/pull/746)

## [1.28.5]

### Changed

- Delay repair of an out-of-cluster unreachable machine in [#744](https://github.com/cybozu-go/cke/pull/744)

## [1.28.4]

### Changed

- Update etcd image to 3.5.14.1 [#742](https://github.com/cybozu-go/cke/pull/742)

## [1.28.3]

### Changed

- Update kubernetes 1.28.10 [#740](https://github.com/cybozu-go/cke/pull/740)

## [1.28.2]

### Added

- Add sabakan-triggered automatic repair functionality in [#725](https://github.com/cybozu-go/cke/pull/725) and [#732](https://github.com/cybozu-go/cke/pull/732)

### Changed

- Perform eviction dry-run before actual eviction during node reboot feature in [#736](https://github.com/cybozu-go/cke/pull/736)

### Fixed

- Fix not to send unassigned query parameters in Sabakan integration in [#725](https://github.com/cybozu-go/cke/pull/725)

## [1.28.1]

### Added

- Take in updates of CKE 1.27.11. See [CKE 1.27.11](https://github.com/cybozu-go/cke/blob/v1.27.11/CHANGELOG.md#12711).

### Changed

- change CKE to proceed rebooting immediately after draining of node is completed [#707](https://github.com/cybozu-go/cke/pull/707)
- change backoff algorithm to exponential backoff [#726](https://github.com/cybozu-go/cke/pull/726)

## [1.28.0]

### Changed

- Update for Kubernetes 1.28 [#721](https://github.com/cybozu-go/cke/pull/721)
- Update dependencies [#719](https://github.com/cybozu-go/cke/pull/719)
- Update actions/setup-go to v5 [#722](https://github.com/cybozu-go/cke/pull/722)

## [1.28.0-rc.1]

### Changed

- Update for Kubernetes 1.28 [#721](https://github.com/cybozu-go/cke/pull/721)
- Update dependencies [#719](https://github.com/cybozu-go/cke/pull/719)
- Update actions/setup-go to v5 [#722](https://github.com/cybozu-go/cke/pull/722)

## Ancient changes

- See [release-1.27/CHANGELOG.md](https://github.com/cybozu-go/cke/blob/release-1.27/CHANGELOG.md) for changes in CKE 1.27.
- See [release-1.26/CHANGELOG.md](https://github.com/cybozu-go/cke/blob/release-1.26/CHANGELOG.md) for changes in CKE 1.26.
- See [release-1.25/CHANGELOG.md](https://github.com/cybozu-go/cke/blob/release-1.25/CHANGELOG.md) for changes in CKE 1.25.
- See [release-1.24/CHANGELOG.md](https://github.com/cybozu-go/cke/blob/release-1.24/CHANGELOG.md) for changes in CKE 1.24.
- See [release-1.23/CHANGELOG.md](https://github.com/cybozu-go/cke/blob/release-1.23/CHANGELOG.md) for changes in CKE 1.23.
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

[Unreleased]: https://github.com/cybozu-go/cke/compare/v1.28.6...HEAD
[1.28.6]: https://github.com/cybozu-go/cke/compare/v1.28.5...v1.28.6
[1.28.5]: https://github.com/cybozu-go/cke/compare/v1.28.4...v1.28.5
[1.28.4]: https://github.com/cybozu-go/cke/compare/v1.28.3...v1.28.4
[1.28.3]: https://github.com/cybozu-go/cke/compare/v1.28.2...v1.28.3
[1.28.2]: https://github.com/cybozu-go/cke/compare/v1.28.1...v1.28.2
[1.28.1]: https://github.com/cybozu-go/cke/compare/v1.28.0...v1.28.1
[1.28.0]: https://github.com/cybozu-go/cke/compare/v1.27.10...v1.28.0
[1.28.0-rc.1]: https://github.com/cybozu-go/cke/compare/v1.27.10...v1.28.0-rc.1
