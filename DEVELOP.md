How to develop CKE
==================

## Go environment

Use Go 1.21 or higher.

## Starting development for a new Kubernetes minor release

Each CKE release corresponds to a Kubernetes version.
For example, CKE 1.16.x corresponds to Kubernetes 1.16.x.

When we start development for a new Kubernetes minor release on the `main` branch,
create a maintenance branch for the previous Kubernetes minor release.
For example, when we start development for Kuberntes 1._17_, create and push `release-1.16`
branch as follows:

```console
$ git fetch origin
$ git checkout -b release-1.16 origin/main
$ git push -u origin release-1.16
```

Then, clear the change log entries in `CHANGELOG.md`.

### Update `k8s.io` modules

CKE uses `k8s.io/client-go`.

Modules under `k8s.io` are compatible with Go modules.
Therefore, when `k8s.io/client-go` is updated as follows, dependent modules are also updated.

```console
$ VERSION=v0.17.4
$ go get -d k8s.io/client-go@${VERSION} k8s.io/api@${VERSION} k8s.io/apimachinery@${VERSION} \
            k8s.io/apiserver@${VERSION} k8s.io/kube-scheduler@${VERSION} k8s.io/kubelet@${VERSION} \
            k8s.io/kube-proxy@${VERSION}
```

### Update the Kubernetes resource definitions embedded in CKE

The Kubernetes resource definitions embedded in CKE is defined in `./static/resource.go`.
This needs to be updated by `make static` whenever `images.go` updates.

### Update `cke-tools`

Edit [`tools/Makefile`](tools/Makefile) and update `CNI_PLUGIN_VERSION` for the [latest release](https://github.com/containernetworking/plugins/releases).
Edit [`tools/CHANGELOG.md`](tools/CHANGELOG.md) to prepare the new version.

After these changes are merged, create and push a tag like `tools-1.17.0`.
Read [`tools/RELEASE.md`](tools/RELEASE.md) for details.

## Back-porting fixes

When vulnerabilities or critical issues are found in the main branch, 
consider back-porting the fixes to older branches as follows:

```
$ git checkout release-1.16
$ git cherry-pick <commit from main>
```
