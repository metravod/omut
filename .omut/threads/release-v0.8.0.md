---
status: active
updated: 2026-11-05
by: codex/GPT-6
---
# Release v0.8.0
**Goal:** commit and publish the prelaunch improvements with verified CI, assets and Homebrew metadata.
**Where we got:** human authorized commit and release; v0.7.0 is the current published release. v0.8.0 adds Unicode size warnings, clarified setup/evidence boundaries and worked continuation evidence. Release is driven by a v* tag after main CI/meta checks; its workflow gates GoReleaser on macOS tests.
**Verified:** local checks passed during implementation; remote main matches the local origin/main. Prior local handoff commit is retained.
**Not verified:** current release commit, remote CI, tag workflow, published assets and updated tap.
**Open:** none; release authorized.
**Next step:** run the local gate, commit the explicit change set, push main, wait for CI/meta, then push v0.8.0 and verify release artifacts and tap.
