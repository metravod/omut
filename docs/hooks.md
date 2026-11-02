# Hooks

The rituals in `AGENTS.md` rest on the agent's discipline, and the end of a session is the weakest one: no agent gets a reliable "the session is over" signal. Hooks move what can be moved out of discipline and into the tools. `omut hook` is the one command they all call; the recipes below are thin adapters around it.

`omut hook` only reads. In a directory with no `.omut/` above it, it says nothing and exits 0, so it is safe to install globally.

## What there is

| Hook | When | What it does |
|---|---|---|
| `omut hook start` | an agent's session starts | Prints the context to start from: the brief, decisions waiting for the human, unfinished threads with their next step, and what `omut check` found. In a pack root, it tells the agent it is the lead and gives the pack's check. |
| `omut hook commit` | git pre-commit | Reminds when staged work is newer than the last write to `.omut/log.md`. Blocks a commit that carries meta with shape errors. `--strict` also blocks work committed without a log entry. |

Why the end of a session is a commit and not the agent's "stop": agents fire their stop hook at the end of every turn, not of the session, so a reminder there would nag after every answer. A commit is the moment a piece of work is declared done, it is the same for every agent and for people, and its output is something agents read. Work left uncommitted is caught at the next start: `stale-brief` in the start context says the previous session may not have recorded what it did.

`--agent claude|codex|cursor` makes `omut hook start` read the agent's JSON payload for the session's directory and answer in that agent's format. Without it, the current directory and plain text.

## git

Any repository, any agent, any human. In `.git/hooks/pre-commit` (`chmod +x` it):

```sh
#!/bin/sh
exec omut hook commit
```

With a hook manager, call `omut hook commit` from it. Add `--strict` where agents do most of the commits; `git commit --no-verify` is the way around it.

## CI

`omut check` exits non-zero on shape errors. A GitHub Actions job, `.github/workflows/omut.yml`:

```yaml
name: omut
on: [push, pull_request]
jobs:
  check:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: stable
      - run: go install github.com/metravod/omut/cmd/omut@latest
      - run: omut check
```

## Claude Code

`SessionStart` adds the hook's stdout to the context. In `~/.claude/settings.json` (every project) or `.claude/settings.json` (one project):

```json
{
  "hooks": {
    "SessionStart": [
      {
        "matcher": "startup|resume|clear|compact",
        "hooks": [{ "type": "command", "command": "omut hook start --agent claude" }]
      }
    ]
  }
}
```

## Codex

`SessionStart` adds plain stdout as developer context. In `~/.codex/hooks.json` or `<repo>/.codex/hooks.json`; review it once with `/hooks`, and a project's `.codex/` has to be trusted:

```json
{
  "hooks": {
    "SessionStart": [
      {
        "matcher": "startup|resume|clear|compact",
        "hooks": [{ "type": "command", "command": "omut hook start --agent codex" }]
      }
    ]
  }
}
```

## Cursor

`sessionStart` takes `additional_context` from the hook's JSON. In `~/.cursor/hooks.json` or `<project>/.cursor/hooks.json`:

```json
{
  "version": 1,
  "hooks": {
    "sessionStart": [{ "command": "omut hook start --agent cursor" }]
  }
}
```

## Other agents

An agent without hooks still gets the omut block in `AGENTS.md`, and the git hook works for it as for anyone. A hook that runs a command at session start and passes plain stdout to the agent can use `omut hook start` as is.

## How far this is verified

The output formats follow each vendor's hook documentation as read on 2026-11-02; `omut hook` itself is covered by tests. None of the agent recipes has been run in a live session yet; the git hook has been run by hand on this repository.
