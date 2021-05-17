# Change Log

All notable changes to this project will be documented in this file.
This project employs a versioning scheme described in [RELEASE.md](RELEASE.md#versioning).

## [Unreleased]

## [1.20.2] - 2021-05-17

- Update revision cke.cybozu.com/revision (#462)

## [1.20.1] - 2021-05-13

- Update kubernete image to 1.20.7

## [1.20.0] - 2021-05-12

No change from 1.20.0-rc.1

## [1.20.0-rc.1] - 2021-05-07

### Added
- Support [KubeProxyConfiguration](https://pkg.go.dev/k8s.io/kube-proxy@v0.20.6/config/v1alpha1#KubeProxyConfiguration)

### Changed
- Update images
  - Kubernetes 1.20.6
  - cke-tools 1.20.0
  - CoreDNS 1.8.3
  - Unbound 1.13.1

### Removed
- Disable PodSecurityPolicy
- Drop Docker support

## Ancient changes

- See [release-1.19/CHANGELOG.md](https://github.com/cybozu-go/cke/blob/release-1.19/CHANGELOG.md) for changes in CKE 1.19.
- See [release-1.18/CHANGELOG.md](https://github.com/cybozu-go/cke/blob/release-1.18/CHANGELOG.md) for changes in CKE 1.18.
- See [release-1.17/CHANGELOG.md](https://github.com/cybozu-go/cke/blob/release-1.17/CHANGELOG.md) for changes in CKE 1.17.
- See [release-1.16/CHANGELOG.md](https://github.com/cybozu-go/cke/blob/release-1.16/CHANGELOG.md) for changes in CKE 1.16.
- See [release-1.15/CHANGELOG.md](https://github.com/cybozu-go/cke/blob/release-1.15/CHANGELOG.md) for changes in CKE 1.15.
- See [release-1.14/CHANGELOG.md](https://github.com/cybozu-go/cke/blob/release-1.14/CHANGELOG.md) for changes in CKE 1.14.
- See [release-1.13/CHANGELOG.md](https://github.com/cybozu-go/cke/blob/release-1.13/CHANGELOG.md) for changes in CKE 1.13.
- See [release-1.12/CHANGELOG.md](https://github.com/cybozu-go/cke/blob/release-1.12/CHANGELOG.md) for changes in CKE 1.12.

[Unreleased]: https://github.com/cybozu-go/cke/compare/v1.20.2...HEAD
[1.20.2]: https://github.com/cybozu-go/cke/compare/v1.20.1...v1.20.2
[1.20.1]: https://github.com/cybozu-go/cke/compare/v1.20.0...v1.20.1
[1.20.0]: https://github.com/cybozu-go/cke/compare/v1.20.0-rc.1...v1.20.0
[1.20.0-rc.1]: https://github.com/cybozu-go/cke/compare/v1.19.8...v1.20.0-rc.1
