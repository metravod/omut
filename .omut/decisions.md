# Decisions

- D1 [accepted 2026-10-31] Agent- and provider-agnostic: state in files, entry through AGENTS.md and the CLI, hooks only as adapters. Rejected: relying on Claude Code mechanisms (auto memory, Tasks, ~/.claude/projects), since people mix agents and hand work between them
- D2 [accepted 2026-10-31] One pack root directory; the lead is any agent started in the root, not a separate program
- D3 [accepted 2026-10-31] Where to keep it, what to commit and how to sync is the user's call; the system only shapes the meta
- D4 [accepted 2026-10-31] The pack view is assembled on read, there is no registry. Rejected: a registry like drift's, which went stale (active projects missing, empty ones listed)
- D5 [accepted 2026-11-01] Project meta: brief / decisions / threads / log in a hidden `.omut/`. Agents get exact paths from the AGENTS.md block, so hiding costs nothing; listings stay clean. Rejected: a visible `lead/`; threads with an `idea` status
- D6 [accepted 2026-10-31] A decision registry instead of owning ADRs: a line with a status and a link to the project's ADR, or a short reason right in the line
- D7 [accepted 2026-10-31] Commands: init, start, freeze, thaw, kill, check, zero; no commands that write meta. kill writes a tombstone instead of scrubbing mentions
- D8 [accepted 2026-11-01] Go, one static binary. Rejected: Python and Node, which pull in a runtime and contradict "no runtime of its own"
- D9 [accepted 2026-11-01] One active writer of the meta; parallel work only with an explicit split of responsibility
- D10 [accepted 2026-11-01] Templates, CLI messages and docs in English; the meta's text is written in whatever language the project's people use, headings and labels stay fixed for `check`
