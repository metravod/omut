---
status: active
updated: 2026-11-12
verified: 2026-11-11
by: zcode/glm-5.3
dreamed: 2026-11-11
---
## Why
Keep a pack of projects so that the human and any agent (Claude Code, Codex, others) know what is where, why, and what comes next, and can hand a project from one agent or model to another without digging through the code. Grew out of using drift, without its TUI editor and with the meta tied to reality.

## What it is
A Go CLI with no dependencies. Agents keep project memory as markdown; the CLI lays down templates, checks shape and recorded freshness, and assembles recorded context. `cmd/omut` is the entry point, `internal/check` the rules, `internal/context` the context collector, `templates/` the embedded format. Intent: docs/spec-v0.md; neighbours: docs/research/competitors.md.

## Where it lives
GitHub: github.com/metravod/omut (public, 2026-11-01). CI and scheduled meta checks: `.github/workflows/ci.yml` and `omut.yml`. Tagged releases use `release.yml`, `macos.yml` and goreleaser; packages go to metravod/homebrew-tap. Install options: README.md. Nothing runs as a service. This copy: local/instance.md.

## Now
- v0.11.2 (2026-11-12, patch): the pointer block and the project rules carry the degraded-mode fallback — with omut installed, `omut hook start` gives the session start assembled; without it, the agent may suggest the human install omut (Homebrew or install.sh, no Go). Spec §3 and the hook rows in spec/help say an agent may run start when no hook is wired → threads/release-v0.11.2.md.
- Released v0.11.1 (2026-11-11): memory dreaming — `omut dream` and the volume-based `dreamed:` reminder (D25) — and the lean startup selection (D26) → threads/release-v0.11.0.md, threads/release-v0.11.1.md. Earlier releases and their evidence live in the log and log-archive.
- The format works end to end: one memory per clone (D17), two layers (D15), the ADR registry (D14), private meta (D16), budgets (D19), work plans (D23), assembled context with Git observations (D21–D22), agent-owned verification through claims (D11) and rules that travel in `.omut/README.md` (D12). The decisions hold the reasons; the release threads hold the evidence.
- Launch preparation stands: the memorilla comparison → docs/research/competitors.md; the Russian Threads introduction drafted, image and publication pending → threads/threads-introduction.md; comparative benchmarking stopped after the pilot, no task-success regression (D20) → docs/research/handoff-benchmark-results.md.
**Verified:** v0.11.0 and v0.11.1 main CI, macOS release gate, six archives/checksums each, Homebrew casks, checksum-verified installers and released-binary smoke → threads/release-v0.11.0.md, threads/release-v0.11.1.md. The first dream's four claims ran against reality on 2026-11-11; three held, the version claim was corrected. go vet ./... and go test ./... pass; size regressions include Cyrillic and LF/CRLF. Native private Codex startup/continuation, worktree recovery and safe opt-out → docs/handoff.md, docs/research/hn-prelaunch-review.md. Earlier release/live-check history → log-archive/brief-before-prelaunch-2026-11-05.md.
**Not verified:** cross-provider or comparative task-quality gains, context token savings or a fresh lead using the new overview, actual crash recovery, Codex resume/compact/fork, Cursor hooks, D18's memory question during a real adoption, a fresh agent deciding from help alone, the badge on another repository, and performance on a large pack.

## Next
1. Publish the prepared introduction and gather outside attempts; worked case → docs/handoff.md.
2. Live with the active pack; keep copy-only facts in local/ and give checkable claims their `how` → docs/research/reality-check.md.
3. Try the badge on another public repository, then the Cursor hook recipe → docs/hooks.md.
4. Gather outside attempts after publication; launch research → docs/research/hn-launch-timing.md.
