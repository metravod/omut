---
status: done
updated: 2026-11-06
by: codex/GPT-6
---
# Release v0.9.0
**Goal:** publish deterministic context assembly with verified CI, release assets and Homebrew metadata.
**Where we got:** the human requested the release. Research committed as 42294b8, context and release preparation as cc07cef; pushed to main. Annotated v0.9.0 points at cc07cef730caaedd9b11206c7a10114da66f4614. Published release and notes: https://github.com/metravod/omut/releases/tag/v0.9.0. Upgrade notes call for `omut zero` in the pack root; no metadata migration.
**Verified:** local `go vet ./...`, `go test ./...`, own meta check and diff whitespace pass. Main CI 42177890996 and omut check 42177890975 succeeded on the tagged commit. Release workflow 42178344299 succeeded, including macOS vet/tests and GoReleaser. Six platform archives and checksums are published; all six hashes match GitHub asset digests and the four Homebrew cask hashes match their assets, version 0.9.0. install.sh downloaded and checksum-verified macOS arm64; the released binary reports 0.9.0, passes own meta check, assembles project and six-project pack context, and supplies identical pack-hook context. Fresh fixtures verify embedded lead instructions, warning-only success and incomplete JSON with exit 1 on shape errors.
**Not verified:** running the other five released binaries locally, a Homebrew upgrade on this machine or a fresh native lead using the overview. The system installation and existing hook configuration were not changed. Token/task-quality gains remain unmeasured.
**Open:** none.
**Next step:** none.
