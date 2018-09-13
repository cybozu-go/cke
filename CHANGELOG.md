# Change Log

All notable changes to this project will be documented in this file.
This project adheres to [Semantic Versioning](http://semver.org/).

## [Unreleased]

## [0.5] - 2018-09-13

### Added
- Kubernetes cluster has employed TLS security.
- Support for [Service Accounts][].

## [0.4] - 2018-09-06

### Added
- CKE deploys etcd with TLS (#31, #32).

## [0.3] - 2018-09-04

### Added
- CKE now uses Vault to issue TLS certificates (#24, #29).

### Changed
- Kubernetes is updated to 1.11.2 (#23).
- etcd is updated to 3.3.9.
- TLS is used for etcd communication.

## [0.2] - 2018-08-29

This is the first release.

### Added
- Deploy etcd and kubernetes services.

[Unreleased]: https://github.com/cybozu-go/sabakan/compare/v0.5...HEAD
[0.5]: https://github.com/cybozu-go/sabakan/compare/v0.4...v0.5
[0.4]: https://github.com/cybozu-go/sabakan/compare/v0.3...v0.4
[0.3]: https://github.com/cybozu-go/sabakan/compare/v0.2...v0.3
[0.2]: https://github.com/cybozu-go/sabakan/compare/v0.1...v0.2
[Service Accounts]: https://kubernetes.io/docs/reference/access-authn-authz/service-accounts-admin/
