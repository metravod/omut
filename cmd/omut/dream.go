package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"

	"github.com/metravod/omut/internal/check"
	"github.com/metravod/omut/internal/dream"
	"github.com/metravod/omut/internal/pack"
)

func cmdDream(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("dream", flag.ContinueOnError)
	apply := fs.Bool("apply", false, "the contract says the human has approved: the dream agent applies the changes and sets dreamed; the CLI itself writes nothing")
	asJSON := fs.Bool("json", false, "output the same digest and contract as JSON")
	pos, err := parse(fs, args)
	if err != nil {
		return err
	}
	if len(pos) > 1 {
		return errors.New("omut dream [project] [--apply] [--json]")
	}
	var report *dream.Report
	p, err := findPack()
	switch {
	case errors.Is(err, pack.ErrNoPack):
		dir := "."
		if len(pos) == 1 {
			dir = pos[0]
		}
		pl := pack.Locate(dir)
		if pl.Dir == "" || pl.Foreign {
			return errNoMeta
		}
		if pack.IsRoot(pl.Dir) {
			report = dream.ForPack(&pack.Pack{Root: pl.Dir}, check.Options{}, *apply)
		} else {
			report = dream.ForProject(pl, check.Options{}, *apply)
		}
	case err != nil:
		return err
	default:
		name, err := packScope(p, pos)
		if err != nil {
			return err
		}
		if name == "" {
			report = dream.ForPack(p, check.Options{}, *apply)
			break
		}
		pl := pack.Locate(p.Dir(name))
		if pl.Dir == "" || pl.Foreign || pack.IsRoot(pl.Dir) {
			return fmt.Errorf("%s has no project meta: omut start %s first", name, name)
		}
		report = dream.ForProject(pl, check.Options{}, *apply)
	}
	if *asJSON {
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		err = enc.Encode(report)
	} else {
		_, err = io.WriteString(out, report.Markdown())
	}
	if err != nil {
		return err
	}
	if report.Incomplete {
		return exitCode(1)
	}
	return nil
}
