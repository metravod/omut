# omut

A pack of projects that any agent can tell the truth about. Short meta in every project, rules in the root, and a check that what is written still matches reality.

> *Omut pamyati* is the Russian name of Dumbledore's Pensieve: you dip into someone's memories and see how it really was.

Status: v0, the first working CLI.

## Why

Agents build the projects, and you stop remembering what is where and why. Something runs on a server for half a year though it was meant for a week. A project moves from Claude Code to Codex, or to a colleague, and the new agent digs through the code to find out what was decided and what comes next.

omut keeps a few plain markdown files next to each project: a brief (why, what it is, where it lives, now, next), a decision registry, threads of work and a log. Any agent that reads `AGENTS.md` can keep them. The CLI only lays down templates and checks the shape and freshness of what agents wrote.

## How it looks

```sh
cd ~/projects && omut init      # the pack root: AGENTS.md, .omut/LEAD.md, .omut/conventions.md
omut start hn-parser            # adopt a directory: .omut/ templates, a block in AGENTS.md
omut check                      # what is stale, what is unfilled, which decisions wait
```

Agents do the rest: Claude Code, Codex, or anything else that reads `AGENTS.md`. Started in the root, an agent is the lead: it knows the whole pack and tells you what is where. Started in a project, it keeps that project's meta.

```
~/projects/
  AGENTS.md              # your rules + the omut block
  .omut/                 # the lead's instructions, pack focus, decisions, graveyard
  hn-parser/
    AGENTS.md
    .omut/
      brief.md           # why · what it is · where it lives · now · next
      decisions.md       # - D3 [open] Keep collecting after the week? → threads/hn-future.md
      threads/           # unfinished work, with the next step
      log.md             # - 2026-10-31 · codex/gpt-5 · what was done → link
```

| Command | What it does |
|---|---|
| `init [dir]` | create a pack |
| `start <project>` | a new project, or adopt an existing directory |
| `freeze` / `thaw <project>` | freeze / unfreeze |
| `kill <project> --yes --reason "..."` | remove, write a tombstone, list the mentions |
| `check [project] [--json]` | the report; non-zero exit on shape errors |
| `zero [project]` | repair the scaffold, update the tool files |

The templates are plain markdown in [`templates/`](templates/), so a pack can be set up by hand too.

## Install

```sh
brew install metravod/tap/omut
```

Or with Go:

```sh
go install github.com/metravod/omut/cmd/omut@latest
```

Binaries for macOS, Linux and Windows are on the [releases page](https://github.com/metravod/omut/releases).

Or build from source:

```sh
go build ./cmd/omut
go test ./...
```

## Docs

- [Specification v0](docs/spec-v0.md)
- [Competitors and neighbours](docs/research/competitors.md)

This repo keeps its own meta by its own spec: see [`.omut/`](.omut/).

## Credits

The idea grew out of using [drift](https://github.com/snowtema/drift).

## License

[MIT](LICENSE)
