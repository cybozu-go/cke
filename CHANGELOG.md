# Change Log

All notable changes to this project will be documented in this file.
This project employs a versioning scheme described in [RELEASE.md](RELEASE.md#versioning).

## [Unreleased]

## [1.17.0] - 2020-04-01

No change from v1.17.0-rc.1.

## [1.17.0-rc.1] - 2020-03-31

### Changed
- Add new op for upgrading Kubelet without draining nodes (#304)
- Update etcd: v3.3.19.1 (#303)
- Update images for Kubernetes 1.17 (#302)
- Add label for each role (#300)
- Server Side Apply (#299)
    - Kubernetes 1.17.4
    - CNI plugins 0.8.5
    - CoreDNS 1.6.7
    - Unbound 1.10.0

## [1.16.4] - 2020-02-20

### Added
- Expose metrics to prometheus. (#292)

## [1.16.3] - 2020-02-07

### Changed
- Run Sonobuoy on multiple GCE instances. (#289)
- node-dns: use TCP to connect upstream. (#293)

## [1.16.2] - 2020-01-20

No change from v1.16.2-rc.1.

## [1.16.2-rc.1] - 2020-01-17

### Added
- Add preStop hook to wait for graceful termination. (#286)

### Changed
- Update images for Kubernetes 1.16.5 (#287)
  - hyperkube 1.16.5
  - CNI plugins 0.8.4

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

[Unreleased]: https://github.com/cybozu-go/cke/compare/v1.17.0...HEAD
[1.17.0]: https://github.com/cybozu-go/cke/compare/v1.17.0-rc.1...v1.17.0
[1.17.0-rc.1]: https://github.com/cybozu-go/cke/compare/v1.16.4...v1.17.0-rc.1
[1.16.4]: https://github.com/cybozu-go/cke/compare/v1.16.3...v1.16.4
[1.16.3]: https://github.com/cybozu-go/cke/compare/v1.16.2...v1.16.3
[1.16.2]: https://github.com/cybozu-go/cke/compare/v1.16.2-rc.1...v1.16.2
[1.16.2-rc.1]: https://github.com/cybozu-go/cke/compare/v1.16.1...v1.16.2-rc.1
[1.16.1]: https://github.com/cybozu-go/cke/compare/v1.16.0...v1.16.1
[1.16.0]: https://github.com/cybozu-go/cke/compare/v1.16.0-rc.3...v1.16.0
[1.16.0-rc.3]: https://github.com/cybozu-go/cke/compare/v1.16.0-rc.2...v1.16.0-rc.3
[1.16.0-rc.2]: https://github.com/cybozu-go/cke/compare/v1.16.0-rc.1...v1.16.0-rc.2
[1.16.0-rc.1]: https://github.com/cybozu-go/cke/compare/v1.15.7...v1.16.0-rc.1
