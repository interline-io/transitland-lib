# Releasing transitland-lib

Releases are driven by [changesets](https://github.com/changesets/changesets).
Versioning is continuous with the historical `vX.Y.Z` git tags — changesets uses
the `v`-prefixed tag format for single-package repositories, so the series simply
continues (`v1.3.3` → `v1.3.4` → ...).

The Go module version is still embedded at build time from the git tag (via
`-ldflags -X main.tag=...`, read in `version.go`); `package.json` is only the
bookkeeping the changesets tooling reads. The two stay in lockstep because the
release tag is derived from the `package.json` version.

## Day-to-day: add a changeset to your PR

When a PR makes a user-visible change, record it:

```bash
pnpm changeset
```

Choose the bump (patch / minor / major) and write a short summary. Commit the
generated `.changeset/<name>.md` alongside your code. PRs without a changeset are
fine for changes that don't warrant a release (CI, docs, internal refactors).

## How a release happens (automated)

Everything runs in `.github/workflows/release.yml`, which fires only after the
**Test Suite** passes on `main`:

1. **Merge feature PRs.** Each merge to `main` (after tests pass) triggers the
   release workflow. While unreleased changesets exist, it opens/updates a
   **"Version Packages"** PR that bumps `package.json` and writes `CHANGELOG.md`.
2. **Merge the "Version Packages" PR.** After tests pass on that merge, the
   workflow sees the new version has no tag yet, then builds + signs the Linux
   and macOS binaries, creates the `vX.Y.Z` tag and GitHub Release (notes pulled
   from `CHANGELOG.md`, binaries attached), and dispatches the Homebrew formula
   update.

You never run `changeset version`, `changeset tag`, or create the Release by
hand — CI does. The built-in `GITHUB_TOKEN` covers all in-repo steps; the GitHub
App token is used only for the cross-repo Homebrew dispatch.

> Note: the auto-generated "Version Packages" PR does not get its own Test Suite
> run (pushes made with `GITHUB_TOKEN` don't trigger workflows). It only edits
> `package.json`, `CHANGELOG.md`, and `.changeset/*`; the full suite still runs on
> `main` before anything is built or released.

## Major versions (v2+) — read before choosing a `major` bump

`transitland-lib` is both a CLI and a **Go library** imported by other projects,
so Go module rules apply. A `major` changeset will bump to `2.0.0` and tag
`v2.0.0` — but for Go modules **a v2+ release also requires changing the module
path**, which is not automatic:

- Update `module github.com/interline-io/transitland-lib` →
  `.../transitland-lib/v2` in `go.mod`.
- Update internal imports of the module path accordingly.
- Consumers must then `import ".../transitland-lib/v2"` and `go get .../v2`.

Without the `/vN` suffix, `go get` cannot resolve `v2.0.0` and downstream builds
break. Because changesets makes a major bump look as cheap as any other, treat
`major` as a deliberate, separately-reviewed change that includes the module-path
move (and ideally a migration note in the changelog). For `v0`/`v1` this does not
apply.

Never delete or re-point a published tag — consumers may depend on it.

## Build provenance & verification

Released binaries carry [SLSA build provenance](https://docs.github.com/en/actions/concepts/security/artifact-attestations)
(keyless, via GitHub OIDC + Sigstore) and a `SHA256SUMS` asset. To verify a
downloaded binary:

```bash
gh attestation verify ./transitland-linux --repo interline-io/transitland-lib
# and/or
sha256sum -c SHA256SUMS
```

## Dependency / supply-chain policy

The changesets tooling is third-party JavaScript and is treated as untrusted:

- **All GitHub Actions are pinned to commit SHAs** (with a `# vX.Y.Z` comment).
  Bump them deliberately, never via floating tags.
- **Exact dependency versions** in `package.json` (no `^`/`~`), with a committed
  `pnpm-lock.yaml` (integrity-hashed). CI installs with
  `pnpm install --frozen-lockfile --ignore-scripts`.
- **Cooldown:** `minimumReleaseAge` in `pnpm-workspace.yaml` keeps freshly
  published versions out of the lockfile for a few days, so a compromised release
  has time to be caught/yanked before we can pull it in. It applies when
  resolving (local `pnpm install` / `pnpm update`), not to frozen CI installs.
- **No install scripts:** `onlyBuiltDependencies: []` (pnpm) plus
  `ignore-scripts=true` (`.npmrc`) prevent dependency lifecycle scripts from
  running.

To update the release tooling, run `pnpm update` locally (the cooldown will hold
back anything too new), review the lockfile diff, and open a normal PR.
