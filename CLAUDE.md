# Working on pdfgen

pdfgen is a pure-Go HTML/CSS-to-PDF library and CLI. The module is
`github.com/tomasweigenast/pdfgen`; the CLI lives in `cmd/pdfgen`. Use the Go
version declared in `go.mod`. Keep release binaries CGO-free.

## Commits and pull requests

Use Conventional Commits for commits and PR titles:

- `feat: add a renderer option` releases a minor version.
- `fix: correct line wrapping` releases a patch version.
- `perf: reduce layout allocations` releases a patch version.
- `docs:`, `test:`, `refactor:`, `build:`, `ci:` and `chore:` do not release on
  their own. Use `chore(deps):` for routine dependency updates and `fix(deps):`
  for a dependency fix that affects users.
- Declare breaking changes with `!` after the type/scope, or a
  `BREAKING CHANGE:` footer explaining the change and migration. These release
  a minor version below 1.0 and a major version from 1.0 onward.

Prefer a merge commit to preserve the individual commit history. Give the PR a
Conventional Commit title, and make each releasable commit inside it Conventional
as well. Preserve `BREAKING CHANGE:` and intentional `Release-As:` footers in
the commit message. Keep changes focused. Do not commit generated `dist/` files, binaries,
credentials, or test output. Do not create commits, push, merge, or publish unless
requested by the user.

## Validation

Run `gofmt` on changed Go files, `go vet ./...`,
`staticcheck ./...` (pinned version in CI), and `go test ./...`. Tests need
Ghostscript and access to the OS font cache and localhost listeners. Report
blocked checks accurately. Preserve Linux, macOS, and Windows CI coverage.

For release changes also run:

```sh
node --test .github/scripts/release.test.mjs
# Install release-please@17.3.0 in a temporary npm prefix first:
RELEASE_TOOLS=/absolute/path/to/npm-prefix node .github/scripts/release-policy.test.cjs
goreleaser check
goreleaser release --snapshot --clean
actionlint -shellcheck=''
```

Use GoReleaser OSS v2.18.3 and Actionlint v1.7.12. CI additionally validates the
Release Please schema and builds and smoke-tests the container. Reference actions by their version tags (for example `actions/checkout@v4`).

## Releases

Release Please owns versions, tags, release PRs and release notes. The first
release is `v0.1.0`. Merge a releasable change into `main`, review the resulting
release PR and merge it to publish. Merge this implementation PR with a merge
commit and a Conventional Commit title such as
`feat: automate GitHub and container releases` to trigger initial onboarding.

Publication targets are GitHub Releases and GHCR only. Homebrew, Scoop and
Chocolatey are out of scope. Never create release tags manually, rewrite a
published tag, overwrite conflicting assets or move container aliases backwards.
Do not add a version file or convert `metadata.Version` to a constant: release
builds inject the version and `main.commit` with linker flags; `pdfgen.Version()`
and PDF Producer metadata retain their existing behavior.

The release workflow invokes publishers in the same run; do not add tag-push or
release-event publishing triggers. The caller holds the repository release
concurrency lock; taking that lock in a nested reusable workflow will deadlock.
Use job-level least privilege and keep checkout credentials disabled. Dispatch
inputs belong in environment variables or action inputs, never shell source.

Use `release-please-config.json` and `.github/workflows/` as the source of truth
for release automation and recovery behavior.
