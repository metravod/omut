package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/metravod/omut/internal/pack"
	"github.com/metravod/omut/templates"
)

// Help answers what someone about to run a command has to know without trying
// it in a scratch repository: what it writes, what it leaves alone, when it
// refuses. The flags are printed from their definitions, so they can't drift;
// help_test.go compares the rest with what the commands really do.

const usage = `omut: project memory that any agent can read, keep and check.

Plain markdown in .omut/ next to the code says why a project exists, what was
decided, what is unfinished and where it runs. Agents write it as they work;
this CLI lays down templates, checks their shape and freshness, and assembles
recorded context without interpreting it.

Start here:
  omut start             in your repository: .omut/ and a block in AGENTS.md
  omut start --private   in a repository that is not yours: git sees no change
Then tell your agent: "adopt this project following .omut/README.md".

Commands:
  start [project] [--private]    give a project its meta, or adopt a directory
  check [project] [--json] [--fail-on warn]
                                 report on the shape and freshness of the meta
  context [project] [--json]      assemble recorded state with source locations
  zero [project]                 repair the scaffold, update the tool files
  freeze [project]               status: frozen
  thaw [project]                 status: active
  init [dir]                     create a pack: a root for several projects
  kill <project> --yes --reason "..."
                                 remove a project of a pack, write a tombstone
  hook start|commit              for agents' and git's hooks, not called by hand
  version                        print the version

More:
  omut help <command>            what it writes, leaves alone and refuses, its
                                 flags and exit codes (or omut <command> --help)
  omut help meta                 the format and the rules of the meta, before
                                 anything is laid down

A single repository needs no pack: outside one, start, freeze, thaw, check,
context and zero work at any depth or from a linked worktree.

omut never commits or stages, changes git config or installs hooks, edits
agents' settings or uses the network. It runs git only to read, and writes
only inside the project or the pack it works on.
`

