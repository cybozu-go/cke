Release procedure
=================

This document describes how to release a new version of cke.

Versioning
----------

Given a version number MAJOR.MINOR.PATCH.
The MAJOR and MINOR version matches that of Kubernetes.
The patch version is increased with CKE update.


Prepare change log entries
--------------------------

Add notable changes since the last release to [CHANGELOG.md](CHANGELOG.md).
It should look like:

```markdown
(snip)
## [Unreleased]

### Added
- Implement ... (#35)

### Changed
- Fix a bug in ... (#33)

### Removed
- Deprecated `-option` is removed ... (#39)

(snip)
```

Bump version
------------

Note:
If kubernates MINOR version supported by CKE is updated, create a new branch `k8s-x.y`(kubernetes version x.y before update) before master merge.
   ```console
   $ git checkout -b k8s-x.y
   $ git push origin k8s-x.y
   ```
1. Determine a new version number.  Let it write `$VERSION`.
2. Checkout `master` branch.
3. Update `version.go`.
4. Edit `CHANGELOG.md` for the new version ([example][]).
5. Commit the change and add a git tag, then push them.

    ```console
    $ git commit -a -m "Bump version to $VERSION"
    $ git tag v$VERSION
    $ git push origin master --tags
    ```

Publish GitHub release page
---------------------------

Go to https://github.com/cybozu-go/cke/releases and edit the tag.
Finally, press `Publish release` button.

Publish Docker image in quay.io
-------------------------------

The `Dockerfile` for cke is hosted in [github.com/cybozu/neco-containers][].

1. Clone [github.com/cybozu/neco-containers][].
2. Edit `cke/Dockerfile`, `cke/TAG` and `cke/BRANCH` as in [this commit](https://github.com/cybozu/neco-containers/commit/463415b0430d03e822a3405662ccef3d18bfd213)
3. Once the change is merged in the master branch, CircleCI builds the container and uploads it to [quay.io](https://quay.io/cybozu/cke).

[example]: https://github.com/cybozu-go/etcdpasswd/commit/77d95384ac6c97e7f48281eaf23cb94f68867f79
[CircleCI]: https://circleci.com/gh/cybozu-go/etcdpasswd
[github.com/cybozu/neco-containers]: https://github.com/cybozu/neco-containers
