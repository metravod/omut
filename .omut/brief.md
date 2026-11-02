---
status: active
updated: 2026-11-02
verified: 2026-11-02
by: claude-code/opus-5.5
---
## Why
Keep a pack of projects so that the human and any agent (Claude Code, Codex, others) know what is where, why, and what comes next, and can hand a project from one agent or model to another without digging through the code. Grew out of using drift, without its TUI editor and with the meta tied to reality.

## What it is
A Go CLI without dependencies: `cmd/omut` (commands), `internal/meta` (parsing meta files), `internal/check` (rules), `internal/scaffold` (init/start/zero/kill), `templates/` (embedded, usable by hand). The intent is `docs/spec-v0.md`, the neighbours are `docs/research/competitors.md`.

## Where it lives
GitHub: github.com/metravod/omut (public, 2026-11-01). Releases via goreleaser on a v* tag (.github/workflows/release.yml, needs the HOMEBREW_TAP_TOKEN secret); the Homebrew cask goes to github.com/metravod/homebrew-tap (`brew install metravod/tap/omut`). Nothing runs anywhere. Local checkout: `~/oss_projects/omut`, moved from `~/Documents/pet_projects` on 2026-11-02; `~/oss_projects` is the first real pack (6 projects).

## Now
v0: all 7 commands and the check rules from the spec. Templates, CLI messages and docs are in English. Since 2026-11-02 (unreleased): every project carries its rules in `.omut/README.md`, and a single repository works without a pack (D12).
**Verified:** `go test ./...` (19 tests: meta parsing, init/start/zero/freeze/kill/check end to end, `stale-brief` through a symlink and with git, `pointer` via `CLAUDE.local.md`, a single project without a pack, README in pack projects); `zero` on `~/oss_projects` root and omut retires `conventions.md` and lays down the README; `check` on the pack: 0.6 s (two git calls per project).
**Not verified:** Codex with AGENTS.md above the git root; a handoff to Codex on real meta, now also without the pack (a clone with only `.omut/README.md`); check on a large pack; releases from CI (need a tap token). The other 5 projects of the pack have no README yet (`omut zero <name>`).

## Next
1. Live with a real pack of active projects; `omut zero` the other 5 projects
2. Codex: a handoff on real meta ("what is this and what's next") and AGENTS.md above the git root
3. Session hooks: check at start, log/"Now" at the end, for every agent, not only Claude Code → threads/review-2026-11-01.md
4. Triage findings 7–13 from the first real pack (drift migration, `start --local`, ADR registry, unknown dates, gitignored files in `kill`) → threads/review-2026-11-01.md
5. Visible from outside: a static badge linking to `.omut/`, then a live one (fresh/stale) from `check` in CI
6. Machine-checkable "Where it lives" → D11
7. Releases from CI: set the HOMEBREW_TAP_TOKEN secret, then tag
