---
status: done
updated: 2026-11-06
by: codex/GPT-6
---
# Goals, tasks and continuation
**Goal:** support optional work plans without reorganizing a task package another agent is already executing.
**Where we got:** implemented D23: optional plan pointers in brief (project-relative) and local/instance.md (local-relative); existing Markdown needs no migration or schema. Context and startup include literal indexes, without following cards or computing task readiness. Project plans enter pack overviews; local plans stay in project context and main-checkout memory. Path/readability and 80-line/8000-character checks reuse shape/size. Rules carry a goals/tasks example, one authoritative state, acceptance/dependency guidance and active-writer protection. Help/spec/README agree. Refreshed only this repository's tool rules and pointer block with the new binary. Copy-specific review evidence → .omut/local/work-planning-review.md.
**Verified:** go vet ./... and go test ./... pass. CLI fixtures cover Markdown without frontmatter, text/JSON source provenance, CRLF, local privacy in pack/startup, frozen selection, branch/private worktrees, read/zero preservation of plan/card bytes and mtimes, missing/invalid/directory/symlink pointers and Unicode/line budgets without truncation. Built /tmp binary; zero . and check . --fail-on warn pass in this repository. No files in the other session were written; installed binaries and its active task package were not changed.
**Not verified:** task execution or improved continuation/token costs in live plan use. Existing plans' task syntax, dependencies, card links and readiness remain agent-owned, deliberately not checked by the CLI.
**Open:** none for implementation; released in v0.10.0 → release-v0.10.0.md. System installation was not changed.
**Next step:** try the convention with a future package or an explicit handoff from the active package's keeper; don't reorganize that keeper's current cards.
