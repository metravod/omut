# Hooks

The rituals in `AGENTS.md` rest on the agent's discipline, and the end of a session is the weakest one: no agent gets a reliable "the session is over" signal. Hooks move what can be moved out of discipline and into the tools. `omut hook` is the one command they all call; the recipes below are thin adapters around it.

`omut hook` only reads. In a directory with no `.omut/` above it, it says nothing and exits 0, so it is safe to install globally.

## What there is

| Hook | When | What it does |
|---|---|---|
| `omut hook start` | an agent's session starts | Prints the context to start from: the brief, decisions waiting for the human, unfinished threads with their next step, and what `omut check` found. Warns about unrecorded work: files newer than the last write to `.omut/log.md` that no commit holds. In a pack root, it tells the agent it is the lead and gives the pack's check. |
| `omut hook commit` | git pre-commit | Reminds when staged work is newer than the last write to `.omut/log.md`. Blocks a commit that carries meta with shape errors. `--strict` also blocks work committed without a log entry. |

`--agent claude|codex|cursor` makes `omut hook start` read the agent's JSON payload for the session's directory and answer in that agent's format. Without it, the current directory and plain text.

## The end of a piece of work

No agent tells a hook "the session is over". Claude Code, Codex and Cursor fire their stop hook after every turn, so a reminder there would nag after every answer; `SessionEnd` fires when the agent can no longer act. So omut leans on three things, in this order:

1. **Prose.** The rules tie the ritual to a moment the agent does see: before it tells the human a piece of work is done. The omut block in `AGENTS.md`, `.omut/README.md` and the start context all say so.
2. **The commit.** It is when a piece of work is declared done, it is the same for every agent and for people, and agents read its output: `omut hook commit`.
3. **The next start.** Work that neither was recorded nor went into a commit shows up in the next session's context as unrecorded work, with the file and the time, and the agent is asked to record it before going on.

**Without git** there is no commit, so only 1 and 3 are left: a piece of work that ended unrecorded is caught at the start of the next session, not when it ends. If that is not enough, `git init` is enough: a local repository with no remote gives the commit hook its signal (and `omut kill` its copy).

## git

Any repository, any agent, any human. In `.git/hooks/pre-commit` (`chmod +x` it):

```sh
#!/bin/sh
exec omut hook commit
```

With a hook manager, call `omut hook commit` from it. Add `--strict` where agents do most of the commits; `git commit --no-verify` is the way around it.

## CI and the badge

`omut check` exits non-zero on shape errors; with `--fail-on warn` on warnings too: a stale brief, a brief not checked against reality for 30 days, an unfilled template, a missing pointer. That is the difference between "the meta is well-formed" and "the meta is fresh", and it is what the badge shows. A GitHub Actions workflow, `.github/workflows/omut.yml`:

```yaml
name: omut
on:
  push:
    branches: [main]
  pull_request:
  schedule:
    - cron: "0 6 * * 1"   # what goes stale with time alone, such as `verified`
jobs:
  check:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: stable
      - run: go install github.com/metravod/omut/cmd/omut@latest
      - run: omut check --fail-on warn
```

`--fail-on` needs omut 0.2.0 or later. Drop it to fail on shape errors only.

The badge, in the project's `README.md`, links to `.omut/`, where GitHub shows `.omut/README.md`: what the meta is and how to read it. `omut start` prints it with the owner, repository and branch filled in when it adopts the root of a GitHub repository; otherwise put yours in the URL.

```markdown
[![omut](https://github.com/OWNER/REPO/actions/workflows/omut.yml/badge.svg?branch=main)](.omut/)
```

Without CI, a static badge says only that the meta is there:

```markdown
[![omut](https://img.shields.io/badge/meta-omut-blue)](.omut/)
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

The output formats follow each vendor's hook documentation as read on 2026-11-02; `omut hook` itself is covered by tests, with and without git. The Claude Code recipe runs live (2026-11-03: a headless `claude -p` session in a pack project quoted the omut context it got at start). The Codex and Cursor recipes have not been run in a live session yet; the git hook has been run by hand on this repository.
