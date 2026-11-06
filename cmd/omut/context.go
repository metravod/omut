package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/metravod/omut/internal/check"
	"github.com/metravod/omut/internal/context"
	"github.com/metravod/omut/internal/pack"
)

func cmdContext(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("context", flag.ContinueOnError)
	asJSON := fs.Bool("json", false, "output the same selection as structured JSON, with source paths and lines")
	pos, err := parse(fs, args)
	if err != nil {
		return err
	}
	if len(pos) > 1 {
		return errors.New("omut context [project] [--json]")
	}
	var report *context.Report
	p, err := findPack()
	switch {
	case errors.Is(err, pack.ErrNoPack):
		dir := "."
		if len(pos) == 1 {
			dir = pos[0]
		}
		info, err := os.Stat(dir)
		if err != nil {
			return err
		}
		if !info.IsDir() {
			return fmt.Errorf("%s is not a directory", dir)
		}
		pl := pack.Locate(dir)
		if pl.Dir == "" || pl.Foreign {
			return errNoMeta
		}
		if pack.IsRoot(pl.Dir) {
			report = context.ForPack(&pack.Pack{Root: pl.Dir}, check.Options{})
		} else {
			report = context.ForProject(pl, check.Options{})
		}
	case err != nil:
		return err
	default:
		name, err := packScope(p, pos)
		if err != nil {
			return err
		}
		if name == "" {
			report = context.ForPack(p, check.Options{})
		} else {
			dir := p.Dir(name)
			pl := pack.Locate(dir)
			if pl.Dir == "" || pl.Foreign || pack.IsRoot(pl.Dir) {
				return fmt.Errorf("%s has no project meta: omut start %s first", name, name)
			}
			report = context.ForProject(pl, check.Options{})
		}
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