// helpTexts holds one text per command; printHelp adds the flags.
var helpTexts = map[string]string{
	"start": `omut start [project] [--private] [--by agent/model]

Gives a project its meta: a new directory, or an existing one as it is. Safe to
repeat: it only adds what is missing.

Which project:
  inside a pack   the project named, or the one you are in, at any depth. A
                  project is a directory right under the pack root; a missing
                  one is created.
  outside a pack  the directory given, as a single project; a missing one is
                  created, but not its parents. No omut init needed. Without a
                  name, the project you are in, at any depth; if there is none,
                  the current directory. omut start . gives a subdirectory its
                  own meta, for a monorepo.

What it writes in the project:
  .omut/README.md, .gitignore, .gitattributes
                        tool files, always at this version's text
  .omut/brief.md, decisions.md, log.md, threads/
                        empty templates; an existing file is never rewritten
  .omut/local/instance.md, local/log.md
                        this copy's memory; .omut/.gitignore keeps it out of git
  AGENTS.md             the omut block between its markers, appended; the rest
                        of the file stays as it is; created when missing
  CLAUDE.md             a symlink to AGENTS.md, only when there is no CLAUDE.md;
                        an existing one stays (check says if it doesn't lead to
                        AGENTS.md)

What it leaves alone: code and every other file, the git index and history, git
config and hooks, agents' settings, anything outside the project. Nothing is
committed: AGENTS.md and .omut/ are ordinary changes for you to review.

With --private, for a repository that is not yours:
  .git/info/exclude     gets /.omut/ and /CLAUDE.local.md; git never commits
                        this file; a clone's worktrees share it
  CLAUDE.local.md       takes the block instead of AGENTS.md (unless AGENTS.md
                        has it already), appended when the file exists
  AGENTS.md, CLAUDE.md, .gitignore and all tracked files don't change: git
  status stays empty.
  Claude Code reads CLAUDE.local.md. Codex has no such file: give it the
  session-start hook (omut help hook), review it with /hooks and start a new
  session; or, in the main checkout, this line in ~/.codex/AGENTS.md:
    If the repository has .omut/, keep its meta by .omut/README.md.
  To share the meta later: remove the omut lines from .git/info/exclude, move
  this copy's facts to .omut/local/, run omut zero, commit .omut/.

Refuses:
  a frozen project (omut thaw first); a path that is a file
  a path inside a pack that is not right under its root
  --private: outside git; on a directory that doesn't exist; where git tracks
  .omut/ (committed or staged)

--private over a .omut/ not yet in git keeps it as it is and makes it private.
What an earlier omut start did to AGENTS.md and CLAUDE.md stays, and omut says
what git still sees there.

In a linked git worktree: a clone keeps one memory, in its main checkout.
Private meta goes there, whichever copy you run this in, and every worktree
reads it. .omut/local/ stays there too. Only committed meta exists in each
worktree, and it follows the branch. A worktree beside the checkout doesn't read
CLAUDE.local.md: omut hook start is what gives an agent there the path.

What's next: the templates are empty, and check reports them as unfilled and
unverified until an agent fills them in. Tell your agent "adopt this project
following .omut/README.md"; omut help meta shows those rules. In a repository
whose origin is on GitHub, start also prints a badge line for README.md.

Exit code: 0 done or nothing to do; 1 refused, with the reason on stderr.
`,

	"check": `omut check [project] [--json] [--fail-on warn]

Reports on the shape and freshness of the meta. Changes nothing. Runs without a
network and reads git only.

What it checks:
  in a project of a pack   that project, at any depth
  at a pack's root         the whole pack
  a name, or a path        that project; .. from a project is the whole pack
  outside a pack           the project of the directory given or the current
                           one, at any depth or from a linked worktree; it needs
                           .omut/brief.md

Rules:
` + checkRulesText() + `
A few shape findings are warnings. A freshly laid template is not broken: it
gives unfilled and unverified, not errors.
A clean report checks shape and freshness, not the truth of the recorded claims.

--json prints an array, [] when all is good, of
  {"project", "rule", "level", "file", "line", "message"}
with file and line left out when there are none. Paths are relative to the pack
root, or to the parent of a single project.

Exit code: 0 no findings at the failing level; 1 errors, or warnings too with
--fail-on warn (CI and the badge use that), or no meta to check.
`,

	"context": `omut context [project] [--json]

Assembles recorded state without a model, network, cache or writes. Text is
copied from the meta with source paths and 1-based lines; only line endings are
normalized. No claim is verified and no text is shortened to fit a budget.

Scope (like check):
  at a pack root          overview of the pack and all its projects
  in a project            its current records, at any depth or in a worktree
  a project name or path  that project; .. from a project means the whole pack
  outside a pack          the project of the directory given or the current one

Pack overview:
  the pack's full brief and decisions
  project status, updated/verified, Why, What it is, Now and Next
  open/doubted decision lines and waiting ADR titles/statuses
  unfinished thread titles, Goal, Open and Next step, with status and owner
  check findings, including stale briefs and stalled threads
  frozen projects: brief sections only, with no pending work listed

Project records:
  full brief and decisions, active/parked threads and waiting ADR bodies
  filled local instance memory; the latest five entries from each log
  check findings; private and main-checkout/worktree locations when applicable

Pack overview omits project footprints, local memory, logs and decided project
records. Use context <project> for its current records. Completed/dropped
threads, decided ADR bodies and older history stay in their source files.
Invalid selected records and read failures are reported, not silently skipped.

--json emits the same selection as one object: scope, root, as_of, selection,
pack (when applicable), projects, findings, problems and incomplete. Each
project has metadata and sources with kind, file, line and literal text; thread
sources also carry status, updated, by and parked_because. Source and finding
paths are relative to root. Arrays are empty rather than null. Project and file
order is stable; freshness findings depend on the date and filesystem/git state.

Exit code: 0 assembled (warnings are included); 1 incomplete because of read or
shape errors, or no meta. Partial output remains available, including in JSON.
`,
	"zero": `omut zero [project] [--by agent/model]

Repairs the scaffold and brings the tool files to this version: this is how a
project or a pack is updated after a new release of omut.

Which part:
  in a project of a pack   that project, at any depth; it needs .omut/
  at a pack's root         the root only: each project takes its own run
  a name, or a path        that project, its .omut/ created when missing; ..
                           from a project is the root
  outside a pack           the project of the directory you are in, or the
                           directory given; it needs .omut/

What it does:
  rewrites    tool files: a project's .omut/README.md, .gitignore and
              .gitattributes; the root's .omut/LEAD.md
  rewrites    the omut block between its markers; the rest of the file stays
  creates     whatever is missing, from the templates, as start does
  removes     tool files of earlier versions (the root's conventions.md), only
              if they still carry the tool-file mark
  never       rewrites brief.md, decisions.md, log.md, threads/, local/; it
              lists their shape errors, which are the agent's to fix
  never       writes the line a human chose to add outside the block

In a linked worktree: committed meta is that branch's, and local/ is not created
there; where the worktree has no .omut/, zero works on the main checkout's.

In private meta (git ignores .omut/) the block stays in CLAUDE.local.md unless
AGENTS.md has it, and no CLAUDE.md is made.

Exit code: 0 even with shape errors listed (check fails on them); 1 refused.
`,

	"freeze": `omut freeze [project]

Sets status: frozen in .omut/brief.md. That one field, nothing else: not even
"updated".

The project is the one named or the one you are in, at any depth or from a
linked worktree; outside a pack, the directory given also works. It needs
.omut/brief.md.

Do it with the human's consent, after a final "Now" and a log line. If "Where it
lives" is not empty, freeze warns: make sure nothing still runs or spends money;
check goes on warning (frozen-but-live) until that section is emptied.

While frozen: start refuses; check and the pack context omit pending decisions,
and the startup hook omits due claims. omut thaw undoes it.

Exit code: 0 done or frozen already; 1 no brief, or a broken one.
`,

	"thaw": `omut thaw [project]

Sets status: active in .omut/brief.md. That one field, nothing else.

The project is the one named or the one you are in, at any depth or from a
linked worktree; outside a pack, the directory given also works. It needs
.omut/brief.md.

Exit code: 0 done or active already; 1 no brief, or a broken one.
`,

	"init": `omut init [dir] [--by agent/model]

Creates a pack: a root directory that keeps several projects and gives an agent
started there the overview of all of them. One repository doesn't need this:
omut start works without a pack.

What it writes in dir (the current directory by default, created when missing):
  .omut/LEAD.md         a tool file: what the agent started in the root does
  .omut/brief.md        empty template; its ignore: lists root directories that
                        are not projects
  .omut/decisions.md    empty template, for decisions of the whole pack
  .omut/graveyard.md    tombstones of projects removed with omut kill
  AGENTS.md             the omut block; the rest of the file stays as it is
  CLAUDE.md             a symlink to AGENTS.md, only when there is no CLAUDE.md

What it leaves alone: subdirectories. Adopt them one by one with omut start
<dir>; until then check reports each as no-meta, unless it is hidden or ignored.

Refuses a directory that is a pack already (omut zero updates one).

Exit code: 0 done; 1 refused.
`,

	"kill": `omut kill <project> --yes --reason "..." [--force]

Removes a project's whole directory and writes a tombstone. There is no undo.
Works only inside a pack; a single repository is yours to delete.

Do it only with the human's explicit consent.

What it does:
  removes     the project's directory (of a symlinked project, only the link)
  appends     to .omut/graveyard.md: name, date, the reason, the brief's "Why"
  prints      where other projects' meta still mentions it; fixing that is the
              agent's job, and check warns about it (ghost)

Refuses:
  without --yes or without --reason
  without --force, when something would be lost: uncommitted changes, gitignored
  files no build recreates (.env, local data, a filled .omut/local/; caches like
  node_modules/ don't count), commits on no remote, or no git at all

Exit code: 0 removed; 1 refused, with the reason on stderr.
`,

	"hook": `omut hook start|commit [--agent claude|codex|cursor] [--strict]

For agents' and git's hooks, not called by hand. Only reads. Says nothing and
exits 0 where there is no meta, so it is safe to install globally. omut installs
no hook itself: those files are yours. In a linked worktree it finds the clone's
meta in the main checkout and gives the agent the full path.

  start    at an agent's session start: prints the brief, this copy's memory,
           decisions waiting for the human, unfinished threads, what check
           found, claims due for checking, and work newer than the log that no
           commit holds. In a pack root: tells the agent it is the lead and
           includes the same assembled overview as omut context, with sources,
           findings and any incomplete-context notice.
  commit   as git's pre-commit: reminds on stderr when staged work is newer than
           .omut/log.md; blocks a commit carrying meta with shape errors.

Install, in .git/hooks/pre-commit (chmod +x it):
  #!/bin/sh
  exec omut hook commit

Install for Claude Code, in ~/.claude/settings.json:
  { "hooks": { "SessionStart": [{ "matcher": "startup|resume|clear|compact",
      "hooks": [{ "type": "command", "command": "omut hook start --agent claude" }] }] } }

Codex, Cursor, CI and the badge:
  https://github.com/metravod/omut/blob/main/docs/hooks.md

Exit code: 0, but 1 when commit blocks.
`,

	"version": `omut version

Prints the version of this binary. Also: omut --version.
`,
}

