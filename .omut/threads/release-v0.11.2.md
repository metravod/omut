---
status: done
updated: 2026-11-12
by: zcode/glm-5.3
---
# Release v0.11.2
**Goal:** ship the degraded-mode fallback as a patch: the pointer block and the rules tell the agent about omut itself — `omut hook start` when the CLI is there, an install suggestion (brew or install.sh, no Go) when it is not.
**Where we got:** work commit 85099bf, release-prep 5e58df7, annotated v0.11.2 on it. Published release: https://github.com/metravod/omut/releases/tag/v0.11.2.
**Verified:** final go vet ./... and go test ./... pass; strict check clean; built-binary zero dogfood and fresh-start smoke show the new bullet. Release workflow 43227595632 green (macos/test, goreleaser); six archives, checksums.txt matches the downloaded darwin/arm64 archive; cask 0.11.2. Released binary reports 0.11.2, preview works, fresh start carries the install bullet. This machine upgraded via brew; zero across the pack updated the root LEAD.md and five projects' blocks/READMEs.
**Not verified:** the other five released binaries; install.sh against 0.11.2 (CI's install job covers it).
**Open:** zymi-core has a pre-existing shape error (`status "open"` in threads/multiagent-supervision-idea.md:2; allowed: active, parked, done, dropped) — for its human, not this release.
**Next step:** none.
