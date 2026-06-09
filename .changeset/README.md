# Changesets

This directory holds [changesets](https://github.com/changesets/changesets): one
markdown file per set of changes, declaring how they bump the version and a
human-readable summary that becomes the `CHANGELOG.md` entry.

`transitland-lib` is a Go project; changesets is used only to track changes and
drive versioning/releases. Versions stay continuous with the existing `vX.Y.Z`
git tags (changesets uses the `v`-prefixed format for single-package repos).

## Adding a changeset (in your feature PR)

```bash
pnpm changeset
```

Pick the bump level (patch / minor / major) and write a short summary. Commit
the generated `.changeset/<name>.md` file with your PR.

## What happens next (automated in CI)

1. When your PR merges to `main`, a bot opens/updates a **"Version Packages"** PR
   that consumes pending changesets, bumps `package.json`, and writes
   `CHANGELOG.md`.
2. Merging that PR creates the `vX.Y.Z` git tag and GitHub Release, which builds
   and ships the binaries.

You do not run `changeset version` or `changeset tag` by hand; CI does.

See [`RELEASING.md`](../RELEASING.md) for the full flow and the dependency
hardening policy.
