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
- Doesn't run checks against reality. It remembers what each claim says, how to check it and when that was last done; agents run the checks, with the human's permissions (D11).

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
    .gitignore         ← the tool's: keeps local/ out of git
    .gitattributes     ← the tool's: log.md merges by union, parallel branches don't conflict
    brief.md           ← why, what it is, where it lives, now, next
    decisions.md       ← the decision registry
    threads/<slug>.md  ← threads of work
    log.md             ← history: one entry per session
    log-archive/       ← rotated parts of the log
    local/             ← the instance layer, never in git
      instance.md      ← this copy: its machine, servers, access, runbook
      log.md           ← this copy's events
```

`.omut/` is hidden so that it stays out of the way in listings. Agents don't need to discover it: the pointer block names the exact paths. Whether `.omut/` goes into `.gitignore` is the user's call.

The pointer block is needed because not every agent climbs above the git root looking for `AGENTS.md`. It speaks to two readers: one who only wants to understand the project (read `brief.md`, then the decisions) and one who works on it (the minimal ritual: what to read at the start, what to write at the end). The full rules are in the project's own `.omut/README.md`, not in the pack: a project is cloned, published and handed to colleagues without its pack, and its meta must still be readable and maintainable there. On GitHub the same README is what a person sees on opening `.omut/`. `zero` updates the block and the README.

**Two layers (D15).** Everything in `.omut/` except `local/` is the project layer: it goes into git and to everyone who clones the repository, and it is about continuity: why, what, decisions, the work, the public footprint. `local/` is the instance layer, the memory of one copy, like `.env`: its machine, servers, where secrets are kept, how it is deployed and moved. The test for a fact: would a stranger who clones the repository need it, and is it true for them? `.omut/.gitignore` keeps `local/` out of git without touching the project's own `.gitignore`; `check` warns when git would take it anyway and about facts of a copy (home paths, IP addresses) in the project layer. A pack is the instance layer of several projects: one person's arrangement on one machine.

**One format, visibility is one bit (D16).** omut has no modes for personal, team or open-source use: an agent reading any repository must not need to know how its meta was set up. What differs is whether the project layer is shared, and that is read from git, never stored: when git ignores `.omut/` and tracks nothing in it, the meta is private, both layers belong to this copy. `start --private` sets that up for a repository that is not yours: it adds `.omut/` and `CLAUDE.local.md` to `.git/info/exclude` and puts the pointer block into `CLAUDE.local.md`, so the repository's own files stay unchanged byte for byte. Codex has no local file that adds to `AGENTS.md` (`AGENTS.override.md` replaces it), so for Codex the pointer is the session-start hook or a line in `~/.codex/AGENTS.md`. Sharing later is removing the exclude lines, `zero` (which writes the block into `AGENTS.md`) and a commit. Teams need rules, not a mode: one keeper per thread, decisions named after the human who accepted them, and `.omut/.gitattributes` with `log.md merge=union`, so log lines from parallel branches merge without a conflict.

A pack is optional. Outside one, `start`, `freeze`, `thaw`, `check` and `zero` work on a single project: the one you are in at any depth, or the directory given. `omut start .` gives a subdirectory its own meta, for a monorepo.

**One memory per clone (D17).** In a linked git worktree, committed meta is that branch's own, as any tracked file. This copy's layer, and all of private meta, is in the main checkout: a worktree doesn't get its own, which would go when the worktree goes. Every command finds it from a worktree, including `hook start` and `hook commit`, and the session context gives the agent the full path, since relative paths would write a second memory. A worktree right under a pack root is not one more project.

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
The public footprint, with dates: the repository, releases, packages, public sites, external documents. This copy's machine and servers go to local/instance.md. Empty means it is published nowhere.

## Now
What works, what is broken.
**Verified:** ... (how)
**Not verified:** ...

## Next
A short plan, 3–7 items. Small ideas live here too.
```

Headings, field names and labels are fixed because `check` looks for them. The text itself is written in whatever language the project's people use.

### local/: the instance layer, never in git

