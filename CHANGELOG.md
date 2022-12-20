# Change Log

All notable changes to this project will be documented in this file.
This project employs a versioning scheme described in [RELEASE.md](RELEASE.md#versioning).

## [Unreleased]

## [1.24.0-rc.2]

### Changed

- Update Kubernetes to v1.24.9 [#591](https://github.com/cybozu-go/cke/pull/591)
- Mount directories related to CNI on kubelet [#592](https://github.com/cybozu-go/cke/pull/592)
- Update coredns to 1.10.0 [#594](https://github.com/cybozu-go/cke/pull/594)

## [1.24.0-rc.1]

### Changed

- Support Kubernetes 1.24 [#584](https://github.com/cybozu-go/cke/pull/584)
    - Update Kubernetes to v1.24.8
    - Update some dependencies
    - Remove kubelet flag (`--network-plugin`) related to dockershim removal
- Fixed sonobuoy test failing. [#589](https://github.com/cybozu-go/cke/pull/589)
    - Fix docker-compose download URL
    - Fix confirmation of container exit status

## Ancient changes

- See [release-1.23/CHANGELOG.md](https://github.com/cybozu-go/cke/blob/release-1.23/CHANGELOG.md) for changes in CKE 1.2.
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

[Unreleased]: https://github.com/cybozu-go/cke/compare/v1.23.5...HEAD
[1.24.0-rc.2]: https://github.com/cybozu-go/cke/compare/1.24.0-rc.1...1.24.0-rc.2
[1.24.0-rc.1]: https://github.com/cybozu-go/cke/compare/v1.23.5...1.24.0-rc.1
