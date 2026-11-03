---
status: active
updated: 2026-11-03
verified: 2026-11-03
by: claude-code/opus-5.5
---
## Why
Keep a pack of projects so that the human and any agent (Claude Code, Codex, others) know what is where, why, and what comes next, and can hand a project from one agent or model to another without digging through the code. Grew out of using drift, without its TUI editor and with the meta tied to reality.

## What it is
A Go CLI without dependencies: `cmd/omut` (commands), `internal/meta` (parsing meta files), `internal/check` (rules), `internal/scaffold` (init/start/zero/kill), `templates/` (embedded, usable by hand). The intent is `docs/spec-v0.md`, the neighbours are `docs/research/competitors.md`.

## Where it lives
GitHub: github.com/metravod/omut (public, 2026-11-01). CI: Linux tests on every push (.github/workflows/ci.yml); this repo's own meta checked with `--fail-on warn` on push and weekly (omut.yml, the badge in README.md); macOS tests by hand and before every release (macos.yml, called by release.yml). Releases via goreleaser on a v* tag (.github/workflows/release.yml; the HOMEBREW_TAP_TOKEN secret is a fine-grained token with Contents: write on metravod/homebrew-tap only); the Homebrew cask goes to github.com/metravod/homebrew-tap (`brew install metravod/tap/omut`). Nothing runs anywhere. This copy's machine (the checkout, the pack, the installed binary, hooks): `local/instance.md`, not in git.

## Now
v0: all 7 commands and the check rules from the spec. Templates, CLI messages and docs are in English. In v0.2.0 (2026-11-03): every project carries its rules in `.omut/README.md`, a single repository works without a pack (D12), `omut hook start|commit` for agents' and git's hooks (D13, docs/hooks.md); the rules tie recording to "before telling the human it's done", not to a session end no agent sees; `hook start` reports work newer than the log that no commit holds, which is the only signal without git; `check --fail-on warn` in CI is the omut badge (fresh, not only well-formed), dogfooded in this repo's README. In v0.3.0 (2026-11-03): with `adr:` the ADRs are the decision registry, `check` and `hook start` read their statuses (D14); `start` prints the badge for the root of a GitHub repository; `kill` refuses over gitignored files no build recreates; a frozen project's open decisions are not listed; rules for moving a project and for history with unknown dates. Since v0.3.0 (unreleased): two layers (D15): `.omut/local/` is this copy's memory (machine, servers, access, runbook), kept out of git by `.omut/.gitignore`; `check` warns about a copy's facts in the project layer (`leak`) and about an instance layer git would take.
**Verified:** `go test ./...` (30 tests: meta parsing, init/start/zero/freeze/kill/check end to end, `stale-brief` through a symlink and with git, `pointer` via `CLAUDE.local.md`, a single project without a pack, README in pack projects, `hook start`/`hook commit`, unrecorded work with and without git, `--fail-on warn`, `stale-brief` in the committer's time zone, `kill` over a gitignored `.env`, open decisions muted when frozen, the badge from the origin URL, ADR statuses in five formats and the ADR registry end to end, the instance layer out of git and in `hook start`, `leak` without false alarms on versions); on zymi-core's 46 real ADRs `check` lists exactly the two `Proposed` ones; the Claude Code SessionStart recipe live (2026-11-03, `~/.claude/settings.json`, a headless session quoted the context); `zero` on `~/oss_projects` root and omut retires `conventions.md` and lays down the README; `check` on the pack: 0.6 s (two git calls per project).
**Not verified:** Codex with AGENTS.md above the git root; a handoff to Codex on real meta, now also without the pack (a clone with only `.omut/README.md`); the Codex and Cursor hook recipes live (formats from vendor docs, 2026-11-02); the badge on a repository other than this one; check on a large pack. A release from CI is verified (v0.2.0 via `brew upgrade`); rerunning goreleaser for an existing tag needs the release's assets deleted first.

## Next
1. Live with a real pack of active projects; commit the pack's meta once it is cleaned of internal details (pack D3), sorting facts by D15 rather than deleting them: `omut check` lists the leaks (home paths; host names like `workhorse` are not caught yet)
2. Codex: a handoff on real meta ("what is this and what's next") and AGENTS.md above the git root
3. D11 on top of D15: reality checks for the instance layer first, the core of the product, not to be rushed → docs/research/reality-check.md
4. The badge on the first public repository of the pack: the workflow from docs/hooks.md, then see it green
5. Codex and Cursor hook recipes live → docs/hooks.md
