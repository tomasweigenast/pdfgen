# Releasing pdfgen

## Release

Use Conventional Commit messages. Release Please opens or updates a release PR after
changes reach `main`. Review the version and changelog, then merge that PR. The workflow
publishes the GitHub release and `ghcr.io/tomasweigenast/pdfgen`.

- `feat:` bumps the minor version.
- `fix:` and `perf:` bump the patch version.
- `!` or a `BREAKING CHANGE:` footer bumps the minor version before 1.0 and the major
  version from 1.0 onward.
- `docs:`, `test:`, `refactor:`, `build:`, `ci:` and `chore:` do not release by
  themselves.

The first release is `v0.1.0`. To graduate to 1.0, add `Release-As: 1.0.0` to a
releasable commit. For a hotfix, merge a `fix:` commit into `main` and merge its release
PR.

## Docker tags

Stable releases get a version tag. They also update minor, major and `latest` aliases
where applicable:

- `1.2.3`: `1.2.3`, `1.2`, `1`, `latest`
- `0.2.3`: `0.2.3`, `0.2`, `latest`
- `1.0.0-rc.1`: `1.0.0-rc.1` only

The image supports Linux AMD64 and ARM64. It runs as a non-root user; mounted output
directories must be writable by the selected user.

## Recover a failed publication

Run the Release workflow from `main` with an existing tag:

```sh
gh workflow run release-please.yml --ref main -f tag=v0.1.0 -f publisher=all
```

Use `publisher=binaries` or `publisher=docker` to retry one publisher. For a partial
binary upload, pass the failed run ID to reuse its saved build files:

```sh
gh workflow run release-please.yml --ref main \
  -f tag=v0.1.0 -f publisher=binaries -f recovery_run_id=123456789
```

Recovery checks the tag, commit and assets. Matching files are kept, missing files are
uploaded, and conflicting files stop the run. Docker recovery reuses the version image
and does not move aliases back to an older version.

If a release is defective, fix it in a new release. Do not delete or recreate published
tags.
