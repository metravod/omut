---
status: active
updated: 2026-11-06
by: codex/GPT-6
---
# Release v0.10.0
**Goal:** publish optional work-plan support at the human's request, preserving the task package another agent is executing.
**Where we got:** Linux CI passed; the meta job found an archive-link spelling that only resolved when local evidence existed. Replaced it with the supported project-root local-layer reference before tagging.  v0.10.0 is free; remote main matches the local base 52b1a2c. Reviewed release workflows and changes. Pre-release review reproduced a macOS case-alias escape of the local-plan exclusion; directory identity comparison fixes it, with a regression case.
**Verified:** final go vet ./... and go test ./... pass, including the macOS case-alias regression (failed before fix). Built CLI passes check . --fail-on warn and git diff --check is clean. Optional-plan fixtures cover selection, provenance, privacy, worktrees, errors, budgets and read/zero preservation.
**Not verified:** pushed CI, tag/release assets, Homebrew metadata and released installer smoke.
**Open:** finish final checks before publishing; no writes to the other session's project or task package.
**Next step:** run final checks, commit and push main, verify CI, tag v0.10.0, verify release assets/Homebrew and checksum-verified installer.
