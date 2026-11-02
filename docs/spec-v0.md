# omut: specification v0

Open questions are at the end.

## 1. What it is

A way to keep meta for a **pack of projects**. A pack is one root directory with projects inside. Any agent started in the root becomes the **lead**: it knows the whole pack and can tell what is going on where. An agent started inside a project works on that project and keeps its meta.

There is no runtime of its own: only files and a small CLI that creates the structure and checks its shape.

### Problems it solves

1. **Seeing the pack.** Agents build the projects, and the human stops remembering what is where and why. Something quietly runs on a server for half a year though it was meant for a week.
2. **Handing a project between agents and models.** The limits ran out, a task is cheaper on another model, a colleague works in another tool. The new agent has to understand the history, the current state, the plans and the reasons behind decisions without digging through the code.
3. **Parking unfinished work.** A session should be stoppable without losing what was done, the same way every time.

### What sets it apart

The meta can be checked. The lead doesn't just read what is written, it compares it with reality: the intent, "Where it lives", how fresh the brief is. Mismatches go to the human. The comparison with neighbours is in [research/competitors.md](research/competitors.md).

## 2. Principles

1. **Everything is plain files.** No assumptions about git, where the root lives or how it syncs. What to commit and where to keep it is the user's call.
2. **Agent-agnostic.** An agent enters through `AGENTS.md` and the CLI. Hooks of particular agents are optional adapters.
3. **The tool creates the structure and checks the shape; agents write the meaning.** The CLI has no commands that write meta.
4. **The meta owns only itself.** It links to project artifacts (ADRs, plans, docs) instead of copying them.
5. **The view of the pack is assembled on read.** No registries to maintain.
6. **What is declared is checked against what is observed.** A conflict is not papered over with a confident update; it is shown to the human.
7. **Size is part of the shape.** `check` enforces budgets, or the meta bloats.
8. **One active writer.** At any moment one agent keeps a project's meta, and the same goes for the pack's. Parallel work needs an explicit split of responsibility.

## 3. What it doesn't do

- No code review, no diffs.
- Not a task tracker: no sprints, no estimates.
- Not an orchestrator of parallel agents.
- Not the working memory of the current task. That is another layer, such as planning-with-files.
- Doesn't store the user's personal preferences. That is a particular agent's memory.
- Automates nothing by itself. The rituals rest on discipline; cron and hooks can be bolted on from outside.

## 4. Layout

### Pack root

```
<root>/
  AGENTS.md            ← the user's: shared pack rules + the omut pointer block
  .omut/
    LEAD.md            ← the tool's: the lead's instructions
    brief.md           ← pack meta: focus, shared infrastructure
    decisions.md       ← pack-level decisions
    graveyard.md       ← tombstones of removed projects
  <project>/ ...
```

Every file has exactly one owner:
- **Tool files** (`LEAD.md` in the root, `.omut/README.md` in every project): `zero` updates them to the current template. Nobody edits them by hand.
- **User files** (`AGENTS.md`): the tool doesn't touch them, except for the block between `<!-- omut:start -->` and `<!-- omut:end -->`.
- **Meta files** (`brief.md`, `decisions.md`, `graveyard.md`): agents write them. The tool only checks them and never overwrites them.

The pointer block in the root `AGENTS.md`:

```markdown
<!-- omut:start -->
This directory is an omut pack of projects.
- If you were started in this directory, you are the lead: read `.omut/LEAD.md`.
- If you work inside a project, keep its meta by that project's `.omut/README.md`.
<!-- omut:end -->
```

### Project

```
<project>/
  AGENTS.md            ← the user's: project specifics + the pointer block
  .omut/
    README.md          ← the tool's: what this directory is, how to read it, the full rules
    brief.md           ← why, what it is, where it lives, now, next
    decisions.md       ← the decision registry
    threads/<slug>.md  ← threads of work
    log.md             ← history: one entry per session
    log-archive/       ← rotated parts of the log
```

`.omut/` is hidden so that it stays out of the way in listings. Agents don't need to discover it: the pointer block names the exact paths. Whether `.omut/` goes into `.gitignore` is the user's call.

