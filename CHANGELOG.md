# Change Log

All notable changes to this project will be documented in this file.
This project employs a versioning scheme described in [RELEASE.md](RELEASE.md#versioning).

## [Unreleased]

- Delay repair of a rebooting unreachable machine in [#733](https://github.com/cybozu-go/cke/pull/733)

## [1.27.11]

### Added

- Add sabakan-triggered automatic repair functionality in [#727](https://github.com/cybozu-go/cke/pull/727)

### Fixed

- Fix not to send unassigned query parameters in Sabakan integration in [#727](https://github.com/cybozu-go/cke/pull/727)

## [1.27.10]

### Changed

- change priorty of reboot queue cancel [#715](https://github.com/cybozu-go/cke/pull/715)
- Embed root.hints file in unbound container [#716](https://github.com/cybozu-go/cke/pull/716)
- Add test for too long repair execution [#713](https://github.com/cybozu-go/cke/pull/713)

## [1.27.9]

### Added

- Add -output option to `ckecli reboot-queue list` [#708](https://github.com/cybozu-go/cke/pull/708)

### Changed

- Remove deprecated DualStack flag [#705](https://github.com/cybozu-go/cke/pull/705)
- Update Vault to 1.15.6 [#709](https://github.com/cybozu-go/cke/pull/709)
- Remove Unhealthy/Unreachable taint [#710](https://github.com/cybozu-go/cke/pull/710)
- Add rw option to extra volumes [#712](https://github.com/cybozu-go/cke/pull/712)

## [1.27.8]

This release was canceled because the release workflow was failed.

## [1.27.7]

### Fixed

- add setup-go in release-cke-image CI in [#703](https://github.com/cybozu-go/cke/pull/703)

## [1.27.6]

### Fixed

- fix GO_VERSION in bin/env-sonobuoy in [#701](https://github.com/cybozu-go/cke/pull/701)

## [1.27.5]

### Changed

- Fix condition of re-taint operation in [#699](https://github.com/cybozu-go/cke/pull/699)

## [1.27.4]

### Added

- Implement repair queue in [#692](https://github.com/cybozu-go/cke/pull/692)

## [1.27.3]

### Changed

- Adjustment network component resources in [#695](https://github.com/cybozu-go/cke/pull/695)
- Update for Kubernetes 1.27.10 [#696](https://github.com/cybozu-go/cke/pull/696)

## [1.27.2]

### Changed

- Update for Kubernetes 1.27.9 [#691](https://github.com/cybozu-go/cke/pull/691)

## [1.27.1]

### Added

- Implement ckecli resource get [#688](https://github.com/cybozu-go/cke/pull/688)

### Changed

- Fix reboot_queue_running to report internal-state more precisely [#685](https://github.com/cybozu-go/cke/pull/685)
- Update go modules [#689](https://github.com/cybozu-go/cke/pull/689)

## [1.27.0]

### Added

- Implement reboot-queue status metrics [#678](https://github.com/cybozu-go/cke/pull/678)

### Changed

- Update for Kubernetes 1.27.8 [#672](https://github.com/cybozu-go/cke/pull/672)
- Update Vault to 1.15.3 [#680](https://github.com/cybozu-go/cke/pull/680)
- Migrate to ghcr.io [#683](https://github.com/cybozu-go/cke/pull/683)

## [1.27.0-rc.2]

### Added

- Implement reboot-queue status metrics [#678](https://github.com/cybozu-go/cke/pull/678)

### Changed

- Update Vault to 1.15.3 [#680](https://github.com/cybozu-go/cke/pull/680)
- Migrate to ghcr.io [#683](https://github.com/cybozu-go/cke/pull/683)

## [1.27.0-rc.1]

### Changed

- Update for Kubernetes 1.27.8 [#672](https://github.com/cybozu-go/cke/pull/672)

## Ancient changes

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

[Unreleased]: https://github.com/cybozu-go/cke/compare/v1.27.11...HEAD
[1.27.11]: https://github.com/cybozu-go/cke/compare/v1.27.10...v1.27.11
[1.27.10]: https://github.com/cybozu-go/cke/compare/v1.27.9...v1.27.10
[1.27.9]: https://github.com/cybozu-go/cke/compare/v1.27.7...v1.27.9
[1.27.8]: https://github.com/cybozu-go/cke/compare/v1.27.7...v1.27.8
[1.27.7]: https://github.com/cybozu-go/cke/compare/v1.27.6...v1.27.7
[1.27.6]: https://github.com/cybozu-go/cke/compare/v1.27.5...v1.27.6
[1.27.5]: https://github.com/cybozu-go/cke/compare/v1.27.4...v1.27.5
[1.27.4]: https://github.com/cybozu-go/cke/compare/v1.27.3...v1.27.4
[1.27.3]: https://github.com/cybozu-go/cke/compare/v1.27.2...v1.27.3
[1.27.2]: https://github.com/cybozu-go/cke/compare/v1.27.1...v1.27.2
[1.27.1]: https://github.com/cybozu-go/cke/compare/v1.27.0...v1.27.1
[1.27.0]: https://github.com/cybozu-go/cke/compare/v1.26.4...v1.27.0
[1.27.0-rc.2]: https://github.com/cybozu-go/cke/compare/v1.27.0-rc.1...v1.27.0-rc.2
[1.27.0-rc.1]: https://github.com/cybozu-go/cke/compare/v1.26.4...v1.27.0-rc.1
