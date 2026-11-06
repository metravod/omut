---
status: active
updated: 2026-11-06
by: codex/GPT-6
---
# Release v0.9.0
**Goal:** publish deterministic context assembly with verified CI, release assets and Homebrew metadata.
**Where we got:** the human requested the release. v0.9.0 is the next minor version after v0.8.0. The implementation and completed handoff pilot are ready; keep the research in its own commit, then commit the context feature and release preparation. Release notes are prepared in local/.
**Verified:** local and remote main both start at 52c4601; latest published release is v0.8.0. The release workflow gates GoReleaser on macOS vet/tests, builds six archives and updates metravod/homebrew-tap. The prior implementation turn passed vet/tests, own meta checks and real-pack smoke.
**Not verified:** new main CI, tagged macOS gate, published v0.9.0 assets/cask and checksum-verified installation.
**Open:** none.
**Next step:** run the local gates, commit/push the prepared changes, wait for main CI/meta, push annotated v0.9.0, verify the release and installer, then record the final result.
