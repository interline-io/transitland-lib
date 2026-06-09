---
"transitland-lib": patch
---

Adopt changesets for versioning and a consolidated, automated release workflow (continuous with the existing vX.Y.Z tags). Hardened with SHA-pinned actions, a committed pnpm lockfile, an install cooldown, and no dependency lifecycle scripts. Releases now publish SLSA build provenance and a SHA256SUMS checksum file, and binaries are built with -trimpath.
