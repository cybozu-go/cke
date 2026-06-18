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

PullImage behaviour
-------------------

Before pulling an image, CKE checks whether a suitable image is already present on the node using `docker image list --format '{{.Repository}}:{{.Tag}}@{{.Digest}}'`.

Each line of the output is compared against two conditions:

1. **FullRef match** — the line equals `img.FullRef()` (e.g. `ghcr.io/cybozu/etcd:3.6.11.1@sha256:...`).  
   This is the normal case after an image has been pulled from a registry.

2. **No-digest match** — the line equals `img.TagRef()+"@<none>"` (e.g. `ghcr.io/cybozu/etcd:3.6.11.1@<none>`).  
   This covers images loaded via `docker load` from a tar archive, which have a tag but no RepoDigest.

If neither condition is met (including when the tag matches but the digest differs), the image is considered absent and the following steps are executed:

1. `docker image pull <FullRef>` — pulls the image by digest. Docker stores it with `<none>` as the tag.
2. `docker image tag <FullRef> <TagRef>` — assigns the tag so the image can be addressed by `TagRef` in subsequent `docker run` calls.

Running containers
------------------

All `docker run` invocations use:

- `--pull=never` — prevents Docker from attempting a pull at run time; the image must already be present from `PullImage`.
- `TagRef` as the image argument — works for both registry-pulled images (which have the tag) and `docker load` images (which lack a RepoDigest and cannot be addressed by digest).

Air-gap environments
--------------------

In air-gapped environments, images are pre-loaded onto nodes via `docker load` from a tar archive. These images have a tag but no RepoDigest.

CKE handles this as follows:

1. `PullImage` detects the no-digest match and skips the pull.
2. `docker run` addresses the image by `TagRef`, which succeeds because the tag is present.
