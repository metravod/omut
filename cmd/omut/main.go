// Command omut lays down and checks the meta of a pack of projects.
// It never writes meaning into the meta: agents do that.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/metravod/omut/internal/check"
	"github.com/metravod/omut/internal/meta"
	"github.com/metravod/omut/internal/pack"
	"github.com/metravod/omut/internal/scaffold"
)

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		var exit exitCode
		if errors.As(err, &exit) {
			os.Exit(int(exit))
		}
		fmt.Fprintln(os.Stderr, "omut:", err)
		os.Exit(1)
	}
}

// exitCode ends the program with a code and no message: the report has
// already been printed.
type exitCode int

func (e exitCode) Error() string { return fmt.Sprintf("exit %d", int(e)) }

func run(args []string, out io.Writer) error {
	if len(args) == 0 {
		fmt.Fprint(out, usage)
		return nil
	}
	err := dispatch(args[0], args[1:], out)
	var help helpRequest
	if errors.As(err, &help) {
		printHelp(out, help.fs)
		return nil
	}
	return err
}

func dispatch(cmd string, args []string, out io.Writer) error {
	switch cmd {
	case "init":
		return cmdInit(args, out)
	case "start":
		return cmdStart(args, out)
	case "freeze":
		return cmdStatus("freeze", "frozen", args, out)
	case "thaw":
		return cmdStatus("thaw", "active", args, out)
	case "kill":
		return cmdKill(args, out)
	case "check":
		return cmdCheck(args, out)
	case "context":
		return cmdContext(args, out)
	case "zero":
		return cmdZero(args, out)
	case "hook":
		return cmdHook(args, out, os.Stdin, os.Stderr)
	case "version", "--version":
		if _, err := parse(flag.NewFlagSet("version", flag.ContinueOnError), args); err != nil {
			return err
		}
		fmt.Fprintln(out, version)
		return nil
	case "help", "-h", "--help":
		return cmdHelp(args, out)
	}
	return fmt.Errorf("no command %q\n\n%s", cmd, usage)
}

// parse lets flags and positional arguments come in any order, so an agent
// can write both "kill x --yes" and "kill --yes x". Every command calls it
// before doing anything, so --help is safe anywhere on the line.
func parse(fs *flag.FlagSet, args []string) ([]string, error) {
	fs.SetOutput(io.Discard)
	var pos []string
	for {
		if err := fs.Parse(args); errors.Is(err, flag.ErrHelp) {
			return nil, helpRequest{fs}
		} else if err != nil {
			return nil, fmt.Errorf("%w (omut help %s)", err, fs.Name())
		}
		args = fs.Args()
		if len(args) == 0 {
			return pos, nil
		}
		pos = append(pos, args[0])
		args = args[1:]
	}
}

func byFlag(fs *flag.FlagSet) *string {
	return fs.String("by", os.Getenv("OMUT_BY"), "`agent/model` who fills in the templates, such as claude-code/opus-5.5, for the by field of new files (default: $OMUT_BY)")
}

func cmdInit(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	by := byFlag(fs)
	pos, err := parse(fs, args)
	if err != nil {
		return err
	}
	dir := "."
	if len(pos) > 0 {
		dir = pos[0]
	}
	res, err := scaffold.Init(dir, scaffold.Options{By: *by})
	if err != nil {
		return err
	}
	printResult(out, res)
	fmt.Fprintln(out, "\nThe pack is ready. Start an agent in this directory and ask for an overview, or adopt projects: omut start <dir>.")
	return nil
}

func findPack() (*pack.Pack, error) { return pack.Find(".") }

