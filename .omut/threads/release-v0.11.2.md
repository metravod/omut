---
status: active
updated: 2026-11-12
by: zcode/glm-5.3
---
# Release v0.11.2
**Goal:** ship the degraded-mode fallback as a patch: the pointer block and the rules tell the agent about omut itself — `omut hook start` when the CLI is there, an install suggestion (brew or install.sh, no Go) when it is not.
**Where we got:** work commit 85099bf; tag v0.11.2 pending on the release-prep commit; release.yml runs macOS tests, goreleaser and the cask.
**Verified:** go vet ./... and go test ./... pass; `check --fail-on warn` clean; built-binary zero dogfood on this repository (block and .omut/README.md updated) and a fresh-start smoke show the new bullet. Release evidence to be filled after the workflow.
**Not verified:** the release workflow and the published artifacts, yet.
**Open:** tag, push, watch release.yml, upgrade this machine via brew, zero across the pack, smoke the released binary.
**Next step:** tag v0.11.2 and push it.
