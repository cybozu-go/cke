# Change Log

All notable changes to this project will be documented in this file.
This project adheres to [versioning](RELEASE.md#versioning).

## [Unreleased]

## [1.13.10] - 2019-03-18

### Changed
- CKE is ready for enabling PodSecurityPolicy (#149).

## [1.13.9] - 2019-03-15

### Added
- [User-defined resources](docs/user-resources.md) (#145).

### Changed
- Enable [NodeRestriction admission controller](https://kubernetes.io/docs/reference/access-authn-authz/admission-controllers/#noderestriction) (#148).

## [1.13.8] - 2019-03-08

### Changed
- Correct kube-proxy flags to handle load balancers with `externalTrafficPolicy=Local` (#139).
- Retry image pulling to be more robust (#140).

## [1.13.7] - 2019-03-07

### Added
- CNI configuration can be specified in `cluster.yml` (#136).

### Changed
- Fix a bug that prevents kubelet to be restarted cleanly (#138).

## [1.13.6] - 2019-03-07

### Changed
- Update Kubernetes to 1.13.4 (#137).
- Apply kube-proxy patch to fix kubernetes/kubernetes#72432 (#137).

## [1.13.5] - 2019-03-01

### Changed
- Remove the step to pull `pause` container image (#135).

## [1.13.4] - 2019-02-26

### Added
- Support remote runtime for kubernetes pod (#133).
- Support log rotation of remote runtime for kubelet configuration (#133).

## [1.13.3] - 2019-02-12

### Added
- Add audit log support (#130).

### Changed
- Fix removing node resources if hostname in cluster.yaml is specified (#129).

## [1.13.2] - 2019-02-07

### Added
- [FAQ](./docs/faq.md).

### Changed
- `ckecli ssh` does not look for the node in `cluster.yml` (#127).
- kubelet reports OS information correctly (#128).
- When kubelet restarts, OOM score adjustment did not work (#128).
- Specify rshared mount option instead of shared for /var/lib/kubelet (#128).

## [1.13.1] - 2019-02-06

### Changed
- Logs from Kubernetes programs (apiserver, kubelet, ...) and etcd are sent to journald (#126).

## [1.13.0] - 2019-01-25

### Changed
- Support for kubernetes 1.13 (#125).
- Update etcd to 3.3.11, CoreDNS to 1.3.1, unbound to 1.8.3.

## Ancient changes

See [CHANGELOG-1.12](./CHANGELOG-1.12.md).

[Unreleased]: https://github.com/cybozu-go/cke/compare/v1.13.10...HEAD
[1.13.10]: https://github.com/cybozu-go/cke/compare/v1.13.9...v1.13.10
[1.13.9]: https://github.com/cybozu-go/cke/compare/v1.13.8...v1.13.9
[1.13.8]: https://github.com/cybozu-go/cke/compare/v1.13.7...v1.13.8
[1.13.7]: https://github.com/cybozu-go/cke/compare/v1.13.6...v1.13.7
[1.13.6]: https://github.com/cybozu-go/cke/compare/v1.13.5...v1.13.6
[1.13.5]: https://github.com/cybozu-go/cke/compare/v1.13.4...v1.13.5
[1.13.4]: https://github.com/cybozu-go/cke/compare/v1.13.3...v1.13.4
[1.13.3]: https://github.com/cybozu-go/cke/compare/v1.13.2...v1.13.3
[1.13.2]: https://github.com/cybozu-go/cke/compare/v1.13.1...v1.13.2
[1.13.1]: https://github.com/cybozu-go/cke/compare/v1.13.0...v1.13.1
[1.13.0]: https://github.com/cybozu-go/cke/compare/v1.12.0...v1.13.0
