# Change Log

All notable changes to this project will be documented in this file.
This project adheres to [Semantic Versioning](http://semver.org/).

## [Unreleased]

## [0.8] - 2018-09-28
### Added
- Add ckecli subcommands to issue client certificate.

## [0.7] - 2018-09-21
### Changed
- Change Docker image file system to ext4 from btrfs.

## [0.6] - 2018-09-19
### Added
- Opt in to [Go modules](https://github.com/golang/go/wiki/Modules).
- Enable [RBAC](https://kubernetes.io/docs/reference/access-authn-authz/rbac/) (#47).
- Enable [CNI network plugin](https://kubernetes.io/docs/concepts/extend-kubernetes/compute-storage-net/network-plugins/) (#54).
- Support SELinux enabled node OS (#50, #53).

## [0.5] - 2018-09-13

### Added
- Kubernetes cluster has employed TLS security.
- Support for [Service Accounts](https://kubernetes.io/docs/reference/access-authn-authz/service-accounts-admin/) (#43).

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

[Unreleased]: https://github.com/cybozu-go/cke/compare/v0.8...HEAD
[0.8]: https://github.com/cybozu-go/cke/compare/v0.7...v0.8
[0.7]: https://github.com/cybozu-go/cke/compare/v0.6...v0.7
[0.6]: https://github.com/cybozu-go/cke/compare/v0.5...v0.6
[0.5]: https://github.com/cybozu-go/cke/compare/v0.4...v0.5
[0.4]: https://github.com/cybozu-go/cke/compare/v0.3...v0.4
[0.3]: https://github.com/cybozu-go/cke/compare/v0.2...v0.3
[0.2]: https://github.com/cybozu-go/cke/compare/v0.1...v0.2
[Service Accounts]: https://kubernetes.io/docs/reference/access-authn-authz/service-accounts-admin/
