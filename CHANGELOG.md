# Change Log

All notable changes to this project will be documented in this file.
This project employs a versioning scheme described in [RELEASE.md](RELEASE.md#versioning).

## [Unreleased]

## [1.26.4]

### Changed

- Backport #675 to enable DNSSEC validation [#676](https://github.com/cybozu-go/cke/pull/676)

## [1.26.3]

### Added

- Add reboot queue backoff reset command [#667](https://github.com/cybozu-go/cke/pull/667)

### Fixed

- Expose CoreDNS metrics on host [#668](https://github.com/cybozu-go/cke/pull/668)

## [1.26.2]

### Added

- Add `register-date` and `retire-date` labels [#663](https://github.com/cybozu-go/cke/pull/663)

### Fixed

- Fix `cke_node_reboot_status` metrics [#660](https://github.com/cybozu-go/cke/pull/660)
- Fix blocking by kubelet-restart op [#661](https://github.com/cybozu-go/cke/pull/661)

## [1.26.1]

### Added

- Retry eviction [#633](https://github.com/cybozu-go/cke/pull/633)

### Changed

- Revert the custom rank feature for user defined resources(#640, #638, #634, #617) [#655](https://github.com/cybozu-go/cke/pull/655)

### Fixed

- Fix to check error of etcd watch response in [#654](https://github.com/cybozu-go/cke/pull/654)

## [1.26.0]

### Added

- Add setting of reboot retry interval in [#645](https://github.com/cybozu-go/cke/pull/645)

### Changed

- Update for Kubernetes 1.26.6 [#646](https://github.com/cybozu-go/cke/pull/646)

## [1.26.0-rc.1]

### Added

- Add setting of reboot retry interval in [#645](https://github.com/cybozu-go/cke/pull/645)

### Changed

- Update for Kubernetes 1.26.6 [#646](https://github.com/cybozu-go/cke/pull/646)

## Ancient changes

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

[Unreleased]: https://github.com/cybozu-go/cke/compare/v1.26.4...HEAD
[1.26.4]: https://github.com/cybozu-go/cke/compare/v1.26.3...v1.26.4
[1.26.3]: https://github.com/cybozu-go/cke/compare/v1.26.2...v1.26.3
[1.26.2]: https://github.com/cybozu-go/cke/compare/v1.26.1...v1.26.2
[1.26.1]: https://github.com/cybozu-go/cke/compare/v1.26.0...v1.26.1
[1.26.0]: https://github.com/cybozu-go/cke/compare/v1.25.8...v1.26.0
[1.26.0-rc.1]: https://github.com/cybozu-go/cke/compare/v1.25.8...v1.26.0-rc.1
