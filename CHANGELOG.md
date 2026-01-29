# Change Log

All notable changes to this project will be documented in this file.
This project employs a versioning scheme described in [RELEASE.md](RELEASE.md#versioning).


## [Unreleased]

### Changed

- Update docker-compose.yml for the new release in [#845](https://github.com/cybozu-go/cke/pull/845)

## [1.33.1]

### Changed

- Implement rolling update for kube-apiservers in [#824](https://github.com/cybozu-go/cke/pull/824)

## [1.33.0]

### Changed

- Support Kubernetes 1.33 in [#841](https://github.com/cybozu-go/cke/pull/841)
    - Update Go modules and GitHub Actions in [#840](https://github.com/cybozu-go/cke/pull/840)
    - Update Sabakan mock in [#840](https://github.com/cybozu-go/cke/pull/840)
    - Disable deprecation check for Endpoints temporarily in [#840](https://github.com/cybozu-go/cke/pull/840)
    - Update Ubuntu of CKE container base to 24.04 in [#841](https://github.com/cybozu-go/cke/pull/841)
    - Update miscellaneous tools in [#841](https://github.com/cybozu-go/cke/pull/841)
    - Update containerd for mtest to 2.2.1 in [#841](https://github.com/cybozu-go/cke/pull/841)
    - Enable coordinated leader election by default in [#841](https://github.com/cybozu-go/cke/pull/841)
    - Update ClusterRole `system:kube-apiserver-to-kubelet` for kubelet fine-grained authorization in [#841](https://github.com/cybozu-go/cke/pull/841)
- Change key transferring method for `ckecli ssh` and `ckecli scp`in [#815](https://github.com/cybozu-go/cke/pull/815)

### [update since 1.33.0-rc.1](https://github.com/cybozu-go/cke/compare/v1.33.0-rc.1...v1.33.0)

#### Changed

- Change key transferring method for `ckecli ssh` and `ckecli scp`in [#815](https://github.com/cybozu-go/cke/pull/815)

## [1.33.0-rc.1]

### Changed

- Support Kubernetes 1.33 in [#841](https://github.com/cybozu-go/cke/pull/841)
- Update Go modules and GitHub Actions in [#840](https://github.com/cybozu-go/cke/pull/840)
- Update Sabakan mock in [#840](https://github.com/cybozu-go/cke/pull/840)
- Disable deprecation check for Endpoints temporarily in [#840](https://github.com/cybozu-go/cke/pull/840)
- Update Ubuntu of CKE container base to 24.04 in [#841](https://github.com/cybozu-go/cke/pull/841)
- Update miscellaneous tools in [#841](https://github.com/cybozu-go/cke/pull/841)
- Update containerd for mtest to 2.2.1 in [#841](https://github.com/cybozu-go/cke/pull/841)
- Enable coordinated leader election by default in [#841](https://github.com/cybozu-go/cke/pull/841)
- Update ClusterRole `system:kube-apiserver-to-kubelet` for kubelet fine-grained authorization in [#841](https://github.com/cybozu-go/cke/pull/841)

## Ancient changes

- See [release-1.32/CHANGELOG.md](https://github.com/cybozu-go/cke/blob/release-1.32/CHANGELOG.md) for changes in CKE 1.32.
- See [release-1.31/CHANGELOG.md](https://github.com/cybozu-go/cke/blob/release-1.31/CHANGELOG.md) for changes in CKE 1.31.
- See [release-1.30/CHANGELOG.md](https://github.com/cybozu-go/cke/blob/release-1.30/CHANGELOG.md) for changes in CKE 1.30.
- See [release-1.29/CHANGELOG.md](https://github.com/cybozu-go/cke/blob/release-1.29/CHANGELOG.md) for changes in CKE 1.29.
- See [release-1.28/CHANGELOG.md](https://github.com/cybozu-go/cke/blob/release-1.28/CHANGELOG.md) for changes in CKE 1.28.
- See [release-1.27/CHANGELOG.md](https://github.com/cybozu-go/cke/blob/release-1.27/CHANGELOG.md) for changes in CKE 1.27.
- See [release-1.26/CHANGELOG.md](https://github.com/cybozu-go/cke/blob/release-1.26/CHANGELOG.md) for changes in CKE 1.26.
- See [release-1.25/CHANGELOG.md](https://github.com/cybozu-go/cke/blob/release-1.25/CHANGELOG.md) for changes in CKE 1.25.
- See [release-1.24/CHANGELOG.md](https://github.com/cybozu-go/cke/blob/release-1.24/CHANGELOG.md) for changes in CKE 1.24.
- See [release-1.23/CHANGELOG.md](https://github.com/cybozu-go/cke/blob/release-1.23/CHANGELOG.md) for changes in CKE 1.23.
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

[Unreleased]: https://github.com/cybozu-go/cke/compare/v1.33.1...HEAD
[1.33.1]: https://github.com/cybozu-go/cke/compare/v1.33.0...v1.33.1
[1.33.0]: https://github.com/cybozu-go/cke/compare/v1.32.6...v1.33.0
[1.33.0-rc.1]: https://github.com/cybozu-go/cke/compare/v1.32.6...v1.33.0-rc.1
