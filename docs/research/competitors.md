# Competitors and neighbours

A snapshot as of 2026-11-01. The repositories were read from source (shallow clones), not just from their descriptions.

2026-11-07 addendum: Memorilla reviewed through its README, paper abstract, evaluation guide and `memorilla/memory.py` / `memorilla/vllm.py` via GitHub. No model execution or independent benchmark replication.

## Summary

| Project | Level | What it stores | What holds it together | Truth checking | Pack of projects |
|---|---|---|---|---|---|
| **agent_collab_treaty** | one repo | `AGENTS.md`, `project_overview.md`, `next_steps.md`, `work_log.md` + archive | discipline + a Python CLI over Copier (`init`/`update`/`validate`) | no, shape only | no |
| **Cline Memory Bank** | one project | 6 md files: brief, product, active context, patterns, tech, progress | an instruction in Cline's rules | no; size and staleness are not discussed | no |
| **planning-with-files** (~26k ⭐) | one task | `task_plan.md`, `findings.md`, `progress.md`; gitignored, overwritten | 3–8 hooks for each of 19+ hosts | no | no |
| **codex-smart-project-memory** | pack | a separate memory tree: 7 starter files in the root + 9 per project | a Codex skill + Python scripts | `validate` only checks that files exist | a manual registry + a manual router |
| **projectmem** (`pjm`) | repo + shared view | typed events (issue/attempt/fix/decision) in `.projectmem/events.jsonl` | MCP (17 tools), git hooks, a file watcher, Python | yes: a decision that names a file is marked stale if the file changed since | a dashboard built on read |
| **projmem** (m4ll0k) | code | "beliefs" about the code, tied to an index | a CLI + a `PreToolUse` hook that blocks actions | yes: beliefs are checked against the live code index | no |
| **Codex-Workspace** | pack of repos | docs, manifests, a cache, a Hub app (React + API) | a platform: start/stop services, previews | partly: the Hub sees runtime state | yes, through a heavy app |
| **Pensieve** (~2.5k ⭐) | one repo | 4 layers in `.pensieve/`: maxims (MUST), decisions (WANT), pipelines (HOW), knowledge (IS) + `short-term/` with a TTL; a graph of `[[...]]` links | a skill + ~5k lines of bash/Python, Claude Code hooks, writes into auto memory | no: `doctor` checks frontmatter, links and layout; a decision has an "invalidation condition", but nothing checks it | no; only the system code is shared |
| **beads** (`bd`) | tasks | a task graph (now on Dolt) | a CLI | no | no |
| **Claude Code auto memory / Tasks** | agent | notes and tasks in `~/.claude` | built in | no | no; tied to the machine and the vendor |
| **Memorilla** (reviewed 2026-11-07) | model input / document collection | source documents encoded as embeddings, query-conditioned latent memory vectors | trained memory module + encoder + compatible frozen decoder; published CUDA/Python implementation | evaluated for answer/task performance; no project-record freshness workflow in reviewed interface | no project-pack workflow in reviewed interface |

## Each one: what we take and what we avoid

### agent_collab_treaty: closest at the project level
Actively maintained: last release 2026-10-29, 16 public adopters. Versions are tracked, updates go through Copier.

**We take:**
- **Splitting files into "yours" and "ours".** `AGENTS.md` belongs to the user, `treaty_conventions.md` is maintained upstream, so `treaty update` hardly ever conflicts. That is the recipe for `zero` as the update path.
- **Log rotation by a deterministic rule.** The live file keeps the last 5 dates; older ones go to `work_log_archive/`.
- **A log entry names the model** and has a "Verification" part: the commands that were actually run.
- **Don't reconstruct the log from git after the fact.** It produces noise like "implemented X".
- **`AGENTS.md` must sit where the agent looks for it up the tree.** Other tools get pointer files: a CLAUDE.md with an import, a Cursor rule.
- **Live examples instead of placeholders.** The best part of their README.

