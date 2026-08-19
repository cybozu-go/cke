# Change Log

All notable changes to this project will be documented in this file.
This project employs a versioning scheme described in [RELEASE.md](RELEASE.md#versioning).


## [Unreleased]

### Changed

- stop etcd compaction itself in [#913](https://github.com/cybozu-go/cke/pull/913)

    **Upgrade note**: etcd permits the Compact API only for users having the `root`
    role, and CKE grants the role to the `kube-apiserver` user only when it
    bootstraps an etcd cluster.  If your etcd cluster was bootstrapped by CKE older
    than v1.35.0, `kube-apiserver` has never been able to compact, and nothing
    compacts the keyspace once etcd's auto-compaction is disabled by this release.
    The keyspace then keeps growing until etcd reaches its backend quota and stops
    accepting writes.  Grant the role **before** upgrading to this release:

    ```console
    $ etcdctl user get kube-apiserver               # "root" must be in Roles
    $ etcdctl user grant-role kube-apiserver root   # grant it if missing
    ```

    Note also that etcd members are restarted one by one to apply the new
    parameters.  Read [docs/etcd.md](docs/etcd.md#compaction) about compaction.
- Add `ckecli repair-queue wait` command in [#910](https://github.com/cybozu-go/cke/pull/910)
- **BREAKING** `ckecli kubernetes issue` now issues certificates for `cke:user:admin` instead of `admin`, so that audit logs can tell operators from CKE itself (pass `--user=admin` to restore the old name) in [#911](https://github.com/cybozu-go/cke/pull/911)

## [1.35.3]

### Changed

- Add feature to delete Job-managed Pod in [#903](https://github.com/cybozu-go/cke/pull/903)
- Improve CI in [#904](https://github.com/cybozu-go/cke/pull/904)

## [1.35.2]

### Changed

- Add feature to repair control-plane in [#896](https://github.com/cybozu-go/cke/pull/896)

## [1.35.1]

### Changed

- Check volumesInUse in CheckDrainCompletion in [#890](https://github.com/cybozu-go/cke/pull/890)

## [1.35.0]

### Changed

- update etcd to 3.6.11 in [#877](https://github.com/cybozu-go/cke/pull/877)

## [1.35.0-rc.1]

### Changed

- Support Kubernetes 1.35 in [#873](https://github.com/cybozu-go/cke/pull/873)
  - Update Go modules and GitHub Actions in [#871](https://github.com/cybozu-go/cke/pull/871)

## Ancient changes

- See [release-1.34/CHANGELOG.md](https://github.com/cybozu-go/cke/blob/release-1.34/CHANGELOG.md) for changes in CKE 1.34.
- See [release-1.33/CHANGELOG.md](https://github.com/cybozu-go/cke/blob/release-1.33/CHANGELOG.md) for changes in CKE 1.33.
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

[Unreleased]: https://github.com/cybozu-go/cke/compare/v1.35.3...HEAD
[1.35.3]: https://github.com/cybozu-go/cke/compare/v1.35.2...v1.35.3
[1.35.2]: https://github.com/cybozu-go/cke/compare/v1.35.1...v1.35.2
[1.35.1]: https://github.com/cybozu-go/cke/compare/v1.35.0...v1.35.1
[1.35.0]: https://github.com/cybozu-go/cke/compare/v1.35.0-rc.1...v1.35.0
[1.35.0-rc.1]: https://github.com/cybozu-go/cke/compare/v1.34.2...v1.35.0-rc.1
