# Change Log

All notable changes to this project will be documented in this file.
This project adheres to [Semantic Versioning](http://semver.org/).

## [Unreleased]

## [0.18] - 2018-11-15
### Incompatibly Changed
- Changed parameter name in cluster config from "boot-taints" to "boot_taints" (#93).

### Changed
- Use fixed image version for multi-host tests (#94).

## [0.17] - 2018-11-13
### Changed
- Add boot-taints to Node resources again on reboot (#92)

## [0.16] - 2018-11-09
### Changed
- Fix ConfigMap name (#90).

## [0.15] - 2018-11-09
### Added
- Add cluster_overview.md (#89).
- Add Node local DNS deployment operation (#88).
- Add alternative name for cke-etcd (#87).

## [0.14] - 2018-11-07
### Changed
- Create CoreDNS ConfigMap either w/ or w/o upstream DNS servers (#86).

## [0.13] - 2018-11-06
### Changed
- Use cybzou container image for CoreDNS (#85).

## [0.12] - 2018-11-06
### Added
- Add support of Docker compose for quickstart (#82).
- Add CoreDNS deployment operation (#83).

### Changed
- Update Go modules for Go 1.11.2 (#84).

## [0.11] - 2018-10-30
### Added
- Enable to register a node with taint at boot (#77).
- Update node annotations, labels, and taints (#79).
- Remove non-cluster nodes (#80).
- Set deadline for SSH connection (#81).

### Changed
- Fix bugs in #75 (#78).

## [0.10] - 2018-10-18
### Added
- CKE registeres endpoints of etcd as a Kubernetes `Endpoints` (#75).

## [0.9] - 2018-10-17
### Changed
- Fixed API server certificates (#69).
- Revamped strategy and mtest (#66, #73)
- Miscellaneous bug fixes.

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

[Unreleased]: https://github.com/cybozu-go/cke/compare/v0.18...HEAD
[0.18]: https://github.com/cybozu-go/cke/compare/v0.17...v0.18
[0.17]: https://github.com/cybozu-go/cke/compare/v0.16...v0.17
[0.16]: https://github.com/cybozu-go/cke/compare/v0.15...v0.16
[0.15]: https://github.com/cybozu-go/cke/compare/v0.14...v0.15
[0.14]: https://github.com/cybozu-go/cke/compare/v0.13...v0.14
[0.13]: https://github.com/cybozu-go/cke/compare/v0.12...v0.13
[0.12]: https://github.com/cybozu-go/cke/compare/v0.11...v0.12
[0.11]: https://github.com/cybozu-go/cke/compare/v0.10...v0.11
[0.10]: https://github.com/cybozu-go/cke/compare/v0.9...v0.10
[0.9]: https://github.com/cybozu-go/cke/compare/v0.8...v0.9
[0.8]: https://github.com/cybozu-go/cke/compare/v0.7...v0.8
[0.7]: https://github.com/cybozu-go/cke/compare/v0.6...v0.7
[0.6]: https://github.com/cybozu-go/cke/compare/v0.5...v0.6
[0.5]: https://github.com/cybozu-go/cke/compare/v0.4...v0.5
[0.4]: https://github.com/cybozu-go/cke/compare/v0.3...v0.4
[0.3]: https://github.com/cybozu-go/cke/compare/v0.2...v0.3
[0.2]: https://github.com/cybozu-go/cke/compare/v0.1...v0.2