**We avoid:**
- **Their own `next_steps.md` → "Currently Hot" grew into a wall of text,** a paragraph per item, despite the author's discipline. drift's notes went the same way. Conclusion: without a size budget enforced by `check`, meta bloats.
- **Decisions live inside the log.** Collecting "what is still open" means reading the whole log.
- **Python + Copier for templates.** Too heavy a dependency for the job.

### projectmem: strongest at truth checking
Actively maintained (last commit 2026-10-30), has an arXiv paper.

**We take:**
- **The pack view is built on read and never stored** ("a global *view*, never a global *store*"). Same as our "no registries".
- **A stale record is flagged, not deleted:** "predates 7 commits — confirm or supersede". The human decides.
- **`doctor` finds projects without meta.**
- **`brief`: a "where was I" screen.**

**We avoid, and how we differ:**
- **Their checks are at the code level:** a record against the git history of a file. Ours are at the project level: the declared intent against reality, "Where it lives", how fresh the brief is.
- **Mandatory MCP, git hooks and a watcher.** Without git and MCP the system doesn't work.
- **Projects without git and projects that aren't code** (research, an article, a server) stay out of sight.

### planning-with-files: a different layer
The working memory of one task: overwritten, gitignored. The authors themselves say the valuable parts should move into documentation. It complements us rather than competes. Our threads are a level above: the project.

**We take:**
- **When the cwd is ambiguous because of nested projects, inject nothing rather than guess.** For us: the lead only in the root; inside a project, the project's meta applies.
- **An explicit single-writer mode.** Without it, parallel sessions overwrote each other's plan; their issues discuss it.

**We avoid:**
- **Portability through hooks costs a huge machinery:** 3–8 hooks for each of 19+ hosts and separate tooling to keep them consistent. Our hooks are optional: portability rests on files and `AGENTS.md`.

### codex-smart-project-memory: closest in the pack idea
Only 12 commits, the README is mostly SEO. Worth a look because an outside report called it a direct analogue.

**We take:**
- **Routing a vague request to a project** ("what's up with the site"). We don't need a separate file for it: the lead routes by the "Why" sections of the project briefs.

