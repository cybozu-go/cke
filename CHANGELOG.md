# Change Log

All notable changes to this project will be documented in this file.
This project employs a versioning scheme described in [RELEASE.md](RELEASE.md#versioning).

## [Unreleased]

## [1.16.1] - 2019-12-25

### Changed
- Suppress vault tidy exec and logs. (#279)

### Added
- Add "ckecli status" and refactor mtest using it. (#278)

## [1.16.0] - 2019-12-23

### Changed
- Extend sonobuoy timeout period (#276)

## [1.16.0-rc.3] - 2019-12-19

### Changed
- Fix tar decompression target filenames (#273)

## [1.16.0-rc.2] - 2019-12-19

### Changed
- Add mode option to sonobuoy (#270)

## [1.16.0-rc.1] - 2019-12-18

### Changed
- Update images for Kubernetes 1.16 (#262)
    - hyperkube 1.16.4
    - CNI plugins 0.8.3
    - CoreDNS 1.6.6
    - Unbound 1.9.5

## Ancient changes

- See [release-1.15/CHANGELOG.md](https://github.com/cybozu-go/cke/blob/release-1.15/CHANGELOG.md) for changes in CKE 1.15.
- See [release-1.14/CHANGELOG.md](https://github.com/cybozu-go/cke/blob/release-1.14/CHANGELOG.md) for changes in CKE 1.14.
- See [release-1.13/CHANGELOG.md](https://github.com/cybozu-go/cke/blob/release-1.13/CHANGELOG.md) for changes in CKE 1.13.
- See [release-1.12/CHANGELOG.md](https://github.com/cybozu-go/cke/blob/release-1.12/CHANGELOG.md) for changes in CKE 1.12.

[Unreleased]: https://github.com/cybozu-go/cke/compare/v1.16.1...HEAD
[1.16.1]: https://github.com/cybozu-go/cke/compare/v1.16.0...v1.16.1
[1.16.0]: https://github.com/cybozu-go/cke/compare/v1.16.0-rc.3...v1.16.0
[1.16.0-rc.3]: https://github.com/cybozu-go/cke/compare/v1.16.0-rc.2...v1.16.0-rc.3
[1.16.0-rc.2]: https://github.com/cybozu-go/cke/compare/v1.16.0-rc.1...v1.16.0-rc.2
[1.16.0-rc.1]: https://github.com/cybozu-go/cke/compare/v1.15.7...v1.16.0-rc.1
