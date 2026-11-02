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
Statuses: `open` (not decided, there are options), `accepted`, `doubted` (decided, but there is a doubt), `superseded → Dn`. If the project has ADRs, the line links to the ADR. If not, the reason goes right into the line, 1–3 sentences; if it doesn't fit, that's a reason to write an ADR. Lines are never deleted; a decision changes through `superseded`. A neighbouring project's decision is a link: `→ ../<neighbour>/.omut/decisions.md`.

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
- **A decision is made.** A line in `decisions.md`; an ADR too, if the project keeps them.
- **Session end or parking.** Update the thread (create it if there is none), "Now" in `brief.md`, a line in `log.md`. The result must be checkable: what was done, what was verified and how, what was not verified, the next step.
- **Freezing.** A final "Now", a log line, check "Where it lives", then `omut freeze <project>`, only with the human's consent.

### Adopting a project

A project without meta (`omut start` creates the templates):
- **Look for history everywhere, not only in the code:** git, the project's documents and untracked files, old journals (`.drift/` and the like), agents' memory and sessions, neighbouring directories with a similar name, external documents. What only one agent's memory holds goes into the meta: other agents can't see it.
- **Don't invent "Why" or the reasons for decisions.** What the sources don't say, write as unknown ("not known, ask") and ask the human.
- **The log does not retell commits.** Entries reconstructed from sessions are fine, marked `(from session)`. Don't copy an old journal; link to it in the first entry.
- **A fork is a separate project.** Only accepted decisions carry over from the ancestor; goals, loose ends and footprints stay with it.
- **Project rules belong in `AGENTS.md`.** If they live in one agent's file (`CLAUDE.md` and the like), offer the human to move them to `AGENTS.md` and make the vendor file a pointer (`ln -s AGENTS.md CLAUDE.md`).

### Don't

- Only one agent keeps a project's meta at a time. Parallel work needs an explicit split of responsibility.
- Don't copy project artifacts (ADRs, plans, docs) into the meta; link to them.
- Don't fix a mismatch between the record and reality silently; show it to the human.
- If the repo is distributed (a skill, a package, a template), committed meta ships to its users: decide whether `.omut/` belongs in `.gitignore`.
