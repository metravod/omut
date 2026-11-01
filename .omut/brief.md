---
status: active
updated: 2026-11-01
verified: 2026-11-01
by: claude-code/opus-5.5
---
## Why
Keep a pack of projects so that the human and any agent (Claude Code, Codex, others) know what is where, why, and what comes next, and can hand a project from one agent or model to another without digging through the code. Grew out of using drift, without its TUI editor and with the meta tied to reality.

## What it is
A Go CLI without dependencies: `cmd/omut` (commands), `internal/meta` (parsing meta files), `internal/check` (rules), `internal/scaffold` (init/start/zero/kill), `templates/` (embedded, usable by hand). The intent is `docs/spec-v0.md`, the neighbours are `docs/research/competitors.md`.

## Where it lives
GitHub: github.com/metravod/omut (public, 2026-11-01). Releases via goreleaser on a v* tag (.github/workflows/release.yml, needs the HOMEBREW_TAP_TOKEN secret); the Homebrew cask goes to github.com/metravod/homebrew-tap (`brew install metravod/tap/omut`). Nothing runs anywhere.

## Now
v0: all 7 commands and the check rules from the spec. Templates, CLI messages and docs are in English.
**Verified:** `go test ./...` (14 tests: meta parsing, init/start/zero/freeze/kill/check end to end); `check` on a pack of symlinked real projects, 0.03 s.
**Not verified:** Codex with AGENTS.md above the git root; a handoff to Codex on real meta; check on a large pack; releases from CI (need a tap token).

## Next
1. Live with a real pack of active projects
2. Handoff: open a project in Codex, ask "what is this and what's next"
3. Check Codex and AGENTS.md above the git root
4. Releases from CI: set the HOMEBREW_TAP_TOKEN secret, then tag
