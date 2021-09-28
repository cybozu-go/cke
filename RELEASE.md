Release procedure
=================

This document describes how to release a new version of cke.

## Versioning

Given a version number MAJOR.MINOR.PATCH.
The MAJOR and MINOR version matches that of Kubernetes.
The patch version is increased with CKE update.

## Prepare change log entries

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

## Bump version

1. Determine a new version number.  Let it write `$VERSION` as `VERSION=x.y.z`.
2. Make a branch to release

    ```console
    $ git neco dev "$VERSION"
    ```

3. Update `version.go`.
4. Edit `CHANGELOG.md` for the new version ([example][]).
5. Commit the change and create a pull request.

    ```console
    $ git commit -a -m "Bump version to $VERSION"
    $ git neco review
    ```

6. Make sure that Sonobuoy test has been passed when updating to `x.y.0` and its RC.
7. Merge the pull request.
8. Add a git tag to the main HEAD, then push it.

    ```console
    $ git checkout main
    $ git pull
    $ git tag -a -m "Release v$VERSION" "v$VERSION"
    $ git push origin "v$VERSION"
    ```

Then GitHub Actions automatically builds and pushes the tagged container image to [quay.io](https://quay.io/cybozu/cke).

GitHub Actions also creates a GitHub release automatically after running [sonobuoy](./sonobuoy) tests.
So, **DO NOT MANUALLY CREATE GITHUB RELEASES**.  The test results will be attached to the GitHub
release that can be submitted to [cncf/k8s-conformance](https://github.com/cncf/k8s-conformance).

## Maintain docker-compose

After new CKE released, update cke image on docker-compose.yml.

[example]: https://github.com/cybozu-go/etcdpasswd/commit/77d95384ac6c97e7f48281eaf23cb94f68867f79
