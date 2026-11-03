<!-- Tool file: `omut zero` rewrites it to the current version. Don't edit it by hand; put the project's own rules in AGENTS.md. -->
# Project meta

This directory holds the project's meta in the [omut](https://github.com/metravod/omut) format: why the project exists, what state it is in, what comes next, and why it is built the way it is. Agents keep it current as they work, so the next agent or person can pick the project up without digging through the code.

## Reading it

1. `brief.md`: why, what it is, where it runs, what works now, what comes next. At most 60 lines; start here.
2. `decisions.md`: one line per decision, with the reason. `open` and `doubted` ones wait for an answer.
3. `threads/`: unfinished work, each thread with its next step.
4. `log.md`: one line per session, newest first.

Reading needs nothing installed. If you are only reading, you don't write anything here.

## Keeping it

If you work on the project, keep the meta by the rules below. `omut check` checks their shape and freshness, but the rules work without the tool.

Headings, field names and labels stay as below: `check` looks for them. Write the text itself in the language the project's people use.

### Files

**`brief.md`: the project's brief. Always read, at most 60 lines.**
Frontmatter: `status` (`active` | `frozen`), `updated` (when the text changed), `verified` (when "Where it lives" and "Now" were last checked against reality), `by` (`agent/model`), optional `adr` (the project's ADR directory).
Sections, all required:
- **Why**: the original intent, 1–3 lines. Never rewritten after the fact: a changed intent is a decision in `decisions.md`.
- **What it is**: how it works, 3–10 lines, for a human. Links to the project's key documents.
- **Where it lives**: the footprint in reality, with dates: servers, cron and timers, deploys, installs (symlinks, global skills, packages), data, external documents, money. Empty means it runs nowhere.
- **Now**: what works, what is broken. Lines `**Verified:**` (how) and `**Not verified:**`.
- **Next**: 3–7 items. Small ideas live here too.

**`decisions.md`: the decision registry.** One line per decision:
`- D7 [accepted 2026-09-10] What was decided, why, what was rejected → link`
Statuses: `open` (not decided, there are options), `accepted`, `doubted` (decided, but there is a doubt), `superseded → Dn`. If the project has ADRs (`adr:` in the brief), they are the registry for the decisions they record: don't copy them here; `decisions.md` holds the decisions without an ADR, and a line at its top says where the ADRs are. `check` reads an ADR's status from frontmatter, a `Status:` line or a `## Status` section: `proposed` or `draft` waits for the human like `open`, `Firmness: tentative` like `doubted`, no status counts as decided. A registry that already mirrors its ADRs may stay as it is; new ADRs are not copied. Without ADRs, the reason goes right into the line, 1–3 sentences; if it doesn't fit, that's a reason to write an ADR. Lines are never deleted; a decision changes through `superseded`. A neighbouring project's decision is a link: `→ ../<neighbour>/.omut/decisions.md`.

**`threads/<slug>.md`: a thread of work**, only for work on substance (ideas without work go to "Next"). At most 80 lines.
Frontmatter: `status` (`active` | `parked` | `done` | `dropped`), `updated`, `by`, and for `parked`, `parked_because`:
- `stamina`, `limits`: continue from the same spot;
- `doubt`: resolve the doubt first (often together with a `doubted` decision);
- `blocked`: waiting on something external;
- `priority`: deliberately put aside for something else;
- `unknown`: the reason was never recorded; only for threads reconstructed during adoption.

Body: `# Title`, then the lines `**Goal:**`, `**Where we got:**`, `**Verified:**` / `**Not verified:**`, `**Open:**`, `**Next step:**`.

**`log.md`: history, newest first.**
`- 2026-10-31 · claude-code/opus-5.5 · What was done; verified: how → link`
One or two lines per session, details by link to a thread or a decision. The author is `agent/model` or `human`. Past 50 entries, old ones move to `log-archive/<year>.md`.

### Rituals

- **Session start.** Read `brief.md`, the `active` and `parked` threads, the `open` and `doubted` decisions. If a thread says "Verified", rerun that check before continuing.
- **A decision is made.** An ADR, if the project keeps them; a line in `decisions.md` otherwise.
- **A piece of work is done, or parked.** Update the thread (create it if there is none), "Now" in `brief.md`, a line in `log.md` (more work the same session extends that line). Do it before you tell the human the work is done, and before a commit: an agent doesn't see the session end, so waiting for it means never. The result must be checkable: what was done, what was verified and how, what was not verified, the next step.
- **Moving the project.** The meta moves with the directory; what breaks is outside it: symlinks into it, absolute paths in agents' settings and permission rules, virtualenvs, agents' memory kept by path, cron and services. Go through "Where it lives", search for the old path, fix what you can and show the human the rest; record the move in "Where it lives" and the log.
- **Freezing.** A final "Now", a log line, check "Where it lives", then `omut freeze <project>`, only with the human's consent.

### Adopting a project

A project without meta (`omut start` creates the templates):
- **Look for history everywhere, not only in the code:** git, the project's documents and untracked files, old journals (`.drift/` and the like), agents' memory and sessions, neighbouring directories with a similar name, external documents. What only one agent's memory holds goes into the meta: other agents can't see it.
- **Don't invent "Why" or the reasons for decisions.** What the sources don't say, write as unknown ("not known, ask") and ask the human.
- **The log does not retell commits.** Entries reconstructed from sessions are fine: the author is whoever writes the line, the text starts with `(from session)`. Don't copy an old journal into `log.md`: if it stays, link to it in the first entry; if it is going away, move it as is to `log-archive/` (any file name; `check` doesn't read the archive) and link to that.
- **A date the sources don't give stays unknown.** A decision says so in its status: `[accepted ?]`. A log entry takes the latest date it can be pinned to and says `(date approximate)`; history with no dates at all goes to `log-archive/` as is.
- **A fork is a separate project.** Only accepted decisions carry over from the ancestor; goals, loose ends and footprints stay with it.
- **If the repository is public on GitHub,** offer the human the omut badge for `README.md`: `omut start` prints it, with a link to the CI recipe.
- **Project rules belong in `AGENTS.md`.** If they live in one agent's file (`CLAUDE.md` and the like), offer the human to move them to `AGENTS.md` and make the vendor file a pointer (`ln -s AGENTS.md CLAUDE.md`).

### Don't

- Only one agent keeps a project's meta at a time. Parallel work needs an explicit split of responsibility.
- Don't copy project artifacts (ADRs, plans, docs) into the meta; link to them.
- Don't fix a mismatch between the record and reality silently; show it to the human.
- If the repo is distributed (a skill, a package, a template), committed meta ships to its users: decide whether `.omut/` belongs in `.gitignore`.
