---
status: done
updated: 2026-11-06
by: codex/GPT-6
---
# Comparative handoff benchmark
**Goal:** implement and run a first comparative handoff pilot, including source verification and memory-maintenance costs.
**Where we got:** a Go standard-library research runner compares none/note/omut on pinned source snapshots. Source agents write their own records. Nine balanced source/receiver pairs cover pending CRLF tests, stale completion and irrelevant history: omut 3/3, plain note 3/3, none 2/3. Across retained attempts, 29 model sessions completed. Results and per-stage accounting → docs/research/handoff-benchmark-results.md, docs/research/handoff-benchmark-results.json.
**Verified:** go vet ./... and go test ./... pass. Offline preflight validates baseline/regression/reference outcomes and CRLF mutation headroom. All balanced live cells received full Go and held-out policy checks; CRLF completion is mutation-tested. Original scores remain intact; all cells were uniformly rescored to admit relevant regression tests. Both robustness blocks were repeated in full with external TMPDIR after nested tests polluted two original omut fixtures. No selective per-condition retry or unreported source repair → docs/research/handoff-benchmark.md.
**Not verified:** general comparative quality or efficiency gains, independent task diversity, sequential maintenance, source implementation, blinded semantic review, native memory/hooks, cross-provider use or actual crash recovery. One repeat per cell and two scenarios sharing a repair prevent strong inference.
**Open:** none. The human accepts the observed absence of task-success regression as sufficient for this stage and stops further benchmarking (D20). Long-term discipline, recognizable format, startup context and revisiting older claims remain the expected value, not a measured pilot result.
**Next step:** none; a broader comparative study is not scheduled. Raw paths, transcripts and memory snapshots remain in local/.
