# Change Log

All notable changes to this project will be documented in this file.
This project employs a versioning scheme described in [RELEASE.md](RELEASE.md#versioning).

## [Unreleased]

## [1.21.0-rc.2] - 2021-10-12

### Changed

- Add FQDN to Subject Alternative Names in certificate for API servers (#493)
- Maintain EndpointSlice for etcd and API servers as well as Endpoints (#494)
- Use policy/v1 PodDisruptionBudget instead of policy/v1beta1 (#496)
- Run Sonobuoy with Calico instead of flannel (#492)
- Update how to run Sonobuoy (#495)

## [1.21.0-rc.1] - 2021-10-01

### Added

- Enable "DenyServiceExternalIPs" admission plugin (#487)

### Changed

- Update images (#485)
  - Kubernetes 1.21.5
  - cke-tools 1.21.0
  - pause 3.6
  - CoreDNS 1.8.5
  - Unbound 1.13.2
- Use Go 1.17 (#485)
- ckecli rq: Delete non-protected pod when Eviction API returns any kind of errors (#482)
- Remove redundant list of admission plugins enabled by default (#487)

## Ancient changes

- See [release-1.20/CHANGELOG.md](https://github.com/cybozu-go/cke/blob/release-1.20/CHANGELOG.md) for changes in CKE 1.20.
- See [release-1.19/CHANGELOG.md](https://github.com/cybozu-go/cke/blob/release-1.19/CHANGELOG.md) for changes in CKE 1.19.
- See [release-1.18/CHANGELOG.md](https://github.com/cybozu-go/cke/blob/release-1.18/CHANGELOG.md) for changes in CKE 1.18.
- See [release-1.17/CHANGELOG.md](https://github.com/cybozu-go/cke/blob/release-1.17/CHANGELOG.md) for changes in CKE 1.17.
- See [release-1.16/CHANGELOG.md](https://github.com/cybozu-go/cke/blob/release-1.16/CHANGELOG.md) for changes in CKE 1.16.
- See [release-1.15/CHANGELOG.md](https://github.com/cybozu-go/cke/blob/release-1.15/CHANGELOG.md) for changes in CKE 1.15.
- See [release-1.14/CHANGELOG.md](https://github.com/cybozu-go/cke/blob/release-1.14/CHANGELOG.md) for changes in CKE 1.14.
- See [release-1.13/CHANGELOG.md](https://github.com/cybozu-go/cke/blob/release-1.13/CHANGELOG.md) for changes in CKE 1.13.
- See [release-1.12/CHANGELOG.md](https://github.com/cybozu-go/cke/blob/release-1.12/CHANGELOG.md) for changes in CKE 1.12.

[Unreleased]: https://github.com/cybozu-go/cke/compare/v1.21.0-rc.2...HEAD
[1.21.0-rc.2]: https://github.com/cybozu-go/cke/compare/v1.21.0-rc.1...v1.21.0-rc.2
[1.21.0-rc.1]: https://github.com/cybozu-go/cke/compare/v1.20.5...v1.21.0-rc.1
