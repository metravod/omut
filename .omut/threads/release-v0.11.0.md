---
status: done
updated: 2026-11-11
by: zcode/glm-5.3
---
# Release v0.11.0
**Goal:** publish memory dreaming at the human's request: `omut dream`, the volume-based `dreamed:` reminder, `hook start --preview`, stable-first output ordering.
**Where we got:** the other session's parked threads-introduction/memorilla work committed first (71105d0), then the feature (e7b9abe): internal/dream (counter, digest, contracts), cmd/omut/dream.go, the reminder and --preview in hook, stable-first reorder, shape validation of dreamed/dream, spec §5/§7/§8/§9, help.go, templates (README/LEAD/brief), docs/dream.md recipes, README section, D25. Release-prep meta ec0b0f6 carries the annotated tag v0.11.0. Published release and notes: https://github.com/metravod/omut/releases/tag/v0.11.0.
**Verified:** final go vet ./... and go test ./... pass; `check --fail-on warn` on this repository and `git diff --check` clean before tagging. Dream fixtures cover counting per kind (log, lenient archive, thread, decision, graveyard), strictly-after dreamed:, threshold default/override/invalid, the due boundary, report vs apply wording, JSON, no writes on --apply, scopes with `..`, incomplete on read/shape errors; reminder fires at the threshold, extinguishes after dreamed:, frozen excluded; --preview with and without a payload; commit --preview refused. Tagged-commit CI 43068721041 and meta 43068721044 green; release workflow 43068738884 passed macOS vet/tests and GoReleaser. Six archive digests match checksums.txt; Homebrew cask 0.11.0 (tap commit 915f242); install.sh from the tag checksum-verified into a temporary prefix, binary reports 0.11.0; released-binary smoke passes: dream digest/preview on a scratch project, strict check on a filled one. This machine upgraded via brew (omut version 0.11.0); zero run at the pack root (LEAD.md) and in every project (README and the AGENTS.md block); pack check green; zymi-core carries a pre-existing agent-written shape error (thread status "open"), reported, not touched.
**Not verified:** a real dream run by an agent on this pack (the reminder is active on omut: 81 records), the other five released binaries, digest usefulness judged by a human, cross-agent dream recipes executed live.
**Open:** none.
**Next step:** live with the reminder; run the first real dream on omut itself with the human's approval, then judge the report format.