```markdown
---
updated: 2026-11-03
by: claude-code/opus-5.5
---
## Where it lives
This machine (the checkout, symlinks, installs, virtualenvs, agents' configs), servers, containers, cron, data, backups, with dates.

## Access
Where the secrets, tokens and accounts are kept: a path, a password manager entry. Never the values.

## Runbook
How this copy is deployed, rolled back, restarted, moved.
```

No section is required: a copy that runs nowhere but in its directory leaves them empty.

**Claims (D11).** Each `- ` line of "Where it lives" (in `brief.md` and in `local/instance.md`) and of "Access" is a claim about reality. It may carry how to check it and when that was last done:

```markdown
- skill adr: ~/.claude/skills/adr → this checkout · how: readlink ~/.claude/skills/adr · seen 2026-11-03
- container wimt-agent on workhorse, image wimt_agent:62b9ca0 · how: ssh workhorse docker inspect -f '{{.Config.Image}}' wimt-agent · seen 2026-11-02
```

`how` is a command or a place to look. omut never runs it: the agent does, under its own permissions, and updates `seen`. omut keeps the memory: what is claimed, how to check it, how old the check is. `check` warns about claims not seen for 30 days (`unseen`) and counts those without `how` (`unchecked`); `hook start` offers the agent up to two claims not seen for 7 days, never-seen first, so the audit happens a little every session. The truth still rests on the agent; a `local/log.md` line that says what was seen, not just "checked", is the evidence. `local/log.md` holds this copy's events (deployed, moved, rotated a token) in the log format. `hook start` gives an agent both layers; a reader on GitHub sees only the project layer.

### decisions.md: the decision registry

```markdown
# Decisions

- D1 [accepted 2026-06-02] SQLite, not Postgres: one machine, no server needed
- D2 [accepted 2026-09-10] Deploy with a systemd timer → docs/adr/0007-systemd-timer.md
- D3 [open] Keep collecting after the week? Stop / keep with rotation → threads/hn-future.md
- D4 [doubted 2026-10-31] Keep raw HTML: grows 200 MB a month, probably not needed
- D5 [superseded → D2] Deploy with cron
```