The pointer block is needed because not every agent climbs above the git root looking for `AGENTS.md`. It speaks to two readers: one who only wants to understand the project (read `brief.md`, then the decisions) and one who works on it (the minimal ritual: what to read at the start, what to write at the end). The full rules are in the project's own `.omut/README.md`, not in the pack: a project is cloned, published and handed to colleagues without its pack, and its meta must still be readable and maintainable there. On GitHub the same README is what a person sees on opening `.omut/`. `zero` updates the block and the README.

A pack is optional. Outside one, `start`, `freeze`, `thaw`, `check` and `zero` work on a single project: the directory given, or the current one.

If there is no `CLAUDE.md`, `init` and `start` make it a symlink to `AGENTS.md`. An existing `CLAUDE.md` is left alone; `check` warns (`pointer`) when it doesn't lead to `AGENTS.md`.

## 5. Project meta

### brief.md: always read, budget ≤ 60 lines

```markdown
---
status: active              # active | frozen
updated: 2026-10-31         # when the text changed
verified: 2026-10-31        # when "Where it lives" and "Now" were checked against reality
by: claude-code/opus-5.5
adr: docs/adr               # optional: where the project's ADRs are
---
## Why
The original intent, 1–3 lines. Never rewritten after the fact: a changed intent is a decision in decisions.md.

## What it is
How it works, 3–10 lines. For a human, to understand the project without the code. Links to the key documents.

## Where it lives
The footprint in reality: servers, cron, deploys, installs, data, money, with start dates. Empty means it runs nowhere.

## Now
What works, what is broken.
**Verified:** ... (how)
**Not verified:** ...

## Next
A short plan, 3–7 items. Small ideas live here too.
```

Headings, field names and labels are fixed because `check` looks for them. The text itself is written in whatever language the project's people use.

### decisions.md: the decision registry

```markdown
# Decisions

- D1 [accepted 2026-06-02] SQLite, not Postgres: one machine, no server needed
- D2 [accepted 2026-09-10] Deploy with a systemd timer → docs/adr/0007-systemd-timer.md
- D3 [open] Keep collecting after the week? Stop / keep with rotation → threads/hn-future.md
- D4 [doubted 2026-10-31] Keep raw HTML: grows 200 MB a month, probably not needed
- D5 [superseded → D2] Deploy with cron
```

- **Statuses:** `open`: not decided, there are options; `accepted`; `doubted`: decided, but there is a doubt; `superseded → Dn`.
- **If the project has ADRs,** the line links to the ADR, and the full text is written the way the project does it.
- **If there are no ADRs,** the reason goes right into the line: 1–3 sentences, including what was rejected. If it doesn't fit, that's a reason to write an ADR.
- **Lines are never deleted.** A decision changes through `superseded`.

### threads/<slug>.md: threads of work

```markdown
---
status: parked              # active | parked | done | dropped
updated: 2026-10-31
by: claude-code/opus-5.5
parked_because: doubt       # parked only: stamina | doubt | blocked | limits | priority | unknown
---
# Thread title
**Goal:** ...
**Where we got:** ...
**Verified:** ... / **Not verified:** ...
**Open:** ...
**Next step:** ...
```

- **Parking reasons:** `stamina` and `limits`: continue from the same spot; `doubt`: resolve the doubt first (often together with a `doubted` decision); `blocked`: waiting on something external; `priority`: deliberately put aside for something else; `unknown`: the reason was never recorded, only for threads reconstructed during adoption.
- **Threads are only for work on substance.** Ideas without work live in "Next" in `brief.md`.

### log.md: history, newest first

```markdown
- 2026-10-31 · claude-code/opus-5.5 · Discussed the meta, agreed on its parts → threads/meta-format.md
- 2026-10-29 · codex/gpt-5 · Built the projects page; verified: `npm run build`
```

- **One or two lines per session.** Details go by link to a thread or a decision.
- **The author is `<agent>/<model>`** or `human`.
- **Rotation:** past 50 entries, old ones move to `log-archive/<year>.md`.
- **It doesn't retell commits.** Entries reconstructed from agents' sessions are fine, marked `(from session)`.

## 6. Pack meta

- **`.omut/brief.md`**: the pack's focus (what is in progress, what is paused), shared infrastructure (servers and what runs on them, when several projects share them), and the root directories that are not projects (`ignore:` in the frontmatter).
- **`.omut/decisions.md`**: decisions that concern several projects. Same format as a project's.
- **`.omut/graveyard.md`**: tombstones, one line per project: `- hn-parser — removed 2026-10-31 · reason · what it was`.

