---
status: done
updated: 2026-11-05
by: codex/GPT-6
---
# Release v0.8.0
**Goal:** commit and publish the prelaunch improvements with verified CI, assets and Homebrew metadata.
**Where we got:** implementation committed as c9baad2, pushed to main; annotated v0.8.0 points there. Published release: https://github.com/metravod/omut/releases/tag/v0.8.0. Release notes describe the new size warnings and refreshing project rules with omut zero.
**Verified:** go vet ./... and go test ./... locally; main CI and omut meta checks green. Release workflow 42009341931 passed macOS vet/tests and GoReleaser; six platform archives and checksums published. Homebrew cask is 0.8.0 with four platform hashes. install.sh downloaded and checksum-verified the macOS arm64 binary in a temporary directory; version is 0.8.0, own project check passes, and a long Cyrillic paragraph produces a size warning with failure under --fail-on warn.
**Not verified:** other released binaries run locally, or a Homebrew upgrade on this machine; no system installation was changed.
**Open:** none.
**Next step:** none.
