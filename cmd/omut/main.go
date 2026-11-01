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

var version = "dev"

const usage = `omut: meta for a pack of projects that any agent can read and check.

Commands:
  init [dir]                           create a pack (in the current directory by default)
  start <project>                      a new project, or adopt an existing directory
  freeze <project>                     status: frozen
  thaw <project>                       status: active
  kill <project> --yes --reason "..."  remove a project, write a tombstone (--force: even with unsaved work)
  check [project] [--json]             check the shape and freshness of the meta
  zero [project]                       repair the scaffold, update the tool files
  version

Flags of init/start/zero: --by agent/model (or $OMUT_BY), who fills in the templates.
`

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
	cmd, args := args[0], args[1:]
	switch cmd {
	case "init":
		return cmdInit(args, out)
	case "start":
		return cmdStart(args, out)
	case "freeze":
		return cmdStatus(args, out, "frozen")
	case "thaw":
		return cmdStatus(args, out, "active")
	case "kill":
		return cmdKill(args, out)
	case "check":
		return cmdCheck(args, out)
	case "zero":
		return cmdZero(args, out)
	case "version", "--version":
		fmt.Fprintln(out, version)
		return nil
	case "help", "-h", "--help":
		fmt.Fprint(out, usage)
		return nil
	}
	return fmt.Errorf("no command %q\n\n%s", cmd, usage)
}

// parse lets flags and positional arguments come in any order, so an agent
// can write both "kill x --yes" and "kill --yes x".
func parse(fs *flag.FlagSet, args []string) ([]string, error) {
	fs.SetOutput(io.Discard)
	var pos []string
	for {
		if err := fs.Parse(args); err != nil {
			return nil, err
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
	return fs.String("by", os.Getenv("OMUT_BY"), "agent/model for the by field")
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

func cmdStart(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("start", flag.ContinueOnError)
	by := byFlag(fs)
	pos, err := parse(fs, args)
	if err != nil {
		return err
	}
	if len(pos) != 1 {
		return errors.New("need a project name: omut start <project>")
	}
	p, err := findPack()
	if err != nil {
		return err
	}
	name, err := p.Resolve(pos[0])
	if err != nil {
		return err
	}
	res, err := scaffold.Start(p, name, scaffold.Options{By: *by})
	if err != nil {
		return err
	}
	if len(res.Created)+len(res.Updated) == 0 {
		fmt.Fprintf(out, "%s is in the pack already, the scaffold is fine.\n", name)
		return nil
	}
	printResult(out, res)
	fmt.Fprintf(out, "\nFill in %s/.omut/brief.md following \"Adopting a project\" in .omut/conventions.md.\n", name)
	return nil
}

func cmdStatus(args []string, out io.Writer, status string) error {
	fs := flag.NewFlagSet(status, flag.ContinueOnError)
	pos, err := parse(fs, args)
	if err != nil {
		return err
	}
	if len(pos) != 1 {
		return errors.New("need a project name")
	}
	p, err := findPack()
	if err != nil {
		return err
	}
	name, err := p.Resolve(pos[0])
	if err != nil {
		return err
	}
	path := filepath.Join(p.Dir(name), pack.MetaDir, pack.Brief)
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
	reason := fs.String("reason", "", "the reason, for the tombstone")
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
	pos, err := parse(fs, args)
	if err != nil {
		return err
	}
	var findings []check.Finding
	p, err := findPack()
	switch {
	case errors.Is(err, pack.ErrNoPack):
		// A project outside any pack, such as a colleague's clone, still
		// carries its own meta.
		dir := "."
		if len(pos) > 0 {
			dir = pos[0]
		}
		abs, _ := filepath.Abs(dir)
		if _, serr := os.Stat(filepath.Join(abs, pack.MetaDir, pack.Brief)); serr != nil {
			return err
		}
		findings = check.Standalone(abs, check.Options{})
	case err != nil:
		return err
	case len(pos) > 0:
		name, err := p.Resolve(pos[0])
		if err != nil {
			return err
		}
		if _, err := os.Stat(p.Dir(name)); err != nil {
			return fmt.Errorf("no project %s", name)
		}
		findings = check.Project(p, name, check.Options{})
	default:
		findings, err = check.Pack(p, check.Options{})
		if err != nil {
			return err
		}
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
	if check.HasErrors(findings) {
		return exitCode(1)
	}
	return nil
}

func cmdZero(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("zero", flag.ContinueOnError)
	by := byFlag(fs)
	pos, err := parse(fs, args)
	if err != nil {
		return err
	}
	p, err := findPack()
	if err != nil {
		return err
	}
	opt := scaffold.Options{By: *by}
	var res *scaffold.Result
	var findings []check.Finding
	if len(pos) > 0 {
		name, err := p.Resolve(pos[0])
		if err != nil {
			return err
		}
		if res, err = scaffold.ZeroProject(p, name, opt); err != nil {
			return err
		}
		findings = check.Project(p, name, check.Options{})
	} else {
		if res, err = scaffold.ZeroRoot(p, opt); err != nil {
			return err
		}
		findings, _ = check.Pack(p, check.Options{})
	}
	if len(res.Created)+len(res.Updated) == 0 {
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
