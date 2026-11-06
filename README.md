# omut

[![omut](https://github.com/metravod/omut/actions/workflows/omut.yml/badge.svg?branch=main)](.omut/) [![ci](https://github.com/metravod/omut/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/metravod/omut/actions/workflows/ci.yml)

Project memory that any agent can read, keep and check. A few plain markdown files next to the code say why the project exists, what was decided, what is unfinished and where it actually runs; a small CLI checks that they are filled in and fresh.

> *Omut pamyati* is the Russian name of Dumbledore's Pensieve: you dip into someone's memories and see how it really was.

Status: v0.8, early and used daily on a real pack of projects. Questions and stories of how it went: [Discussions](https://github.com/metravod/omut/discussions).

## Why

Agents build the projects, and you stop remembering what is where and why. Something runs on a server for half a year though it was meant for a week. A project moves from Claude Code to Codex, or to a colleague, or back to you three months later, and the new reader digs through the code to find out what was decided and what comes next.

omut keeps the answers next to the project, in files any agent that reads `AGENTS.md` can keep: no server, no database, no vendor. The CLI never writes the meaning itself; it lays down templates, checks their shape and freshness, and assembles recorded context for the agent.

A [worked handoff on omut itself](docs/handoff.md) shows a fresh session finishing a recorded task, a controlled interruption in a worktree, and what the checks do and do not establish.

## Quick start

```sh
brew install metravod/tap/omut
# or without brew (Linux, macOS, WSL):
curl -fsSL https://raw.githubusercontent.com/metravod/omut/main/install.sh | sh

cd my-project && omut start
```

Ask your agent to inspect the project and fill its memory using the rules in `.omut/README.md`. It reads the code, git history and old notes, fills the brief and the decisions, and asks you what the sources don't say. It also asks once whether its own memory should keep repeating those facts; that memory stays as it is unless you choose otherwise. From then on it keeps the meta as it works: the rules travel inside the repository, so the next agent finds them without omut installed.

Not your repository? See [On a repository that is not yours](#on-a-repository-that-is-not-yours).

Two optional steps:
- **Load the meta at every session start** (Claude Code, in `~/.claude/settings.json`; recipes for Codex, Cursor and git in [docs/hooks.md](docs/hooks.md)):
  ```json
  { "hooks": { "SessionStart": [{ "matcher": "startup|resume|clear|compact",
      "hooks": [{ "type": "command", "command": "omut hook start --agent claude" }] }] } }
  ```
- **The badge.** In a public GitHub repository `omut start` prints a badge for `README.md`: it links readers to `.omut/` and stays green while CI finds the meta fresh and well-formed ([workflow](docs/hooks.md#ci-and-the-badge)). It checks structure and recorded dates; agents and humans still have to verify the contents. This repository wears one.

## On a repository that is not yours

A work project, a team that doesn't use omut yet, someone else's open source: you can't commit `.omut/` there, but your agents still forget what they learned yesterday. Keep the meta for yourself:

```sh
cd their-repo && omut start --private
```

**Connect your agent before the next session.** Claude Code reads the local pointer that `start` creates. For Codex, set up the [session-start hook](docs/hooks.md#codex), review and trust it with `/hooks`, then start a new session. For the main checkout only, a line in `~/.codex/AGENTS.md` also works: *If the repository has .omut/, keep its meta by .omut/README.md.* For linked worktrees, use the user-level hook or place the project hook configuration in each worktree; the hook finds private meta in the main checkout.

- `.omut/` and `CLAUDE.local.md` go into `.git/info/exclude`, which is never committed. The repository's own files, `AGENTS.md` and `.gitignore` included, don't change by a byte: `git status` stays empty.
- The repository's `AGENTS.md` is left alone; private memory needs the agent connection above to be discovered automatically.
- Everything else works as usual: `check`, `hook start`, the claims. The agent is told the meta is private and stays out of commits.
- A clone keeps one memory, in its main checkout: every git worktree finds it through `check` and `hook start`, which gives the agent the full path. `local/` of shared meta works the same way.

When the team sees what it gives, share it: remove the omut lines from `.git/info/exclude`, run `omut zero` (it writes the block into `AGENTS.md`), move what is true only for your copy into `.omut/local/`, and commit `.omut/`. Same format, nothing to migrate.

## What it lays down

```
my-project/
  AGENTS.md              # your rules + a short omut block (CLAUDE.md → AGENTS.md)
  .omut/
    README.md            # the rules, for agents and people (the tool's file)
    brief.md             # why · what · where · now · next; 60 lines / 6000 Unicode characters
    decisions.md         # - D3 [open] Keep collecting after the week? → threads/hn-future.md
    threads/             # unfinished work, each with its next step
    log.md               # - 2026-10-31 · codex/gpt-5 · what was done → link
    local/               # this copy only, never in git (see below)
      instance.md        # this machine, servers, where the secrets are kept, the runbook
      log.md             # deployed, moved, rotated a token
```

**Two layers.** Everything except `local/` goes into git and to everyone who clones: why, what, decisions, the work. `local/` is the memory of your copy, like `.env`: paths on this machine, servers, cron, where the tokens are kept (never the values), how to deploy and roll back. The test for a fact: would a stranger who clones need it, and is it true for them? `omut check` warns when a home path or an IP address slips into the shared layer.

**Claims about reality.** A line about where the project lives says how to check it and when it was last seen:

```
- bot on VPS, systemd unit hn-bot · how: ssh vps systemctl is-active hn-bot · seen 2026-10-28
```

omut never runs it. At session start it offers the agent the claims seen longest ago; the agent checks, updates the date, and brings any mismatch to you instead of fixing it silently. `check` warns about checkable claims nobody has looked at for a month.

**Working together.** One format for one person, a team and open source, no modes. Each piece of work has its own thread with one keeper; log lines from parallel branches merge without conflicts (`.omut/.gitattributes`); decisions name the human who accepted them.

If the project keeps ADRs, they stay the decision registry: point the brief at them (`adr: docs/adr`) and their statuses are read as they are.

The brief has both a line and a volume budget: 60 lines and 6000 Unicode characters; each thread has 80 lines and 8000 characters. Keep the brief about the current state and link to history and detailed evidence. `check` warns when either budget is exceeded; it does not shorten anyone's notes.

## Stop using omut

Your notes remain ordinary files. Keep or archive them before removing anything.

1. Remove only the omut hook entries you installed from your agent and git hook configuration; preserve other hooks. If you added a global agent instruction, remove it only when you no longer need it for other projects.
2. Remove the block between `<!-- omut:start -->` and `<!-- omut:end -->` from `AGENTS.md`, or from `CLAUDE.local.md` for private meta. Preserve everything outside it. Delete a file only if omut created it and nothing else remains. A `CLAUDE.md` pointer to `AGENTS.md` can stay.
3. Keep `.omut/` as notes, archive it elsewhere, or remove it after review. For private meta, keep its `.git/info/exclude` entries while the private files remain; remove only the omut entries after those files are archived or removed. Preserve unrelated excludes. For shared meta, review the removal through git as you would other documentation changes.

After disconnecting hooks, uninstall the CLI with the package manager you used (`brew uninstall omut` for Homebrew), or remove the binary installed by `install.sh` (by default `~/.local/bin/omut`). The notes remain readable without the CLI.

`omut kill` deletes a project's directory; it is not an uninstall command.

## A pack of projects

Most people have more than one. A directory of projects can become a pack with a lead:

```sh
cd ~/projects && omut init      # the pack root: AGENTS.md, .omut/LEAD.md, pack meta
omut start hn-parser            # adopt a directory in the pack
omut check                      # across the pack: what is stale, unfilled, waiting for you
omut context                    # assembled overview, with source locations
omut context hn-parser          # full current records of one project
```

An agent started in the root is the lead: it knows the whole pack and tells you what is where. Started in a project, it keeps that project's meta.

The lead's startup hook supplies the same overview as `omut context`: pack context, project status and current brief sections, waiting decisions, unfinished thread summaries, check findings and local Git observations. The CLI collects literal records with source paths and lines; the agent interprets them. It needs no model calls to discover and assemble the files.

Git observations include the branch/upstream, ahead/behind from local refs, changed/staged/untracked paths, renames and conflicts. The lead can use this for an ordinary overview and open a specific diff when it matters. Changes are scoped to the project; branch/divergence describe its repository. No fetch or full diff is included. Private worktrees show the working copy's status while keeping meta in the main checkout. Projects without a Git working tree say so; Git failures are explicit and mark the output incomplete. Git status reads disable optional index refresh and fsmonitor hooks.

The overview omits project footprints, local memory, logs and decided project records; `omut context <project>` includes the full brief and decisions, unfinished threads, waiting ADRs, filled local memory and the latest five entries from each log. Completed threads and older history remain in their source files. Read/Git failures and shape errors mark the output incomplete and return exit 1; warnings remain visible with exit 0. `--json` exposes the same selection for other tools. This reduces file-discovery calls; token savings depend on how much context is selected.

| Command | What it does |
|---|---|
| `init [dir]` | create a pack |
| `start [project] [--private]` | a new project, or adopt an existing directory; `--private` keeps the meta out of git |
| `freeze` / `thaw [project]` | freeze / unfreeze |
| `kill <project> --yes --reason "..."` | remove, write a tombstone, list the mentions; refuses over unsaved work |
| `check [project] [--json] [--fail-on warn]` | the report; non-zero exit on shape errors |
| `context [project] [--json]` | assembled pack overview or full current project records, with sources |
| `zero [project]` | repair the scaffold, update the tool files |
| `hook start\|commit` | session context for agents, a git pre-commit reminder ([recipes](docs/hooks.md)) |

`check`, `context`, `zero`, `freeze` and `thaw` work on a single repository without a pack, from any subdirectory or git worktree of it. In a subdirectory, `omut start` won't lay a second meta by accident; `omut start .` gives it its own, for a part of a monorepo. The templates are plain markdown in [`templates/`](templates/), so all of it can be set up by hand too.

## Install

macOS with Homebrew:

```sh
brew install metravod/tap/omut
```

Linux, macOS and WSL without brew:

```sh
curl -fsSL https://raw.githubusercontent.com/metravod/omut/main/install.sh | sh
```

It takes the latest release for your OS and CPU, checks it against `checksums.txt` and puts `omut` into `~/.local/bin`; `OMUT_DIR=/usr/local/bin` or `OMUT_VERSION=0.4.1` before `sh` change that. The script is short: [read it](install.sh) before piping it to a shell.

With [mise](https://mise.jdx.dev) (it takes a release once it is a day old, by its own default) or Go:

```sh
mise use -g github:metravod/omut
go install github.com/metravod/omut/cmd/omut@latest
```

Binaries for macOS, Linux and Windows are on the [releases page](https://github.com/metravod/omut/releases); Windows itself is built but not yet tried, WSL works like Linux. From source: `go build ./cmd/omut && go test ./...` (Go standard library only).

## Docs

- `omut help <command>`: what a command writes, leaves alone and refuses, right in the terminal. `omut help meta`: the rules of the meta, before anything is laid down.
- [Specification v0](docs/spec-v0.md)
- [Hooks for git, CI, Claude Code, Codex, Cursor, and the badge](docs/hooks.md)
- [Competitors and neighbours](docs/research/competitors.md)
- [Checking the meta against reality: the variants weighed](docs/research/reality-check.md)

This repo keeps its own meta by its own spec: see [`.omut/`](.omut/).

## Credits

The idea grew out of using [drift](https://github.com/snowtema/drift).

## License

[MIT](LICENSE)
