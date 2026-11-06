---
status: done
updated: 2026-11-06
by: codex/GPT-6
---
# Deterministic context assembly
**Goal:** give the pack lead an assembled overview and a project's current records without model-driven file discovery.
**Where we got:** `internal/context` assembles literal records; `omut context [project] [--json]` and pack startup share it. Overview includes pack context, current project sections, waiting decisions/ADRs, unfinished thread summaries and findings. Project view adds full current records, filled instance memory and five entries per log. Selection and omissions are explicit; no model, persistent inbox, cache, runtime or claim execution. D21 → ../decisions.md.
**Verified:** `go vet ./...` and `go test ./...` pass. Fixtures cover exact source contents/line numbers and text/JSON parity, stable ordering, multiline thread fields, ADR selection, frozen projects, partial failures and malformed records, missing empty threads directories, discovery/ignore/symlinks, read-only behavior, deep scope and private/shared worktree layers. Real six-project pack: no errors or warnings; JSON/text both exit 0, about 27 KB of text, and pack startup contains the identical overview. Spec §6.1, help, README and lead template describe the behavior.
**Not verified:** actual model-token or task-quality savings, a fresh lead using this context, or a large pack. The smoke measures assembly/output only. Released in v0.9.0; the released binary passes the context smoke → release-v0.9.0.md. Existing installed hooks/binaries were not changed.
**Open:** none for this implementation. Existing handoff pilot changes were preserved.
**Next step:** none for implementation; try the new overview in normal pack work before adding an interface.
