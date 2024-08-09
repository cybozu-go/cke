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

1. Determine a new version number. Then set `VERSION` variable.

    ```console
    # Set VERSION and confirm it. It should not have "v" prefix.
    $ VERSION=x.y.z
    $ echo $VERSION
    ```

2. Make a branch to release

    ```console
    $ git neco dev "bump-$VERSION"
    ```

3. Update `version.go`.
4. Edit `CHANGELOG.md` for the new version ([example][]).
5. Commit the change and create a pull request.

    ```console
    $ git commit -a -m "Bump version to $VERSION"
    $ git neco review
    ```

6. When updating to `x.y.0` or its RC, run Sonobuoy test manually and make sure that it has been passed.
7. Merge the pull request.
8. Add a git tag to the main HEAD, then push it.

    ```console
    # Set VERSION again.
    $ VERSION=x.y.z
    $ echo $VERSION

    $ git checkout main
    $ git pull
    $ git tag -a -m "Release v$VERSION" "v$VERSION"

    # Make sure the release tag exists.
    $ git tag -ln | grep $VERSION

    $ git push origin "v$VERSION"
    ```

Then GitHub Actions automatically builds and pushes the tagged container image to [ghcr.io](https://github.com/cybozu-go/cke/pkgs/container/cke).

GitHub Actions also creates a GitHub release automatically after running [sonobuoy](./sonobuoy) tests.
So, **DO NOT MANUALLY CREATE GITHUB RELEASES**.  The test results will be attached to the GitHub
release that can be submitted to [cncf/k8s-conformance](https://github.com/cncf/k8s-conformance).

## Maintain docker compose

After new CKE released, update cke image on docker-compose.yml.

[example]: https://github.com/cybozu-go/etcdpasswd/commit/77d95384ac6c97e7f48281eaf23cb94f68867f79