- **Statuses:** `open`: not decided, there are options, optionally since when (`[open 2026-09-10]`); `accepted`; `doubted`: decided, but there is a doubt; `superseded → Dn`.
- **If the project has ADRs** (`adr:` in the brief), they are the registry for the decisions they record (D14): `decisions.md` holds only the decisions without an ADR, and a line at its top says where the ADRs are. `check` reads an ADR's status from frontmatter, a `Status:` line (also `* Status:`, `**Status:**`) or the first line of a `## Status` section: `proposed`, `draft`, `open`, `pending` wait for the human like `open`; `Firmness: tentative` (stare-decisis) like `doubted`, unless the ADR is superseded, deprecated or rejected; no status counts as decided. ADRs are the `.md` files in that directory whose name starts with a number (`0007-x.md`, `ADR-007-x.md`). A registry that already mirrors its ADRs may stay; new ADRs are not copied. A decision line may still link to an ADR.
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
| `start [name]` | No directory: creates it with meta templates. Existing directory: only adds `.omut/` (with its README, its `.gitignore` and an empty `local/`) and the block in `AGENTS.md` (creating `AGENTS.md` if missing). Outside a pack, the project is the directory given; without one, the project you are in at any depth, or the current directory if there is none. `start .` gives a subdirectory its own meta. Doesn't touch code or existing files. Refuses a frozen project and points at `thaw`. In the root of a git repository whose origin is on GitHub, prints the omut badge line for `README.md` (CI recipe in [`hooks.md`](hooks.md)). `--private` (D16): in an existing git repository, keeps the whole meta out of git through `.git/info/exclude` and writes the block into `CLAUDE.local.md` instead of `AGENTS.md`; refuses where git tracks `.omut/`. Run without a name in a linked worktree, it puts private meta in the main checkout (D17). Over a `.omut/` not yet in git it keeps the meta and makes it private; what an earlier `start` did to `AGENTS.md` and `CLAUDE.md` stays, and it says what git still sees there. |
| `freeze [name]` | `status: frozen` in `brief.md`. If "Where it lives" is not empty, warns: make sure nothing still runs. |
| `thaw [name]` | `status: active`. |
| `kill <name> --yes --reason "..."` | Removes the project directory and writes a tombstone (the reason is required; "what it was" comes from "Why"). Without `--force`, refuses when there are uncommitted changes, gitignored files that no build recreates (`.env`, local data, meta kept out of git; caches such as `node_modules/`, `target/`, `.venv/` don't count), or commits that are on no remote, and always for a project without git: there is no copy anywhere. A symlinked project loses only the link. Prints where the project is mentioned in other projects' meta; fixing those mentions is the agent's job. |
| `check [name]` | A report that changes nothing (rules in section 8). Without a name: in a project of a pack, at any depth, that project; at the root, the whole pack (from a project, `..` names it); outside a pack, the project you are in at any depth or from a linked worktree, if it has `.omut/`. `--json` for agents and automation. Exits non-zero on shape errors; `--fail-on warn` on warnings too, for CI and its badge. |
| `zero [name]` | Repairs the scaffold: updates the tool files to the current template, creates missing files from templates, fixes the pointer block. Never touches meta files; only lists their shape errors. Without a name: in a project of a pack, at any depth, that project, if it has `.omut/`; at the root, the root (from a project, `..` names it); outside a pack, the project you are in at any depth or from a linked worktree. In a linked worktree with committed meta it doesn't create `local/`. Removes tool files of earlier versions (the root's `conventions.md`). In a private project the block goes into `CLAUDE.local.md` unless `AGENTS.md` has it already, and no `CLAUDE.md` link is made. This is also how a pack is updated after a new release. |
| `hook start\|commit` | For hooks, never called by hand: `start` prints the context a session starts from (brief, decisions waiting, unfinished threads, findings, and work newer than the last log write that no commit holds), `commit` is a git pre-commit check (a reminder when staged work is newer than the log, a block when the commit carries meta with shape errors; `--strict` also blocks work without a log entry). `--agent claude\|codex\|cursor` reads that agent's payload and answers in its format. Silent outside any meta. Recipes: [`hooks.md`](hooks.md). |
| `help [command\|meta]` | Without an argument, the overview. `help <command>` (the same as `<command> --help` or `-h`, which win over anything else on the line) says what the command writes, what it leaves alone, when it refuses, its flags and exit codes: enough to decide without trying it in a scratch repository. The flags come from their definitions, and tests compare the rest with the commands and the `check` rules with section 8. `help meta` prints the rules a project would get as `.omut/README.md`, before anything is laid down. |

`freeze`, `thaw` and `start` on an existing directory are the only places where the CLI touches meta files. They change the single `status` field or create missing templates.

## 8. check rules

| Code | Level | Condition |
|---|---|---|
| `no-meta` | error | a root subdirectory without `.omut/` (except `ignore` and hidden ones) |
| `shape` | error | a missing required frontmatter field or section, a wrong status, a repeated Dn, a broken link |
| `size` | warn | `brief.md` > 60 lines, a thread > 80 lines, `log.md` > 50 entries (time to rotate) |
| `stale-brief` | warn | the project has changes newer than `updated`, `.omut/` aside. With git: the last commit that touched anything outside `.omut/`, plus the mtimes of changed and untracked files (a fresh clone is not a change). Without git: the mtime of files outside hidden directories. A symlinked project is followed to its target |
| `unverified` | warn | an active project whose `verified` is empty or older than 30 days |
| `unseen` | warn | in an active project, a claim with `how` whose `seen` is missing or older than 30 days (§5) |
| `unchecked` | info | claims without `how`, counted per file |
| `stalled-thread` | info | an `active` thread not updated for more than 14 days |
| `open-decision` | info | `open` and `doubted` decisions, and ADRs that wait in the same sense (§5): a list for the human. Not listed for a frozen project: its questions wait for a thaw |
| `frozen-but-live` | warn | `frozen`, but "Where it lives" in the brief or in `local/instance.md` is not empty |
| `leak` | warn | the project layer (brief, decisions, log, threads) holds what looks like a fact about one copy: a home path (`/Users/…/`, `/home/…/`), an IPv4 address that is not part of a version, `~/` in the brief's "Where it lives". One finding per file. Not for private meta (D16): none of it is shared |
| `local-not-ignored` | warn | in a git repository, `.omut/local/` exists and git does not ignore it; `zero` writes `.omut/.gitignore` |
| `local-in-git` | error | files of `.omut/local/` are tracked by git: every clone gets this copy's facts |
| `ghost` | warn | a name from `graveyard.md` is mentioned in live projects' meta (logs don't count) |
| `unfilled` | warn | empty "Why", "What it is" or "Now", an empty `by`, a thread without "Next step": the template was not filled in |
| `pointer` | warn | `AGENTS.md` has no block, `CLAUDE.md` doesn't lead to `AGENTS.md` (a symlink, or `@AGENTS.md` in `CLAUDE.md` or `CLAUDE.local.md`), or the project has no `.omut/README.md`. For private meta (D16): neither `AGENTS.md` nor `CLAUDE.local.md` has the block |
| `scaffold` | error | the root lacks a tool file or a pack meta file; `zero` fixes it |
| `outdated` | info | a tool file (the root's `LEAD.md`, a project's `.omut/README.md`) differs from this version's template; `zero` updates it |