// target finds the project a command works on. Inside a pack it is a
// project of the pack, named or given as a path ("." by default); outside
// any pack it is the directory itself, and p is nil.
func target(pos []string) (p *pack.Pack, dir, name string, err error) {
	if len(pos) > 1 {
		return nil, "", "", errors.New("one project at a time")
	}
	arg := "."
	if len(pos) == 1 {
		arg = pos[0]
	}
	p, err = findPack()
	if errors.Is(err, pack.ErrNoPack) {
		if len(pos) == 0 {
			if pl := pack.Locate("."); pl.Dir != "" && !pl.Foreign {
				return nil, pl.Dir, filepath.Base(pl.Dir), nil
			}
		}
		if dir, err = filepath.Abs(arg); err != nil {
			return nil, "", "", err
		}
		return nil, dir, filepath.Base(dir), nil
	}
	if err != nil {
		return nil, "", "", err
	}
	if current, ok := p.Current(); ok && len(pos) == 0 {
		return p, p.Dir(current), current, nil
	}
	if name, err = p.Resolve(arg); err != nil {
		return nil, "", "", err
	}
	return p, p.Dir(name), name, nil
}

// packScope finds what check and zero work on inside a pack: a project, or
// the pack itself when name is "". Without an argument it is the project the
// working directory is in, and the pack at the root; from a project, ".."
// names the pack.
func packScope(p *pack.Pack, pos []string) (name string, err error) {
	switch {
	case len(pos) > 1:
		return "", errors.New("one project at a time")
	case len(pos) == 0:
		name, _ = p.Current()
		return name, nil
	}
	if abs, err := filepath.Abs(pos[0]); err == nil && abs == p.Root {
		return "", nil
	}
	return p.Resolve(pos[0])
}

func cmdStart(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("start", flag.ContinueOnError)
	by := byFlag(fs)
	private := fs.Bool("private", false, "keep the whole meta out of git, through .git/info/exclude (a repository that is not yours)")
	pos, err := parse(fs, args)
	if err != nil {
		return err
	}
	p, dir, name, err := target(pos)
	if err != nil {
		return err
	}
	// Private meta is the clone's, so it goes to the main checkout: laid down
	// in a linked worktree, it would go when the worktree goes (D17).
	if main := pack.MainPath(dir); *private && len(pos) == 0 && main != "" {
		dir, name = main, filepath.Base(main)
	}
	if cwd, err := os.Getwd(); err == nil && p == nil && len(pos) == 0 && !sameDir(dir, cwd) {
		fmt.Fprintf(out, "The meta of this directory is in %s. To give this directory its own: omut start .\n", dir)
	}
	res, err := scaffold.StartAt(dir, scaffold.Options{By: *by, Private: *private})
	if err != nil {
		return err
	}
	if len(res.Created)+len(res.Updated) == 0 {
		if p == nil {
			fmt.Fprintf(out, "%s has its meta already, the scaffold is fine.\n", name)
		} else {
			fmt.Fprintf(out, "%s is in the pack already, the scaffold is fine.\n", name)
		}
		return nil
	}
	if p == nil {
		fmt.Fprintf(out, "No pack around %s: it gets its meta as a single project.\n", dir)
	}
	printResult(out, res)
	fmt.Fprintf(out, "\nFill in %s/.omut/brief.md following \"Adopting a project\" in %s/.omut/README.md.\n", name, name)
	if pack.Private(dir) {
		fmt.Fprint(out, privateNote)
		return nil
	}
	if b := badge(dir); b != "" {
		fmt.Fprintf(out, "\nThe repository is on GitHub. If it is public, offer the human the omut badge for README.md: it links readers to .omut/ and turns green while CI finds the meta fresh (workflow: %s).\n  %s\n", badgeRecipe, b)
	}
	return nil
}

func sameDir(a, b string) bool {
	ai, err1 := os.Stat(a)
	bi, err2 := os.Stat(b)
	return err1 == nil && err2 == nil && os.SameFile(ai, bi)
}

const privateNote = `
The meta is private: git ignores .omut/ and CLAUDE.local.md through .git/info/exclude, and this command leaves the repository's own files alone. Claude Code finds the meta through CLAUDE.local.md. Codex has no local file that adds to AGENTS.md: give it the session-start hook (https://github.com/metravod/omut/blob/main/docs/hooks.md#codex) or this line in ~/.codex/AGENTS.md:
  If the repository has .omut/, keep its meta by .omut/README.md.
To share the meta later: remove the omut lines from .git/info/exclude, run omut zero, commit .omut/.
`

