# Change Log

All notable changes to this project will be documented in this file.
This project employs a versioning scheme described in [RELEASE.md](RELEASE.md#versioning).


## [Unreleased]

### Changed

- Update dependencies for Kubernetes 1.36 in [#952](https://github.com/cybozu-go/cke/pull/952)
    - Go 1.27.4
    - Go modules (k8s.io/* v0.36.6, go.etcd.io/etcd/* v3.7.3, etc.)
    - GitHub Actions
    - Ubuntu 26.04
    - Container images
        - kubernetes 1.36.6.1
        - etcd 3.6.11.3
        - pause 3.10.2.1
        - unbound 1.26.2.2
        - unbound_exporter 0.6.0.1
    - Tools for mtest and sonobuoy
        - kubectl 1.36.6
        - crictl 1.36.0
        - containerd 2.4.2
        - sonobuoy 0.57.5
        - docker compose 5.7.0
        - cilium 1.20.4
- Update etcd, cke-tools and coredns images in [#955](https://github.com/cybozu-go/cke/pull/955)
    - etcd 3.6.17.1
    - cke-tools 1.36.0
    - coredns 1.15.0.1
    - etcd 3.6.17 for mtest
- Simplify the REST API server of cke in [#954](https://github.com/cybozu-go/cke/pull/954)
    - Replace well.HTTPServer with net/http.Server
    - Stop outputting access logs
    - Return 405 Method Not Allowed for non-GET requests
    - Use a fixed 2s timeout for /health instead of the etcd timeout in the config file
- Suppress deprecation warnings for Endpoints API requests in [#957](https://github.com/cybozu-go/cke/pull/957)
- Detect a wrong port name and duplicate addresses in Endpoints maintained by CKE in [#957](https://github.com/cybozu-go/cke/pull/957)

## Ancient changes

- See [release-1.35/CHANGELOG.md](https://github.com/cybozu-go/cke/blob/release-1.35/CHANGELOG.md) for changes in CKE 1.35.
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

[Unreleased]: https://github.com/cybozu-go/cke/compare/v1.35.8...HEAD
