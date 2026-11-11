---
status: active
updated: 2026-11-06
verified: 2026-11-06
by: codex/GPT-6
---
## Why
Keep a pack of projects so that the human and any agent (Claude Code, Codex, others) know what is where, why, and what comes next, and can hand a project from one agent or model to another without digging through the code. Grew out of using drift, without its TUI editor and with the meta tied to reality.

## What it is
A Go CLI with no dependencies. Agents keep project memory as markdown; the CLI lays down templates, checks shape and recorded freshness, and assembles recorded context. `cmd/omut` is the entry point, `internal/check` the rules, `internal/context` the context collector, `templates/` the embedded format. Intent: docs/spec-v0.md; neighbours: docs/research/competitors.md.

## Where it lives
GitHub: github.com/metravod/omut (public, 2026-11-01). CI and scheduled meta checks: `.github/workflows/ci.yml` and `omut.yml`. Tagged releases use `release.yml`, `macos.yml` and goreleaser; packages go to metravod/homebrew-tap. Install options: README.md. Nothing runs as a service. This copy: local/instance.md.

## Now
- Memorilla comparison added: overlap in recovering accumulated context, different published interface (model-specific latent vectors versus portable project records); no benchmark replication or integration → docs/research/competitors.md.
- First Russian Threads introduction drafted: a handoff story, a memory example as the image and copyable setup in a reply; image and publication remain future work → threads/threads-introduction.md.
- Standalone projects and optional packs work; `start --private` keeps memory out of git. Shared meta follows branches; private meta and `local/` belong to the main checkout and are found from worktrees.
- Agents own content and verification. Rules travel in `.omut/README.md`; hooks provide startup context and commit reminders. Existing ADRs remain their decision registry. Decisions D11–D18 explain these choices.
- Memory budgets (6000/8000 Unicode characters), compact current-state memory and clearer onboarding/check claims are in place (D19). Native Codex private startup and controlled task continuation work after hook trust/setup → docs/handoff.md, threads/prelaunch-evidence.md.
- First comparative handoff pilot complete: omut and plain note both pass 3/3 balanced scenarios; no saved handoff passes 2/3. No task-success regression observed in this small pilot; general quality/efficiency gains remain unestablished. Human accepts this check and stops further benchmarking (D20); expected long-term value is discipline, format, startup context and revisiting old claims → docs/research/handoff-benchmark-results.md.
- Released v0.10.0 (2026-11-06) → threads/release-v0.10.0.md. `omut context [project] [--json]` assembles recorded meta and separate local Git observations, with explicit selection/failures; pack startup supplies the same overview (D21–D22). Lead guidance uses the overview and reserves specific diffs for questions/discrepancies. Provenance, scoped private worktrees and index/fsmonitor behavior tested → threads/context.md, threads/context-git.md.
- Optional work plans (D23): `plan:` in the brief selects a project index; the same field in local/instance.md selects a local package. Context/startup include literal plans with source locations; checks cover paths/readability and 80-line/8000-character size. Rules describe goals, task ownership, dependencies and acceptance without duplicating state. Existing active packages were not migrated → threads/work-planning.md.
**Verified:** v0.10.0 main CI, macOS release gate, six archives/checksums, Homebrew metadata, checksum-verified installer and released plan/privacy/read-only smoke → threads/release-v0.10.0.md. go vet ./... and go test ./... pass; size regressions include Cyrillic and LF/CRLF. Native private Codex startup/continuation, worktree recovery and safe opt-out → docs/handoff.md, docs/research/hn-prelaunch-review.md. Earlier release/live-check history → log-archive/brief-before-prelaunch-2026-11-05.md.
**Not verified:** cross-provider or comparative task-quality gains, context token savings or a fresh lead using the new overview, actual crash recovery, Codex resume/compact/fork, Cursor hooks, D18's memory question during a real adoption, a fresh agent deciding from help alone, the badge on another repository, and performance on a large pack.

## Next
1. Publish the prepared introduction and gather outside attempts; worked case → docs/handoff.md.
2. Live with the active pack; keep copy-only facts in local/ and give checkable claims their `how` → docs/research/reality-check.md.
3. Try the badge on another public repository, then the Cursor hook recipe → docs/hooks.md.
4. Gather outside attempts after publication; launch research → docs/research/hn-launch-timing.md.