The list of projects is stored nowhere. It is the subdirectories of the root, except those in `ignore` and hidden ones.

## 7. Commands

All are non-interactive, so an agent can call them. Destructive operations need an explicit flag.

| Command | What it does |
|---|---|
| `init [dir]` | Creates the root (and the directory if missing): `AGENTS.md` with the block, `.omut/` with the tool files and pack meta templates. Doesn't touch subdirectories. If the root exists already, refuses and points at `zero`. |
| `start [name]` | No directory: creates it with meta templates. Existing directory: only adds `.omut/` (with its README) and the block in `AGENTS.md` (creating `AGENTS.md` if missing). Outside a pack, the project is the directory itself (the current one by default). Doesn't touch code or existing files. Refuses a frozen project and points at `thaw`. |
| `freeze [name]` | `status: frozen` in `brief.md`. If "Where it lives" is not empty, warns: make sure nothing still runs. |
| `thaw [name]` | `status: active`. |
| `kill <name> --yes --reason "..."` | Removes the project directory and writes a tombstone (the reason is required; "what it was" comes from "Why"). Without `--force`, refuses when there are uncommitted changes or commits that are on no remote, and always for a project without git: there is no copy anywhere. A symlinked project loses only the link. Prints where the project is mentioned in other projects' meta; fixing those mentions is the agent's job. |
| `check [name]` | A report that changes nothing (rules in section 8). Without a name, the whole pack; outside a pack, the project in the current directory if it has `.omut/`. `--json` for agents and automation. Exits non-zero on shape errors. |
| `zero [name]` | Repairs the scaffold: updates the tool files to the current template, creates missing files from templates, fixes the pointer block. Never touches meta files; only lists their shape errors. Without a name, the root; outside a pack, the project in the current directory. Removes tool files of earlier versions (the root's `conventions.md`). This is also how a pack is updated after a new release. |

| `hook start\|commit` | For hooks, never called by hand: `start` prints the context a session starts from (brief, decisions waiting, unfinished threads, findings, and work newer than the last log write that no commit holds), `commit` is a git pre-commit check (a reminder when staged work is newer than the log, a block when the commit carries meta with shape errors; `--strict` also blocks work without a log entry). `--agent claude\|codex\|cursor` reads that agent's payload and answers in its format. Silent outside any meta. Recipes: [`hooks.md`](hooks.md). |

`freeze`, `thaw` and `start` on an existing directory are the only places where the CLI touches meta files. They change the single `status` field or create missing templates.

## 8. check rules

| Code | Level | Condition |
|---|---|---|
| `no-meta` | error | a root subdirectory without `.omut/` (except `ignore` and hidden ones) |
| `shape` | error | a missing required frontmatter field or section, a wrong status, a repeated Dn, a broken link |
| `size` | warn | `brief.md` > 60 lines, a thread > 80 lines, `log.md` > 50 entries (time to rotate) |
| `stale-brief` | warn | the project has changes newer than `updated`, `.omut/` aside. With git: the last commit that touched anything outside `.omut/`, plus the mtimes of changed and untracked files (a fresh clone is not a change). Without git: the mtime of files outside hidden directories. A symlinked project is followed to its target |
| `unverified` | warn | an active project whose `verified` is empty or older than 30 days |
| `stalled-thread` | info | an `active` thread not updated for more than 14 days |
| `open-decision` | info | `open` and `doubted` decisions: a list for the human |
| `frozen-but-live` | warn | `frozen`, but "Where it lives" is not empty |
| `adr-ahead` | warn | the `adr` directory has files newer than `decisions.md` |
| `ghost` | warn | a name from `graveyard.md` is mentioned in live projects' meta (logs don't count) |
| `unfilled` | warn | empty "Why", "What it is" or "Now", an empty `by`, a thread without "Next step": the template was not filled in |
| `pointer` | warn | `AGENTS.md` has no block, `CLAUDE.md` doesn't lead to `AGENTS.md` (a symlink, or `@AGENTS.md` in `CLAUDE.md` or `CLAUDE.local.md`), or the project has no `.omut/README.md` |
| `scaffold` | error | the root lacks a tool file or a pack meta file; `zero` fixes it |
| `outdated` | info | a tool file (the root's `LEAD.md`, a project's `.omut/README.md`) differs from this version's template; `zero` updates it |

