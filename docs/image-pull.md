Image Pull Specification
========================

Image reference format
----------------------

CKE manages container images using digest-pinned references in the form:

```
repository:tag@sha256:<digest>
```

All image constants (e.g. `EtcdImage`, `KubernetesImage`) are defined in this format and provide three accessors:

| Method | Returns |
|--------|---------|
| `FullRef()` | `repository:tag@sha256:<digest>` |
| `TagRef()` | `repository:tag` |
| `DigestRef()` | `repository@sha256:<digest>` |

PullImage behaviour
-------------------

Before pulling an image, CKE checks whether a suitable image is already present on the node using `docker image list --format '{{.Repository}}:{{.Tag}}@{{.Digest}}'`.

Each line of the output is compared against two conditions:

1. **FullRef match** — the line equals `img.FullRef()` (e.g. `ghcr.io/cybozu/etcd:3.6.11.1@sha256:...`).  
   This is the normal case after an image has been pulled from a registry.

2. **No-digest match** — the line equals `img.TagRef()+"@<none>"` (e.g. `ghcr.io/cybozu/etcd:3.6.11.1@<none>`).  
   This covers images loaded via `docker load` from a tar archive, which have a tag but no RepoDigest.

If neither condition is met (including when the tag matches but the digest differs), the image is considered absent and the following steps are executed:

1. `docker image pull <DigestRef>` — pulls the image by digest. Docker stores it with `<none>` as the tag.
2. `docker image tag <DigestRef> <TagRef>` — assigns the tag so the image can be addressed by `TagRef` in subsequent `docker run` calls.

Kubernetes resources
--------------------

Pods deployed by CKE (cluster-dns, node-dns) reference images by `FullRef`.
containerd resolves such a reference by digest only, so an image pre-loaded on a node is used only when it is named `repository@sha256:<digest>` with the registry digest.
Loading an OCI layout archive with `ctr images import` and adding that name with `ctr images tag` satisfies this; a Docker legacy archive does not keep the registry digest.

Running containers
------------------

Every `docker run` invocation first calls `PullImage` for its image, and then uses:

- `--pull=never` — prevents Docker from pulling by tag at run time; the image is present from `PullImage`.
- `TagRef` as the image argument — works for both registry-pulled images (which have the tag) and `docker load` images (which lack a RepoDigest and cannot be addressed by digest).

Outdated containers
-------------------

`RunSystem` records `FullRef` in the CKE label of the container, because docker reports only the reference given to `docker run`.
CKE compares that label with the desired image, so a container is restarted when the pinned digest changes even if the tag does not.
A container started by an older CKE has no such label and is compared by `TagRef`.

Air-gap environments
--------------------

In air-gapped environments, images are pre-loaded onto nodes via `docker load` from a tar archive. These images have a tag but no RepoDigest.

This holds for the classic (graphdriver) image store. With the containerd image store, `docker load` assigns a digest computed from the loaded archive, which differs from the registry digest. Such an image matches neither condition above and is pulled again from the registry.

CKE handles this as follows:

1. `PullImage` detects the no-digest match and skips the pull.
2. `docker run` addresses the image by `TagRef`, which succeeds because the tag is present.