A fresh template is not a shape error but `unfilled` and `unverified`: shape errors are kept for what is broken.

`check` doesn't look inside other artifacts: the format of ADRs and docs is not its business.

## 9. Rituals

The rituals are written in the tool files. Any agent carries them out.

**Inside a project** (`.omut/README.md`):
- **Session start.** Read `brief.md`, the active and parked threads, `open` and `doubted` decisions. If a thread says "Verified", rerun that check before continuing.
- **A decision is made.** An ADR, if the project keeps them; a line in `decisions.md` otherwise.
- **A piece of work is done, or parked.** Update the thread (create it if there is none), "Now" in `brief.md`, a line in `log.md`, before telling the human the work is done and before a commit; what changed only in this copy goes to `local/`. Not "at session end": no agent sees it. The result must be checkable: what was done, what was verified and how, what was not, the next step.
- **Moving the project.** The meta moves with the directory; what breaks is outside it (symlinks into it, absolute paths in agents' settings, virtualenvs, agents' memory kept by path, cron). Go through "Where it lives", search for the old path, record the move.
- **Checking a claim.** Run its `how`, compare, update `seen`; a mismatch goes to the human. Note what was seen in `local/log.md`.
- **Looking for the footprint** when adopting or moving, or to confirm "nothing else runs": search the places footprints hide (agents' skills, MCP configs, agents' settings and memory, tool installs, cron and launchd/systemd, containers, shell profiles); the list is in `.omut/README.md`.
- **Freezing.** A final "Now", a log line, check "Where it lives", then `freeze`.

**In the root** (`LEAD.md`):
- **Overview.** Run `check` and read the `brief.md` of active projects. Tell the human what waits for them: open decisions, doubts, stalled threads, mismatches.
- **Audit.** Check "Where it lives" and "Now" against reality with whatever is at hand (ssh, git, logs). Update `verified`. Don't fix mismatches silently; show them to the human.
- **Project story.** Explain a project in a couple of minutes from `brief.md` and the decisions, without reading the code.
- **Routing.** Match a vague request ("what's up with the site") to a project by "Why" and "What it is".
- **Adopting a project.** `start`, then fill in the meta. Look for history everywhere: git, documents and untracked files, old journals, agents' memory and sessions, neighbouring directories, external documents. Don't invent "Why": what the sources don't say, ask. A fork is a separate project: only accepted decisions carry over from the ancestor. Reconstructed log entries start with `(from session)`; a date the sources don't give stays unknown (`[accepted ?]`, `(date approximate)`); an old journal that is going away moves as is to `log-archive/`. Offer to move project rules from a vendor file into `AGENTS.md`, and, for a public GitHub repository, the badge `start` prints.
- **Moving a project or the pack.** The project ritual for every project moved, plus the symlinks between the pack and the outside world.
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
