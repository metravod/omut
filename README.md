# omut

[![omut](https://github.com/metravod/omut/actions/workflows/omut.yml/badge.svg?branch=main)](.omut/) [![ci](https://github.com/metravod/omut/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/metravod/omut/actions/workflows/ci.yml)

Project memory that any agent can read, keep and check. A few plain markdown files next to the code say why the project exists, what was decided, what is unfinished and where it actually runs; a small CLI checks that they are filled in and fresh.

> *Omut pamyati* is the Russian name of Dumbledore's Pensieve: you dip into someone's memories and see how it really was.

Status: v0.4, early and used daily on a real pack of projects. Questions and stories of how it went: [Discussions](https://github.com/metravod/omut/discussions).

## Why

Agents build the projects, and you stop remembering what is where and why. Something runs on a server for half a year though it was meant for a week. A project moves from Claude Code to Codex, or to a colleague, or back to you three months later, and the new reader digs through the code to find out what was decided and what comes next.

omut keeps the answers next to the project, in files any agent that reads `AGENTS.md` can keep: no server, no database, no vendor. The CLI never writes the meaning itself; it lays down templates and checks the shape and freshness of what agents wrote.

## Quick start

```sh
brew install metravod/tap/omut
# or without brew (Linux, macOS, WSL):
curl -fsSL https://raw.githubusercontent.com/metravod/omut/main/install.sh | sh

cd my-project && omut start
```

Then tell your agent: *"adopt this project following .omut/README.md"*. It reads the code, git history and old notes, fills the brief and the decisions, and asks you what the sources don't say. From then on it keeps the meta as it works: the rules travel inside the repository, so the next agent finds them without omut installed.

Not your repository? See [On a repository that is not yours](#on-a-repository-that-is-not-yours).

Two optional steps:
- **Load the meta at every session start** (Claude Code, in `~/.claude/settings.json`; recipes for Codex, Cursor and git in [docs/hooks.md](docs/hooks.md)):
  ```json
  { "hooks": { "SessionStart": [{ "matcher": "startup|resume|clear|compact",
      "hooks": [{ "type": "command", "command": "omut hook start --agent claude" }] }] } }
  ```
- **The badge.** In a public GitHub repository `omut start` prints a badge for `README.md`: it links readers to `.omut/` and stays green while CI finds the meta fresh, not only well-formed ([workflow](docs/hooks.md#ci-and-the-badge)). This repository wears one.

## On a repository that is not yours

A work project, a team that doesn't use omut yet, someone else's open source: you can't commit `.omut/` there, but your agents still forget what they learned yesterday. Keep the meta for yourself:

```sh
cd their-repo && omut start --private
```

- `.omut/` and `CLAUDE.local.md` go into `.git/info/exclude`, which is never committed. The repository's own files, `AGENTS.md` and `.gitignore` included, don't change by a byte: `git status` stays empty.
- Claude Code finds the meta through `CLAUDE.local.md`. Codex has no local file that adds to `AGENTS.md`, so give it the [session-start hook](docs/hooks.md#codex) or one line in `~/.codex/AGENTS.md`: *If the repository has .omut/, keep its meta by .omut/README.md.*
- Everything else works as usual: `check`, `hook start`, the claims. The agent is told the meta is private and stays out of commits.

When the team sees what it gives, share it: remove the omut lines from `.git/info/exclude`, run `omut zero` (it writes the block into `AGENTS.md`), move what is true only for your copy into `.omut/local/`, and commit `.omut/`. Same format, nothing to migrate.

## What it lays down

```
my-project/
  AGENTS.md              # your rules + a short omut block (CLAUDE.md → AGENTS.md)
  .omut/
    README.md            # the rules, for agents and people (the tool's file)
    brief.md             # why · what it is · where it lives · now · next, at most 60 lines
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

## A pack of projects

Most people have more than one. A directory of projects can become a pack with a lead:

```sh
cd ~/projects && omut init      # the pack root: AGENTS.md, .omut/LEAD.md, pack meta
omut start hn-parser            # adopt a directory in the pack
omut check                      # across the pack: what is stale, unfilled, waiting for you
```

An agent started in the root is the lead: it knows the whole pack and tells you what is where. Started in a project, it keeps that project's meta.

| Command | What it does |
|---|---|
| `init [dir]` | create a pack |
| `start [project] [--private]` | a new project, or adopt an existing directory; `--private` keeps the meta out of git |
| `freeze` / `thaw [project]` | freeze / unfreeze |
| `kill <project> --yes --reason "..."` | remove, write a tombstone, list the mentions; refuses over unsaved work |
| `check [project] [--json] [--fail-on warn]` | the report; non-zero exit on shape errors |
| `zero [project]` | repair the scaffold, update the tool files |
| `hook start\|commit` | session context for agents, a git pre-commit reminder ([recipes](docs/hooks.md)) |

`check`, `zero`, `freeze` and `thaw` work on a single repository without a pack. The templates are plain markdown in [`templates/`](templates/), so all of it can be set up by hand too.

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

- [Specification v0](docs/spec-v0.md)
- [Hooks for git, CI, Claude Code, Codex, Cursor, and the badge](docs/hooks.md)
- [Competitors and neighbours](docs/research/competitors.md)
- [Checking the meta against reality: the variants weighed](docs/research/reality-check.md)

This repo keeps its own meta by its own spec: see [`.omut/`](.omut/).

## Credits

The idea grew out of using [drift](https://github.com/snowtema/drift).

## License

[MIT](LICENSE)
