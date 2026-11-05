---
status: active
updated: 2026-11-05
verified: 2026-11-05
by: codex/GPT-6
---
## Why
Keep a pack of projects so that the human and any agent (Claude Code, Codex, others) know what is where, why, and what comes next, and can hand a project from one agent or model to another without digging through the code. Grew out of using drift, without its TUI editor and with the meta tied to reality.

## What it is
A Go CLI with no dependencies. Agents keep project memory as markdown; the CLI lays down templates and checks shape and recorded freshness. `cmd/omut` is the entry point, `internal/check` the rules, `templates/` the embedded format. Intent: docs/spec-v0.md; neighbours: docs/research/competitors.md.

## Where it lives
GitHub: github.com/metravod/omut (public, 2026-11-01). CI and scheduled meta checks: `.github/workflows/ci.yml` and `omut.yml`. Tagged releases use `release.yml`, `macos.yml` and goreleaser; packages go to metravod/homebrew-tap. Install options: README.md. Nothing runs as a service. This copy: local/instance.md.

## Now
- Standalone projects and optional packs work; `start --private` keeps memory out of git. Shared meta follows branches; private meta and `local/` belong to the main checkout and are found from worktrees.
- Agents own content and verification. Rules travel in `.omut/README.md`; hooks provide startup context and commit reminders. Existing ADRs remain their decision registry. Decisions D11–D18 explain these choices.
- Preparing v0.8.0 with the prelaunch pass complete: Unicode volume budgets (6000/8000 characters), compact current-state memory and clearer onboarding/check claims (D19). Native Codex private startup and controlled task continuation work after hook trust/setup → docs/handoff.md, threads/prelaunch-evidence.md.
**Verified:** go vet ./... and go test ./... pass; size regressions include Cyrillic and LF/CRLF. Native private Codex startup/continuation, worktree recovery and safe opt-out → docs/handoff.md, docs/research/hn-prelaunch-review.md. Earlier release/live-check history → log-archive/brief-before-prelaunch-2026-11-05.md.
**Not verified:** cross-provider or comparative task-quality gains, actual crash recovery, Codex resume/compact/fork, Cursor hooks, D18's memory question during a real adoption, a fresh agent deciding from help alone, the badge on another repository, and performance on a large pack.

## Next
1. Publish the prepared introduction and gather outside attempts; worked case → docs/handoff.md.
2. Live with the active pack; keep copy-only facts in local/ and give checkable claims their `how` → docs/research/reality-check.md.
3. Try the badge on another public repository, then the Cursor hook recipe → docs/hooks.md.
4. Gather outside attempts after publication; launch research → docs/research/hn-launch-timing.md.