**We avoid:**
- **The meta lives in a separate `P01_*` tree, and the path to the code is a column in a manual registry.** The meta drifts away from the project, the registry rots (drift's registry did exactly that).
- **7 starter files in the root + 9 per project.** The reading chain costs more than it gives.
- **`validate` only checks whether files exist.**
- **Tied to Codex:** `agents/openai.yaml`, invoked as `$skill`.

### Pensieve: closest on decisions
`kingkongshot/Pensieve`, MIT, 2.5k stars. Grew out of a Linus-style prompt ("good taste", "don't break userspace"). The last commit to main and zh is from 2026-05-08; the stars keep growing. Pitches itself as "a self-growing CLAUDE.md that runs as a skill".

**We take:**
- **A decision has an "Invalidation condition"** (when to revisit it) and "What to ask less next time". The invalidation condition is ready material for checking: an agent compares it with reality the way it does "Where it lives". For now an open question (spec §12), not a rule.
- **Alternatives are required.** "Alternatives Considered" explains three months later why the obvious option was not taken.
- **Facts are separate from choices.** IS (knowledge, verifiable facts) is apart from WANT (decisions). Our facts live in "Where it lives" and "Now", decisions in `decisions.md`: the same line.
- **`doctor` is read-only and prints a fixed-format report with MUST_FIX.** Same as our `check`; repairs are a separate tool.

**We avoid:**
- **A decision is a separate file with 9 sections.** A high bar for writing one, and `decisions/` grows in files. Ours is one line, `- Dn [status] text → link`, with details behind the link only when needed.
- **`short-term/` with a 7-day TTL and manual promotion via `mv`.** One more queue to work through. Same class of problem as drift's notes: records pile up faster than anyone reviews them.
- **A knowledge graph** rebuilt by hooks after every Write/Edit. Machinery, not meta a human reads.
- **Tied to Claude Code:** hooks in `~/.claude/settings.json`, `PreToolUse` injection into Explore/Plan agents, edits to `~/.claude/projects/*/memory/MEMORY.md`. Other clients are told to "run `self-improve` manually".
- **The system code is installed by `git clone` into the skills directory** and updated with `git reset --hard`.
- **One repo is the level.** No pack; only maxims are shared between projects.

### Codex-Workspace: a platform
A Hub that starts and stops services, manifests, a shared cache, "loop engineering".

**We take:**
- **The idea that a project's runtime state is part of the pack's picture.**

**We avoid:**
- **A platform that has to be run.** Our "Where it lives" is a record the agent checks against reality, not a control panel.

## Memorilla addendum (2026-11-07)

[Memorilla](https://github.com/snap-stanford/memorilla) shares the broad goal of using accumulated information beyond the current conversation. Its [paper](https://arxiv.org/abs/2610.21375) describes reusable document embeddings and a learned module that produces a fixed number of query-dependent vectors in the frozen LLM's input embedding space. The default is 16 memory slots; these are continuous vectors, not sixteen readable words or a portable summary.

The distinction is visible in the implementation: [`MemoryModule.forward`](https://github.com/snap-stanford/memorilla/blob/main/memorilla/memory.py) projects into the decoder embedding dimension; [`MemoryVLLM.generate`](https://github.com/snap-stanford/memorilla/blob/main/memorilla/vllm.py) inserts those vectors into prompt embeddings. The [evaluation guide](https://github.com/snap-stanford/memorilla/blob/main/docs/evaluation.md) requires the decoder used to train the checkpoint; released checkpoints use Qwen3-8B. Thus this published mechanism is not a memory file one can transfer unchanged between Claude Code and Codex through their ordinary text interfaces.

omut's artifact is explicit project state: intent, decisions with reasons, unfinished work, verification records and next steps, maintained by agents and readable by people. Its CLI scaffolds, checks shape/recorded freshness and assembles literal records; it does not learn to compress source history. Memorilla supplies a mechanism for querying a document collection, without defining this project's record-keeping contract in the reviewed interface.

**Real overlap:** reducing the cost of recovering useful context from accumulated history. A sufficiently capable automatic memory system could substitute for some manual project-memory work; different implementation layers alone do not eliminate competition for that need.

**Potential complement, not implemented:** omut records could be source documents for a compatible inference system. This would require integration and validation; it is not a current feature or a reason to add GPU/model dependencies to the CLI.

**Positioning implication:** describe portable project memory, reasons for decisions and continuation across agents, rather than claiming that omut invented AI memory or provides neural compression/token savings. No comparative quality or efficiency claim follows from either project's reported experiments.

## Positioning

Every one of our pieces exists somewhere already. The combination is what's new:

1. **The level is a pack of any projects,** not a repo and not a task. Projects without git and projects that aren't code are equal citizens.
2. **The meta lives inside the project; the pack view is assembled on read.** No registry, no router, no separate memory tree.
3. **The project level is what gets checked:** the declared intent against reality, "Where it lives" against what actually runs, and how fresh the brief is. Mismatches go to the human.
4. **Minimal:** 4 kinds of files per project, 7 commands, no runtime, hooks optional.
5. **Agent independence** rests on `AGENTS.md` and the CLI, with no MCP and no vendor directories.

## Links
- https://github.com/yzhaoinuw/agent_collab_treaty
- https://docs.cline.bot/features/memory-bank
- https://github.com/OthmanAdi/planning-with-files
- https://github.com/richenyu/codex-smart-project-memory
- https://github.com/riponcm/projectmem
- https://github.com/m4ll0k/projmem
- https://github.com/RichardGeorgeDavis/Codex-Workspace
- https://github.com/kingkongshot/Pensieve
- https://github.com/gastownhall/beads
- https://github.com/snowtema/drift
- https://github.com/snap-stanford/memorilla
