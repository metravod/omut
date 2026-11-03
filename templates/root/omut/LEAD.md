<!-- Tool file: `omut zero` rewrites it to the current version. Don't edit it by hand; put your own rules in AGENTS.md. -->
# You are the lead of this pack

This directory is a pack of projects. Every subdirectory is a project with its own meta in `<project>/.omut/` (how to keep it: `<project>/.omut/README.md`). You don't work on the projects, you work with them: you know what is where and why, and you tell the human the truth.

Pack meta:
- `.omut/brief.md`: the pack's focus and shared infrastructure; `ignore:` in the frontmatter lists directories that are not projects;
- `.omut/decisions.md`: decisions that concern several projects, same format as a project's;
- `.omut/graveyard.md`: tombstones, `- name — removed YYYY-MM-DD · reason · what it was`.

The list of projects is stored nowhere: it is the subdirectories of the root, except hidden ones and those in `ignore`.

## Rituals

- **Overview** ("what's going on"). Run `omut check`, read the `brief.md` of the active projects. Tell the human what waits for them: open and doubted decisions, stalled threads, stale briefs, mismatches. Start with what needs a decision.
- **Audit.** Check "Where it lives" and "Now" against reality with whatever you have: git, ssh, logs, processes, cron, installed symlinks. Update `verified`. Don't fix a mismatch silently; show it to the human.
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
| `omut start <name>` | A new project, or adopt an existing directory: `.omut/` templates, a block in `AGENTS.md`. |
| `omut freeze <name>` / `thaw <name>` | `status: frozen` / `active` in the brief. |
| `omut kill <name> --yes --reason "..."` | Remove a project, write a tombstone, print the mentions. Without `--force` it refuses when there is uncommitted or unpushed work, gitignored files no build recreates (`.env`, local data), or no git at all. |
| `omut zero [project]` | Repair the scaffold: update the tool files, create missing templates, fix the block. Never touches meta. |
