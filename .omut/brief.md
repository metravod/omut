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
GitHub: github.com/metravod/omut (public, 2026-11-01). CI: Linux tests on every push (.github/workflows/ci.yml); this repo's own meta checked with `--fail-on warn` on push and weekly (omut.yml, the badge in README.md); macOS tests by hand and before every release (macos.yml, called by release.yml). Releases via goreleaser on a v* tag (.github/workflows/release.yml; the HOMEBREW_TAP_TOKEN secret is a fine-grained token with Contents: write on metravod/homebrew-tap only); the Homebrew cask goes to github.com/metravod/homebrew-tap (`brew install metravod/tap/omut`). Nothing runs anywhere. Local checkout: `~/oss_projects/omut`, moved from `~/Documents/pet_projects` on 2026-11-02; `~/oss_projects` is the first real pack (6 projects).

## Now
v0: all 7 commands and the check rules from the spec. Templates, CLI messages and docs are in English. In v0.2.0 (2026-11-03): every project carries its rules in `.omut/README.md`, a single repository works without a pack (D12), `omut hook start|commit` for agents' and git's hooks (D13, docs/hooks.md); the rules tie recording to "before telling the human it's done", not to a session end no agent sees; `hook start` reports work newer than the log that no commit holds, which is the only signal without git; `check --fail-on warn` in CI is the omut badge (fresh, not only well-formed), dogfooded in this repo's README.
**Verified:** `go test ./...` (24 tests: meta parsing, init/start/zero/freeze/kill/check end to end, `stale-brief` through a symlink and with git, `pointer` via `CLAUDE.local.md`, a single project without a pack, README in pack projects, `hook start`/`hook commit`, unrecorded work with and without git, `--fail-on warn`, `stale-brief` in the committer's time zone); `zero` on `~/oss_projects` root and omut retires `conventions.md` and lays down the README; `check` on the pack: 0.6 s (two git calls per project).
**Not verified:** Codex with AGENTS.md above the git root; a handoff to Codex on real meta, now also without the pack (a clone with only `.omut/README.md`); the hook recipes in a live Claude Code, Codex or Cursor session (formats from vendor docs, 2026-11-02); check on a large pack. A release from CI is verified (v0.2.0 via `brew upgrade`); rerunning goreleaser for an existing tag needs the release's assets deleted first.

## Next
1. Live with a real pack of active projects; commit the pack's meta once it is cleaned of internal details (pack D3)
2. Codex: a handoff on real meta ("what is this and what's next") and AGENTS.md above the git root
3. Try the hook recipes live: Claude Code first (we run in it), then Codex and Cursor → docs/hooks.md
4. Triage findings 7–13 from the first real pack (drift migration, `start --local`, ADR registry, unknown dates, gitignored files in `kill`) → threads/review-2026-11-01.md
5. The badge beyond this repo: offer it when adopting a public project; maybe `omut start` prints the snippet
6. Machine-checkable "Where it lives" → D11
