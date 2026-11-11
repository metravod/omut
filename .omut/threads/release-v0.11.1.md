---
status: done
updated: 2026-11-11
by: zcode/glm-5.3
---
# Release v0.11.1
**Goal:** trim the session-start context by selection, not by modes (D26), at the human's request after they measured the dump: a parked plan was 26% of it.
**Where we got:** the lean commit 35005e3 and release-prep 7dd9943 are pushed; annotated v0.11.1 points at 7dd9943. Published release: https://github.com/metravod/omut/releases/tag/v0.11.1. On omut itself the dump went 15 098 → 10 033 characters; the first dream compresses the brief further.
**Verified:** final go vet ./... and go test ./... pass; new fixtures: parked plan collapses and an active one keeps its body, Access/Runbook pointers, claim locations with no repeated `how`, local log capped at three; TestClaimsAreCheckedByAgents updated to the location format; strict check clean. Release workflow 43085990940 green; six archives match checksums.txt; cask 0.11.1; released binary reports 0.11.1 and collapses a parked plan in smoke. This machine upgraded via brew (0.11.1); zero run across the pack; strict check clean.
**Not verified:** a real dream run by an agent (next in this session); the other five released binaries.
**Open:** none.
**Next step:** run the first real dream on omut itself with the human's approval.
