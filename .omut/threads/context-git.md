---
status: done
updated: 2026-11-06
by: codex/GPT-6
---
# Git observations in assembled context
**Goal:** replace the lead's routine per-project status loop with a deterministic Git summary in context.
**Where we got:** context Markdown/JSON and pack startup include separate Git observations: branch/head/upstream, local divergence, scoped XY changed paths with rename sources/submodule state and conflicts. Non-Git projects are explicit; failures retain meta and mark context incomplete. `Place.WorkDir` preserves the actual path of a private worktree subproject. No fetch or full diff. Lead rules use the supplied overview and reserve specific diffs for questions/discrepancies. D22 → ../decisions.md.
**Verified:** `go vet ./...` and `go test ./...` pass. Real Git fixtures cover local ahead/behind and a missing upstream ref, staged+unstaged changes, deletion, Unicode/tab/newline paths and renames, detached/unborn/no-Git/error states, merge conflicts, scoped private subproject worktrees, index bytes/mtime preservation and suppression of an actual configured fsmonitor hook. Parser tests cover submodules and incomplete records. Real-pack smoke checks Git states and hook/command parity without changing Git indices. Help, spec, README and lead instructions are updated; check rule meanings are unchanged.
**Not verified:** a fresh native lead following the new guidance, model-token/task-quality savings or large-repository performance. Released in v0.9.1; the downloaded binary passes Git-context and private-worktree smoke → release-v0.9.1.md. System installation, existing hook configuration and pack tool files were not changed.
**Open:** none for implementation. Git observations do not certify the truth of recorded project claims or the server's current refs.
**Next step:** none for implementation; try a normal pack overview after upgrading the CLI and refreshing the root tool file.
