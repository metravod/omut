---
status: active
updated: 2026-11-01
by: claude-code/opus-5.5
---
# Specification v0
**Goal:** pin down the substance and the format before and alongside the code, so they don't drift apart.
**Where we got:** docs/spec-v0.md covers the substance, principles, non-goals, layout, project and pack meta, commands, check rules, rituals and compatibility. Decisions D1–D10. The format was laid out by hand on two real projects: one migrated from drift, one with no journal at all.
**Verified:** competitors read from source → docs/research/competitors.md. Both real briefs fit the budgets (≈30 lines, 9–12 decisions).
**Not verified:** the format on a project without git or one that isn't code; a handoff to another agent (Codex) on this meta.
**Open:** spec §12; findings 2, 4, 6 and 7 below.
**Next step:** hand a laid-out project to Codex and ask "what is this and what's next".

## Findings from laying out real projects
Folded into the spec and conventions.md unless noted.
1. **A parking reason can't be recovered after the fact.** A thread sat untouched for two months and nothing said why. Hence `parked_because: unknown`, allowed only for reconstructed threads.
2. **Migrating with an existing journal.** drift's notes are real history, not git noise: link to them instead of copying. Maybe a `history:` field in the brief so `check` knows about the old journal. *Open.*
3. **Decisions depend on neighbouring projects.** One project's decision rested on another's decision to wind down. A link form for neighbours (`→ ../other/.omut/decisions.md#Dn`), and an argument for pack-level decisions (spec §12, q. 7).
4. **"Where it lives" is not only servers.** For a tool, the footprint is its installs: a symlink into an agent's skills directory. Freezing or killing it without knowing that breaks the skill in every session. The template lists the kinds of footprints. *Whether `check` should see installs is open.*
5. **Meta in a repo that is distributed** (a skill, a package): committed meta ships to users. The user decides (D3); conventions.md warns.
6. **stale-brief pays off.** A journal had stopped months ago, followed by a dozen commits, a repositioning and a v2. "Changes newer than updated" would have caught it at once. *Kept as is.*
7. **An audit finds risks, not just status:** a backup with private prompts outside the public repo's `.gitignore`. For now an item in "Next". Maybe risks deserve their own place, without adding a new kind of file. *Open.*
8. **"No journal" doesn't mean "no history".** It was scattered across five places: a sibling repo's journal, an agent's auto memory, session transcripts, untracked docs and an external document. Adoption lists where to look.
9. **One agent's memory is invisible to the others.** The most valuable "why" sat in Claude Code's auto memory, which Codex can't see. Adoption moves vendor memory into the meta.
10. **A transcript is not git.** Entries reconstructed from sessions carry substance (which decision and why), so they are allowed, marked `(from session)`. What is banned is retelling commits.
11. **A fork is a separate project.** Only accepted decisions carry over from the ancestor; goals, loose ends, footprints and the log stay with it. Also: Claude Code encodes `_` and `-` the same in `~/.claude/projects`, so sibling directories share memory and sessions; filter sources by `cwd`.
12. **Parking reasons were missing "deliberately put aside"**: hence `priority`, alongside `unknown`.
13. **Project rules lived in `CLAUDE.md` with no `AGENTS.md`.** Codex wouldn't see them, and Claude Code wouldn't see a new `AGENTS.md`. Adoption offers to move the rules to `AGENTS.md` and make `CLAUDE.md` a symlink, never without asking.
14. **A side project inside another project's directory**, gitignored, with its own open items. The pack overview can't see it; an audit should ask about nested projects.
