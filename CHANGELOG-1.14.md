# Change Log

All notable changes to this project will be documented in this file.
This project employs a versioning scheme described in [RELEASE.md](RELEASE.md#versioning).

## [1.14.15] - 2019-08-20

### Changed
- Backport #219: sabakan: update for gqlgen 0.9+

## [1.14.14] - 2019-08-15

### Changed
- Fix ineffassign errors (#205)
- Close etcd session and enable ineffassign check (#207)
- [ckecli] add options to `ckecli kubernetes issue` command (#206)
- [vault] Check secret before SetToken() (#209)

## [1.14.13] - 2019-08-01

### Changed
- Fix a bug that prevents Node resource creation in some cases (#203)
- Maintain default/kubernetes Endpoints by CKE itself (#204)

## [1.14.12] - 2019-07-19

### Added

- sabakan: weighted selection of node roles (#200)

## [1.14.11] - 2019-07-16

### Added

- Labels and taints for control plane nodes (#199)

## [1.14.10] - 2019-07-12

### Changed

- Fix bug on getting status.Scheduler, again (#198)

## [1.14.9] - 2019-07-11

### Changed

- Rename to recommended label keys (#197).

## [1.14.8] - 2019-07-09

### Added

- Invoke vault tidy periodically (#196).

### Fixed

- log: be silent when checking scheduler status (#195).
- mtest: use docker instead of podman (#194).

## [1.14.7] - 2019-06-28

### Fixed

- Fix bug on getting status.Scheduler (#193)

## [1.14.6] - 2019-06-27

### Added

- Add scheduler extender configurations in cluster.yml (#191, #192)

## [1.14.5] - 2019-06-14

### Added

- Add `ckecli sabakan enable` (#190).

## [1.14.4] - 2019-06-05

### Changed

- Fix `ckecli vault init` for newer vault API, and test re-init by mtest (#187).

## [1.14.3] - 2019-06-04

### Action required

- Updating of the existing installation requires re-invocation of `ckecil vault init`
    to add a new CA to Vault.

### Added

- Enable [API aggregation layer](https://kubernetes.io/docs/tasks/access-kubernetes-api/configure-aggregation-layer/) (#186).

## [1.14.2] - 2019-06-04

### Changed

- Fix a bug to stop CKE when it fails to connect to vault (#185).

## [1.14.1] - 2019-05-28

### Added
- Add `etcd-rivers` as a reverse proxy for k8s etcd (#181).
- Add `--follow` option to `ckecli history` (#180).

### Changed
- Apply nilerr and restrictpkg to test (#176).
- Add a cke container test to mtest (#175).
- Refine the output of `ckecli history` (#170).
- Fix handling etcd API `WithLimit` (#177).
- Fix dockerfile for podman v1.3.2-dev (#182).

## [1.14.0] - 2019-04-22

No user-visible changes since RC 1.

## [1.14.0-rc1] - 2019-04-19

### Changed
- Update kubernetes to 1.14.1
- Update CoreDNS to 1.5.0
- Update CNI plugins to 0.7.5


## Ancient changes

See [CHANGELOG-1.13](./CHANGELOG-1.13.md).
See [CHANGELOG-1.12](./CHANGELOG-1.12.md).

[1.14.15]: https://github.com/cybozu-go/cke/compare/v1.14.14...v1.14.15
[1.14.14]: https://github.com/cybozu-go/cke/compare/v1.14.13...v1.14.14
[1.14.13]: https://github.com/cybozu-go/cke/compare/v1.14.12...v1.14.13
[1.14.12]: https://github.com/cybozu-go/cke/compare/v1.14.11...v1.14.12
[1.14.11]: https://github.com/cybozu-go/cke/compare/v1.14.10...v1.14.11
[1.14.10]: https://github.com/cybozu-go/cke/compare/v1.14.9...v1.14.10
[1.14.9]: https://github.com/cybozu-go/cke/compare/v1.14.8...v1.14.9
[1.14.8]: https://github.com/cybozu-go/cke/compare/v1.14.7...v1.14.8
[1.14.7]: https://github.com/cybozu-go/cke/compare/v1.14.6...v1.14.7
[1.14.6]: https://github.com/cybozu-go/cke/compare/v1.14.5...v1.14.6
[1.14.5]: https://github.com/cybozu-go/cke/compare/v1.14.4...v1.14.5
[1.14.4]: https://github.com/cybozu-go/cke/compare/v1.14.3...v1.14.4
[1.14.3]: https://github.com/cybozu-go/cke/compare/v1.14.2...v1.14.3
[1.14.2]: https://github.com/cybozu-go/cke/compare/v1.14.1...v1.14.2
[1.14.1]: https://github.com/cybozu-go/cke/compare/v1.14.0...v1.14.1
[1.14.0]: https://github.com/cybozu-go/cke/compare/v1.14.0-rc1...v1.14.0
[1.14.0-rc1]: https://github.com/cybozu-go/cke/compare/v1.13.18...v1.14.0-rc1
