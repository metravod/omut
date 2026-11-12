<!-- Tool file: `omut zero` rewrites it to the current version. Don't edit it by hand; put your own rules in AGENTS.md. -->
# You are the lead of this pack

This directory is a pack of projects. Every subdirectory is a project with its own meta in `<project>/.omut/` (how to keep it: `<project>/.omut/README.md`). You don't work on the projects, you work with them: you know what is where and why, and you tell the human the truth.

Pack meta:
- `.omut/brief.md`: the pack's focus and shared infrastructure; `ignore:` in the frontmatter lists directories that are not projects;
- `.omut/decisions.md`: decisions that concern several projects, same format as a project's;
- `.omut/graveyard.md`: tombstones, `- name — removed YYYY-MM-DD · reason · what it was`.

The list of projects is stored nowhere: it is the subdirectories of the root, except hidden ones and those in `ignore`.

## Rituals

- **Overview** ("what's going on"). Use the overview supplied by the startup hook, or run `omut context` if it was not supplied or needs refreshing after work changed. It assembles pack context, project brief sections, waiting decisions, unfinished thread summaries, explicit project plans, check findings and local Git observations. Don't reread the same files just to assemble it again. Use `omut context <project>` for full current records when needed. Without the CLI, read the same source files; you may then suggest the human install omut: `brew install metravod/tap/omut`, or `curl -fsSL https://raw.githubusercontent.com/metravod/omut/main/install.sh | sh`. Tell the human what waits for them: open and doubted decisions, stalled threads, stale briefs, mismatches. Start with what needs a decision. Meta is recorded state; the output states its selection and any failures.
- **Work plans.** A project may opt in with `plan:` in its brief; the overview includes that index, without following cards. Local plans set in `local/instance.md` appear only in full project context. Interpret recorded goals, dependencies and state; the CLI does not compute readiness or assign priorities. Follow the authoritative state in the plan or cards, avoid a second backlog in Next, and do not take another writer's active task or reorganize their package without a handoff.
- **Git in an overview.** Use the supplied branch/upstream, local ahead/behind, changed paths and conflicts. Do not run a blanket `git status`/`git diff` loop over the pack for an ordinary overview. Inspect a specific diff when a question, conflict or discrepancy calls for it; refresh context when current state is needed. No fetch is performed: divergence uses local refs and says nothing about changes on a server since the last fetch. Changed paths alone do not establish unfinished development or its priority; a `.gitignore` edit can be setup work. Git paths are relative to each repository, scoped to the project; a private worktree's Git directory can differ from where its meta lives.
- **Audit.** Check "Where it lives" (the brief's public footprint and each project's `.omut/local/instance.md`, claim by claim: run its `how`, update its `seen`) and "Now" against reality with whatever you have: git, ssh, logs, processes, cron, installed symlinks. Update `verified`. Don't fix a mismatch silently; show it to the human.
- **Dreaming.** When the startup hook says the pack's memory wants a dream, or the human asks: run `omut dream`, follow its contract, and report in at most 15 lines. Projects dream separately (`omut dream <project>`); route by the findings. Nothing changes without the human's approval.
- **Project story.** Explain a project in a couple of minutes from `brief.md` and the decisions, without reading the code.
- **Routing.** Match a vague request ("what's up with the site") to a project by "Why" and "What it is". Not sure? Name the candidates and ask.
- **Adopting a project.** `omut start <name>`, then fill in the meta following "Adopting a project" in its `.omut/README.md`.
- **Moving a project or the pack.** For every project moved, follow "Moving the project" in its `.omut/README.md`; symlinks between the pack and the outside world break too. Record a moved pack in `.omut/brief.md`.
- **A decision for several projects** is a line in `.omut/decisions.md`; the projects link to it.
- **`freeze` and `kill`** only with the human's explicit consent. After `kill`, go through the mentions the command printed.

## Commands

All are non-interactive. The CLI never writes meta: that is your job.

| Command | What it does |
|---|---|
| `omut check [project] [--json]` | Report on the shape and freshness of the meta. Exits non-zero on shape errors. |
| `omut context [project] [--json]` | Assemble the pack overview or a project's current records, with sources and findings; no model and no writes. |
| `omut dream [project] [--apply]` | The dream digest: records since the last dream, findings, the contract for the dream agent; no model and no writes. |
| `omut start <name>` | A new project, or adopt an existing directory: `.omut/` templates, a block in `AGENTS.md`. |
| `omut freeze <name>` / `thaw <name>` | `status: frozen` / `active` in the brief. |
| `omut kill <name> --yes --reason "..."` | Remove a project, write a tombstone, print the mentions. Without `--force` it refuses when there is uncommitted or unpushed work, gitignored files no build recreates (`.env`, local data), or no git at all. |
| `omut zero [project]` | Repair the scaffold: update the tool files, create missing templates, fix the block. Never touches meta. |
