How to develop CKE
==================

## Go environment

Use Go 1.11.2 or higher.

CKE uses [Go modules](https://github.com/golang/go/wiki/Modules) to manage dependencies.
So you must either set `GO111MODULE=on` environment variable or checkout CKE out of `GOPATH`.

## Update dependencies

To update a dependency, just do:

```console
$ go get github.com/foo/bar@v1.2.3
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

Moreover, since `k8s.io/client-go` does not support Go modules yet,
Go fetches incompatible versions of `k8s.io/api` and `k8s.io/apimachinery`
that are direct dependencies of `k8s.io/client-go`.

To workaround the problems, we need to specify explicit versions
for these packages.  For Kubernetes 1.11, these branches should be
specified:

* client-go: [release-8.0](https://github.com/kubernetes/client-go/tree/release-8.0)
* api: [release-1.11](https://github.com/kubernetes/api/tree/release-1.11)
* apimachinery: [release-1.11](https://github.com/kubernetes/apimachinery/tree/release-1.11)

as follows:

```console
$ go get k8s.io/client-go@v8.0.0
$ go get k8s.io/api@release-1.11
$ go get k8s.io/apimachinery@release-1.11

$ go mod tidy
$ go mod vendor
$ git add -f vendor
$ git add go.mod
$ git commit
```