func cmdStatus(cmd, status string, args []string, out io.Writer) error {
	fs := flag.NewFlagSet(cmd, flag.ContinueOnError)
	pos, err := parse(fs, args)
	if err != nil {
		return err
	}
	_, dir, name, err := target(pos)
	if err != nil {
		return err
	}
	path := filepath.Join(dir, pack.MetaDir, pack.Brief)
	d, err := meta.Read(path)
	if err != nil {
		return fmt.Errorf("%s has no .omut/brief.md: omut start %s first", name, name)
	}
	if d.Front["status"] == status {
		fmt.Fprintf(out, "%s is %s already.\n", name, status)
		return nil
	}
	if err := meta.SetFront(path, "status", status); err != nil {
		return err
	}
	fmt.Fprintf(out, "%s: status: %s\n", name, status)
	if status == "frozen" {
		if s, ok := d.Section("Where it lives"); ok && d.Filled(s) {
			fmt.Fprintln(out, "Note: \"Where it lives\" is not empty. Make sure nothing is still running or spending money.")
		}
	}
	return nil
}

func cmdKill(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("kill", flag.ContinueOnError)
	yes := fs.Bool("yes", false, "confirm the removal")
	force := fs.Bool("force", false, "remove even with unsaved work")
	reason := fs.String("reason", "", "`text` the reason, for the tombstone")
	pos, err := parse(fs, args)
	if err != nil {
		return err
	}
	if len(pos) != 1 {
		return errors.New("need a project name: omut kill <project> --yes --reason \"...\"")
	}
	p, err := findPack()
	if err != nil {
		return err
	}
	name, err := p.Resolve(pos[0])
	if err != nil {
		return err
	}
	dir := p.Dir(name)
	if _, err := os.Lstat(dir); err != nil {
		return fmt.Errorf("no project %s", name)
	}
	if !*yes {
		return fmt.Errorf("kill removes the whole directory %s: repeat with --yes", dir)
	}
	if strings.TrimSpace(*reason) == "" {
		return errors.New("the tombstone needs a reason: --reason \"...\"")
	}
	if info, err := os.Lstat(dir); err == nil && info.Mode()&os.ModeSymlink == 0 {
		if unsaved := scaffold.Unsaved(dir); len(unsaved) > 0 && !*force {
			return fmt.Errorf("%s has unsaved work:\n  %s\nSave it or repeat with --force", name, strings.Join(unsaved, "\n  "))
		}
	}
	tomb, mentions, err := scaffold.Kill(p, name, *reason, scaffold.Options{})
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "Removed %s. Tombstone in .omut/graveyard.md:\n%s\n", name, tomb)
	if len(mentions) > 0 {
		fmt.Fprintln(out, "\nMentions in the meta worth fixing:")
		for _, m := range mentions {
			fmt.Fprintf(out, "  %s:%d  %s\n", m.File, m.Line, m.Text)
		}
	}
	return nil
}

func cmdCheck(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("check", flag.ContinueOnError)
	asJSON := fs.Bool("json", false, "output JSON")
	failOn := fs.String("fail-on", "error", "`level` the lowest level that makes the exit code non-zero: error (default) or warn")
	pos, err := parse(fs, args)
	if err != nil {
		return err
	}
	if *failOn != "error" && *failOn != "warn" {
		return fmt.Errorf("--fail-on %q: error or warn", *failOn)
	}
	var findings []check.Finding
	p, err := findPack()
	switch {
	case errors.Is(err, pack.ErrNoPack):
		// A project outside any pack, such as a colleague's clone, still
		// carries its own meta. From inside it or a worktree of it, that too.
		dir := "."
		if len(pos) > 0 {
			dir = pos[0]
		}
		pl := pack.Locate(dir)
		if pl.Dir == "" || pl.Foreign {
			return errNoMeta
		}
		if _, serr := os.Stat(filepath.Join(pl.Dir, pack.MetaDir, pack.Brief)); serr != nil {
			return errNoMeta
		}
		findings = check.Dir(pl.Dir, check.Options{})
	case err != nil:
		return err
	default:
		name, err := packScope(p, pos)
		if err != nil {
			return err
		}
		if name == "" {
			if findings, err = check.Pack(p, check.Options{}); err != nil {
				return err
			}
			break
		}
		if _, err := os.Stat(p.Dir(name)); err != nil {
			return fmt.Errorf("no project %s", name)
		}
		findings = check.Project(p, name, check.Options{})
	}
	if *asJSON {
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		if findings == nil {
			findings = []check.Finding{}
		}
		if err := enc.Encode(findings); err != nil {
			return err
		}
	} else {
		printFindings(out, findings)
	}
	if check.HasErrors(findings) || (*failOn == "warn" && check.HasWarnings(findings)) {
		return exitCode(1)
	}
	return nil
}