A fresh template is not a shape error but `unfilled` and `unverified`: shape errors are kept for what is broken.

`check` doesn't look inside other artifacts: the format of ADRs and docs is not its business.

## 9. Rituals

The rituals are written in the tool files. Any agent carries them out.

**Inside a project** (`.omut/README.md`):
- **Session start.** Read `brief.md`, the active and parked threads, `open` and `doubted` decisions. If a thread says "Verified", rerun that check before continuing.
- **A decision is made.** A line in `decisions.md`; an ADR too, if the project keeps them.
- **A piece of work is done, or parked.** Update the thread (create it if there is none), "Now" in `brief.md`, a line in `log.md`, before telling the human the work is done and before a commit. Not "at session end": no agent sees it. The result must be checkable: what was done, what was verified and how, what was not, the next step.
- **Freezing.** A final "Now", a log line, check "Where it lives", then `freeze`.

**In the root** (`LEAD.md`):
- **Overview.** Run `check` and read the `brief.md` of active projects. Tell the human what waits for them: open decisions, doubts, stalled threads, mismatches.
- **Audit.** Check "Where it lives" and "Now" against reality with whatever is at hand (ssh, git, logs). Update `verified`. Don't fix mismatches silently; show them to the human.
- **Project story.** Explain a project in a couple of minutes from `brief.md` and the decisions, without reading the code.
- **Routing.** Match a vague request ("what's up with the site") to a project by "Why" and "What it is".
- **Adopting a project.** `start`, then fill in the meta. Look for history everywhere: git, documents and untracked files, old journals, agents' memory and sessions, neighbouring directories, external documents. Don't invent "Why": what the sources don't say, ask. A fork is a separate project: only accepted decisions carry over from the ancestor. Offer to move project rules from a vendor file into `AGENTS.md`.
- **`freeze` and `kill`** only with the human's explicit consent.

## 10. Compatibility

- **Entry point:** `AGENTS.md`. Agents look for it up the tree, so each project has its own `AGENTS.md` with the pointer.
- **Claude Code:** `CLAUDE.md` with `@AGENTS.md`, or a symlink to `AGENTS.md`. `@AGENTS.md` may also sit in a gitignored `CLAUDE.local.md`, when the tracked `CLAUDE.md` is not yours to change.
- **Codex:** reads `AGENTS.md`. Whether it climbs above the git root is still to be checked; if not, the project's pointer is enough.
- **An agent without `AGENTS.md` support:** the human says "read AGENTS.md".
- **Hooks (optional, recommended):** `omut hook start` at an agent's session start, `omut hook commit` as the git pre-commit hook, `omut check` in CI. The end of a session is a commit, not the agent's stop hook, which fires after every turn. Recipes for git, CI, Claude Code, Codex and Cursor: [`hooks.md`](hooks.md).

## 11. Implementation

- **Go, one static binary,** templates embedded with `embed`. No network, no daemon, no storage outside the pack's files.
- **The templates work without the CLI:** copy `templates/` by hand and go. The CLI adds `check` and convenience.
- **Build and distribution:** goreleaser + a Homebrew tap.

## 12. Open questions

1. **Finished threads** (`done`/`dropped`): fold the substance into decisions and the log and delete the file, or keep `threads/archive/`. For now the files stay.
2. **`check` thresholds** are fixed (60/80 lines, 30/14 days, 50 entries). Overrides if they turn out to be needed.
3. **`kill --keep-files`.** Not for now: a symlinked project is removed without its files anyway.
4. **Codex and `AGENTS.md` above the git root:** check by hand.
5. **Migrating from drift:** a one-off script, not a command.
6. **The pack as a project:** does the root need its own `log.md`?
7. **Cross-project links.** A decision may rest on a neighbour's decision (`→ ../other/.omut/decisions.md#Dn`), and a fork's accepted decisions come from its ancestor. Is a link enough, or does `check` need to know about it? Where do pack-level decisions that several projects rely on live?
8. **Invalidation condition on a decision** (borrowed from Pensieve, see `research/competitors.md`): an optional tail `· revisit if …` in the decision line. An agent checks it against reality, like "Where it lives"; `check` can't.
9. **Files for other agents:** Cursor rules and the like, on request.
