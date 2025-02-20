# Change Log

All notable changes to this project will be documented in this file.
This project employs a versioning scheme described in [RELEASE.md](RELEASE.md#versioning).


## [Unreleased]

## [1.30.6]

### Changed

- Supress dns log [#790](https://github.com/cybozu-go/cke/pull/790)
  - Supress cluster-dns's log
  - Supress node-dns's log
  - Change the test instance's zone

## [1.30.5]

### Changed

- fix runRepairer to check nil in order to avoid SEGV [#788](https://github.com/cybozu-go/cke/pull/788)
- Enabling etcd data corruption detection [#787](https://github.com/cybozu-go/cke/pull/787)

## [1.30.4]

### Changed

- fix sabakan integration to handle role that is not exist in sabakan [#784](https://github.com/cybozu-go/cke/pull/784)

## [1.30.3]

### Changed

- Revive unreachable taint [#780](https://github.com/cybozu-go/cke/pull/780)

## [1.30.2]

### Added

- add feature to execute user-defined command when repair is successfully finished [#753](https://github.com/cybozu-go/cke/pull/753)

## [1.30.1]

### Changed

- Set /sys mount of kubelet as read-write [#773](https://github.com/cybozu-go/cke/pull/773)

## [1.30.0]

- No changes from 1.30.0-rc.2

## [1.30.0-rc.2]

### Changed

- Update Unbound to 1.21.1 [#769](https://github.com/cybozu-go/cke/pull/769)

## [1.30.0-rc.1]

### Changed

- Update Kubernetes to 1.30.5 [#767](https://github.com/cybozu-go/cke/pull/767)

## Ancient changes

- See [release-1.29/CHANGELOG.md](https://github.com/cybozu-go/cke/blob/release-1.29/CHANGELOG.md) for changes in CKE 1.29.
- See [release-1.28/CHANGELOG.md](https://github.com/cybozu-go/cke/blob/release-1.28/CHANGELOG.md) for changes in CKE 1.28.
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

[Unreleased]: https://github.com/cybozu-go/cke/compare/v1.30.6...HEAD
[1.30.6]: https://github.com/cybozu-go/cke/compare/v1.30.5...v1.30.6
[1.30.5]: https://github.com/cybozu-go/cke/compare/v1.30.4...v1.30.5
[1.30.4]: https://github.com/cybozu-go/cke/compare/v1.30.3...v1.30.4
[1.30.3]: https://github.com/cybozu-go/cke/compare/v1.30.2...v1.30.3
[1.30.2]: https://github.com/cybozu-go/cke/compare/v1.30.1...v1.30.2
[1.30.1]: https://github.com/cybozu-go/cke/compare/v1.30.0...v1.30.1
[1.30.0]: https://github.com/cybozu-go/cke/compare/v1.30.0-rc.2...v1.30.0
[1.30.0-rc.2]: https://github.com/cybozu-go/cke/compare/v1.30.0-rc.1...v1.30.0-rc.2
[1.30.0-rc.1]: https://github.com/cybozu-go/cke/compare/v1.29.0...v1.30.0-rc.1