var errNoMeta = errors.New("no .omut/ here or above: omut start gives this directory its meta (omut init is only for a pack of several projects)")

func cmdZero(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("zero", flag.ContinueOnError)
	by := byFlag(fs)
	pos, err := parse(fs, args)
	if err != nil {
		return err
	}
	opt := scaffold.Options{By: *by}
	var res *scaffold.Result
	var findings []check.Finding
	p, err := findPack()
	switch {
	case errors.Is(err, pack.ErrNoPack):
		_, dir, _, err := target(pos)
		if err != nil {
			return err
		}
		if _, err := os.Stat(filepath.Join(dir, pack.MetaDir)); err != nil {
			return fmt.Errorf("%s has no .omut/ and there is no pack around: omut start creates the meta", dir)
		}
		if res, err = scaffold.ZeroAt(dir, opt); err != nil {
			return err
		}
		findings = check.Standalone(dir, check.Options{})
	case err != nil:
		return err
	default:
		name, err := packScope(p, pos)
		if err != nil {
			return err
		}
		if name == "" {
			if res, err = scaffold.ZeroRoot(p, opt); err != nil {
				return err
			}
			findings, _ = check.Pack(p, check.Options{})
			break
		}
		// A directory you merely stand in doesn't get a scaffold by accident:
		// naming a project is how to ask for one.
		if _, err := os.Stat(filepath.Join(p.Dir(name), pack.MetaDir)); err != nil && len(pos) == 0 {
			return fmt.Errorf("%s has no .omut/: omut start creates the meta, omut zero .. repairs the pack", name)
		}
		if res, err = scaffold.ZeroProject(p, name, opt); err != nil {
			return err
		}
		findings = check.Project(p, name, check.Options{})
	}
	if len(res.Created)+len(res.Updated)+len(res.Removed) == 0 {
		fmt.Fprintln(out, "The scaffold is fine.")
	} else {
		printResult(out, res)
	}
	var shape []check.Finding
	for _, f := range findings {
		if f.Rule == "shape" {
			shape = append(shape, f)
		}
	}
	if len(shape) > 0 {
		fmt.Fprintln(out, "\nShape errors in the meta (zero does not fix them, that is the agent's job):")
		printFindings(out, shape)
	}
	return nil
}

func printResult(out io.Writer, res *scaffold.Result) {
	for _, f := range res.Created {
		fmt.Fprintln(out, "  created ", f)
	}
	for _, f := range res.Updated {
		fmt.Fprintln(out, "  updated ", f)
	}
	for _, f := range res.Removed {
		fmt.Fprintln(out, "  removed ", f)
	}
	for _, n := range res.Notes {
		fmt.Fprintln(out, "  ", n)
	}
}

func printFindings(out io.Writer, findings []check.Finding) {
	if len(findings) == 0 {
		fmt.Fprintln(out, "All good.")
		return
	}
	project := "\x00"
	counts := map[check.Level]int{}
	for _, f := range findings {
		if f.Project != project {
			project = f.Project
			title := f.Project
			if title == "" {
				title = "pack"
			}
			fmt.Fprintf(out, "\n%s\n", title)
		}
		counts[f.Level]++
		loc := f.File
		if f.Line > 0 {
			loc = fmt.Sprintf("%s:%d", f.File, f.Line)
		}
		fmt.Fprintf(out, "  %-5s %-15s %s  %s\n", f.Level, f.Rule, f.Message, loc)
	}
	fmt.Fprintf(out, "\n%d errors, %d warnings, %d info\n", counts[check.Error], counts[check.Warn], counts[check.Info])
}
