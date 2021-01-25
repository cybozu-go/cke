Release procedure
=================

This document describes how to release a new version of `cke-tools`.

## Versioning

Given a version number MAJOR.MINOR.PATCH.
The MAJOR and MINOR version matches that of Kubernetes.
The patch version is increased with `cke-tools` update.

## Publish Docker image to quay.io

1. Edit `CHANGELOG.md` in this directory.
2. Merge the `CHANGELOG.md` change to the `main` branch.
3. Tag the head commit as `tools-X.Y.Z` where `X.Y.Z` is the new semantic version of `cke-tools`.
4. Push the tag to GitHub.

CircleCI will build and push the new image as `quay.io/cybozu/cke-tools:X.Y.Z`.

[semver]: https://semver.org/spec/v2.0.0.html
