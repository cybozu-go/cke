# Change Log

All notable changes to this project will be documented in this file.
This project adheres to [versioning](RELEASE.md#versioning).

## [Unreleased]

### Changed
- `ckecli ssh` does not look for the node in `cluster.yml` (#127).

## [1.13.1] - 2019-02-06

### Changed
- Logs from Kubernetes programs (apiserver, kubelet, ...) and etcd are sent to journald (#126).

## [1.13.0] - 2019-01-25

### Changed
- Support for kubernetes 1.13 (#125).
- Update etcd to 3.3.11, CoreDNS to 1.3.1, unbound to 1.8.3.

## Ancient changes

See [CHANGELOG-1.12](./CHANGELOG-1.12.md).

[Unreleased]: https://github.com/cybozu-go/cke/compare/v1.13.1...HEAD
[1.13.1]: https://github.com/cybozu-go/cke/compare/v1.13.0...v1.13.1
[1.13.0]: https://github.com/cybozu-go/cke/compare/v1.12.0...v1.13.0
