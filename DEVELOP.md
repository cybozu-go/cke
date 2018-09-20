How to develop CKE
==================

## Go environment

Use Go 1.11 or higher.

CKE uses [Go modules](https://github.com/golang/go/wiki/Modules) to manage dependencies.
So you must either set `GO111MODULE=on` environment variable or checkout CKE out of `GOPATH`.

## Update dependencies

To update a dependency, just do:

```console
$ go get -d github.com/foo/bar@v1.2.3
```

To update all dependencies for bug fixes, do:

```console
$ go get -u=patch ./...
```

To update all dependencies for new features, do:

```console
$ go get -u ./...
```

After updating dependencies, run following commands to vendor dependencies:

```console
$ go mod tidy
$ go mod vendor
$ git add -f vendor
$ git add go.mod go.sum
$ git commit
```

### Update `k8s.io` modules

Modules under `k8s.io` such as `k8s.io/client-go` do not have fixed tags
and therefore are incompatible with Go modules.

For this reason, CKE specifies dependencies with commit hashes.

Specifically for Kubernetes 1.11, we picked hashes from these trees:

* client-go: https://github.com/kubernetes/client-go/tree/release-8.0
* api: https://github.com/kubernetes/api/tree/release-1.11
* apimachinery: https://github.com/kubernetes/apimachinery/tree/release-1.11

Pick the HEAD commits for each tree and `go get` as follows:

```console
$ go get -d k8s.io/client-go@HASH
$ go get -d k8s.io/api@HASH
$ go get -d k8s.io/apimachinery@HASH
```
