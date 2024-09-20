Release procedure
=================

This document describes how to release a new version of `cke-tools`.

## Versioning

Given a version number MAJOR.MINOR.PATCH.
The MAJOR and MINOR version matches that of Kubernetes.
The patch version is increased with `cke-tools` update.

## Bump version

1. Determine a new version number. Then set `VERSION` variable.

    ```console
    # Set VERSION and confirm it. It should not have "v" prefix.
    $ VERSION=x.y.z
    $ echo $VERSION
    ```

2. Make a branch to release

    ```console
    $ git checkout main
    $ git pull
    $ git checkout -b "bump-tools-$VERSION"
    ```

3. Edit `CHANGELOG.md` in this directory.
4. Commit the change and create a pull request.

    ```console
    $ git commit -a -m "Bump cke-tools version to $VERSION"
    $ git push -u origin HEAD
    $ gh pr create -f
    ```

5. Merge the pull request.
6. Add a git tag to the main HEAD, then push it.

    ```console
    # Set VERSION again.
    $ VERSION=x.y.z
    $ echo $VERSION

    $ git checkout main
    $ git pull
    $ git tag -a -m "Release tools-$VERSION" "tools-$VERSION"

    # Make sure the release tag exists.
    $ git tag -ln | grep "tools-$VERSION"

    $ git push origin "tools-$VERSION"
    ```

GitHub Actions will build and push the new image as `ghcr.io/cybozu-go/cke-tools:X.Y.Z`.

[semver]: https://semver.org/spec/v2.0.0.html
