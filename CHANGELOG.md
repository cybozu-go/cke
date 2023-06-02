# Change Log

All notable changes to this project will be documented in this file.
This project employs a versioning scheme described in [RELEASE.md](RELEASE.md#versioning).

## [Unreleased]

## [1.25.4]

### Added 

- Update user-defined resource applying logic [#617](https://github.com/cybozu-go/cke/pull/617)

## [1.25.3]

### Changed

- Update Kubernetes to 1.25.10 in [#628](https://github.com/cybozu-go/cke/pull/628)

## [1.25.2]

### Fixed

- Fix Calico manifest to latest version [#627](https://github.com/cybozu-go/cke/pull/627)
 
### Changed

- Update for Kubernetes 1.25.9 [#624](https://github.com/cybozu-go/cke/pull/624)
  - Update Kubernetes to v1.25.9
  - Update some dependencies
- Update CKE image for example in [#618](https://github.com/cybozu-go/cke/pull/618)
  - Update CKE image for example
  - Add email address in PRODUCT.yaml

## [1.25.1]

### Changed

- Change behavior to not do SELinux labeling unless SELinux enforcing mode [#620](https://github.com/cybozu-go/cke/pull/620)
- Retry reboot command if failed in processing reboot queue in [#621](https://github.com/cybozu-go/cke/pull/621)

## [1.25.0]

### Changed

- Support Kubernetes 1.25 [#610](https://github.com/cybozu-go/cke/pull/610)
  - Update Kubernetes to v1.25.6
  - Update some dependencies

## [1.25.0-rc.1]

### Changed

- Support Kubernetes 1.25 [#610](https://github.com/cybozu-go/cke/pull/610)
  - Update Kubernetes to v1.25.6
  - Update some dependencies

## Ancient changes

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

[Unreleased]: https://github.com/cybozu-go/cke/compare/v1.25.4...HEAD
[1.25.4]: https://github.com/cybozu-go/cke/compare/v1.25.3...1.25.4
[1.25.3]: https://github.com/cybozu-go/cke/compare/v1.25.2...1.25.3
[1.25.2]: https://github.com/cybozu-go/cke/compare/v1.25.1...1.25.2
[1.25.1]: https://github.com/cybozu-go/cke/compare/v1.25.0...1.25.1
[1.25.0]: https://github.com/cybozu-go/cke/compare/v1.24.2...1.25.0
[1.25.0-rc.1]: https://github.com/cybozu-go/cke/compare/v1.24.2...1.25.0-rc.1