// checkRule is one line of the rules table in "omut help check". The test
// compares the codes with internal/check and the table in spec §8.
type checkRule struct{ code, level, when string }

var checkRules = []checkRule{
	{"no-meta", "error", `a directory of the pack without .omut/ (hidden and ignore: aside)`},
	{"shape", "error", `a missing field or section, a wrong status, a repeated Dn, a broken link`},
	{"scaffold", "error", `the pack root lacks a tool or meta file; zero fixes it`},
	{"local-in-git", "error", `git tracks files of .omut/local/`},
	{"size", "warn", `brief over 60 lines / 6000 Unicode characters, a thread over 80 / 8000; log over 50 entries`},
	{"stale-brief", "warn", `the project changed after the brief's "updated"`},
	{"unverified", "warn", `an active project's "verified" is empty or over 30 days old`},
	{"unseen", "warn", `in an active project, a claim with how: not seen for 30 days`},
	{"frozen-but-live", "warn", `frozen, but "Where it lives" is not empty`},
	{"leak", "warn", `a home path or an IP in the shared layer (not in private meta)`},
	{"local-not-ignored", "warn", `git doesn't ignore .omut/local/; zero fixes it`},
	{"ghost", "warn", `live meta mentions a project from the graveyard`},
	{"unfilled", "warn", `empty "Why", "What it is", "Now" or by; a thread with no "Next step"`},
	{"pointer", "warn", `no block in AGENTS.md, CLAUDE.md not leading to it, no .omut/README.md`},
	{"unchecked", "info", `claims without how:, counted per file`},
	{"stalled-thread", "info", `an active thread not updated for over 14 days`},
	{"open-decision", "info", `open and doubted decisions, waiting ADRs: a list for the human`},
	{"outdated", "info", `a tool file differs from this version's template; zero updates it`},
}

