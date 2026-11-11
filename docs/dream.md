# The dream: recipes for consolidating the memory

`omut dream` prints a digest — the records written since the last dream, what
`check` found — and a contract for the dream agent. The CLI never runs an
agent and never writes the meta; launching the dream is a recipe, like the
hooks in [hooks.md](hooks.md).

## What a dream is

The memory accumulates: log lines, thread updates, new decisions. A dream is a
session in which an agent re-reads the whole memory — not only the digest —
compares it with reality, and consolidates it: merges what "Now" and the
threads repeat, supersedes decisions reality has outgrown, dates what can rot,
folds finished threads into decisions and the log, verifies the claims it can.
The agent ends with a short report — at most 15 lines — of what it would
change, and changes nothing until the human approves.

The digest is the floor, not the ceiling. If the agent only worked through the
listed signals, it would audit instead of dreaming.

## When it is due

Volume, not the calendar. The counter is deterministic: dated log entries
(`log.md` and `log-archive/`), thread `updated` dates and dated decisions
strictly after `dreamed:` in the brief. At or over the threshold, the startup
hook adds one line asking the agent to offer a dream to the human. A memory
that has never been dreamed counts all of its records, and stays quiet while
they are below the threshold.

The threshold is the brief's own: `dream: 15` in the frontmatter; `10` when
unset. `dreamed: 2026-11-10` is written by the dream agent when it finishes —
never by the CLI — and that is what extinguishes the reminder.

A frozen project gets no reminder: its memory is not growing.

## One-off, by hand

Claude Code:

```sh
omut dream | claude -p "Follow the dream contract."
```

Codex:

```sh
omut dream | codex exec "Follow the dream contract."
```

Review the report the agent comes back with, then either apply it yourself, or
have the agent do it with the approved contract:

```sh
omut dream --apply | claude -p "Follow the dream contract."
```

`--apply` changes the printed contract's words — "the human has approved" —
and nothing else: the CLI still writes nothing. The agent makes the changes,
sets `dreamed: <today>` and records the dream in the log.

In a pack root the dream covers the pack's own memory (brief, decisions,
graveyard); each project dreams separately with `omut dream <project>`.

## On a schedule

The rituals rest on discipline; cron is bolted on from outside, and the CLI
still runs no agent:

```cron
# every evening at 19:00, in the project
0 19 * * * cd ~/oss_projects/hn-parser && omut dream --apply | claude -p "Follow the dream contract." >> /tmp/omut-dream.log 2>&1
```

Schedule a dream of a memory nobody else is writing (D9: one active writer);
a dream colliding with a working agent is what the contract's handoff rule is
for, not what to rely on.

## In a session

An agent that sees the reminder line at startup says so, offers the dream, and
on approval runs `omut dream` in the session and follows the contract. A
report-only dream needs no flags; the human approves, and the same agent
applies it (`omut dream --apply` gives the approved wording) or the human
declines and nothing is written.