func checkRulesText() string {
	var b strings.Builder
	for _, r := range checkRules {
		left := fmt.Sprintf("  %-17s  %-5s  ", r.code, r.level)
		b.WriteString(left + wrap(r.when, len(left)) + "\n")
	}
	return b.String()
}

// helpWidth is the terminal width the help is written for.
const helpWidth = 80

// wrap breaks text to helpWidth; lines after the first are indented, since the
// first follows a label of that width.
func wrap(text string, indent int) string {
	var b strings.Builder
	col := indent
	for i, word := range strings.Fields(text) {
		switch {
		case i == 0:
		case col+1+len(word) > helpWidth:
			b.WriteString("\n" + strings.Repeat(" ", indent))
			col = indent
		default:
			b.WriteByte(' ')
			col++
		}
		b.WriteString(word)
		col += len(word)
	}
	return b.String()
}

// helpRequest is what parse returns on -h or --help: the flag set of the
// command asked about, read before the command does anything.
type helpRequest struct{ fs *flag.FlagSet }

func (helpRequest) Error() string { return "help requested" }

func cmdHelp(args []string, out io.Writer) error {
	if len(args) == 0 {
		fmt.Fprint(out, usage)
		return nil
	}
	switch topic := args[0]; topic {
	case "help", "-h", "--help":
		fmt.Fprint(out, usage)
		return nil
	case "meta":
		return printMeta(out)
	default:
		if _, ok := helpTexts[topic]; !ok {
			return fmt.Errorf("no help on %q: omut help lists the commands", topic)
		}
		return run([]string{topic, "--help"}, out)
	}
}

// printHelp prints a command's text and the flags it really defines.
func printHelp(out io.Writer, fs *flag.FlagSet) {
	fmt.Fprint(out, helpTexts[fs.Name()])
	type line struct{ left, usage string }
	var lines []line
	width := 0
	fs.VisitAll(func(f *flag.Flag) {
		name, text := flag.UnquoteUsage(f) // the name stays in the text
		left := "--" + f.Name
		if name != "" {
			left += " " + name
			text = strings.TrimPrefix(text, name+" ")
		}
		width = max(width, len(left))
		lines = append(lines, line{left, text})
	})
	if len(lines) == 0 {
		return
	}
	fmt.Fprintln(out, "\nFlags:")
	for _, l := range lines {
		left := fmt.Sprintf("  %-*s  ", width, l.left)
		fmt.Fprintln(out, left+wrap(l.usage, len(left)))
	}
}

// printMeta prints the rules a project carries in .omut/README.md, so they can
// be read before anything is laid down.
func printMeta(out io.Writer) error {
	b, err := templates.FS.ReadFile("project/omut/" + pack.Readme)
	if err != nil {
		return errors.New("the rules are missing from this binary: " + err.Error())
	}
	text := string(b)
	if strings.HasPrefix(text, "<!--") {
		if _, rest, ok := strings.Cut(text, "\n"); ok {
			text = rest
		}
	}
	fmt.Fprint(out, "What omut start lays down as .omut/README.md, the rules every agent keeps the meta by:\n\n", text)
	return nil
}
