// Package scaffold lays down and repairs the files of a pack. It writes
// tool files and empty templates; meta written by agents is never
// overwritten.
package scaffold

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/metravod/omut/internal/meta"
	"github.com/metravod/omut/internal/pack"
	"github.com/metravod/omut/templates"
)

// Result lists what a command did, relative to the directory it worked in.
type Result struct {
	Created []string
	Updated []string
	Notes   []string
}

type Options struct {
	By  string // who lays the files down, goes into `by:`
	Now time.Time
}

func (o Options) date() string {
	if o.Now.IsZero() {
		return time.Now().Format(meta.DateLayout)
	}
	return o.Now.Format(meta.DateLayout)
}

// toolFiles belong to the tool: zero rewrites them to the current template.
var toolFiles = []string{pack.LeadFile, pack.Conventions}

// rootMeta and projectMeta belong to agents: created when missing, never
// rewritten.
var (
	rootMeta    = []string{pack.Brief, pack.Decisions, pack.Graveyard}
	projectMeta = []string{pack.Brief, pack.Decisions, pack.Log}
)

// Init makes dir a pack root. It refuses when dir is one already.
func Init(dir string, opt Options) (*Result, error) {
	if pack.IsRoot(dir) {
		return nil, fmt.Errorf("%s is a pack already: to repair or update it, run omut zero", dir)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	return zeroRoot(dir, opt)
}

// ZeroRoot brings the root scaffold back to the current version.
func ZeroRoot(p *pack.Pack, opt Options) (*Result, error) { return zeroRoot(p.Root, opt) }

func zeroRoot(dir string, opt Options) (*Result, error) {
	res := &Result{}
	w := writer{dir: dir, res: res, opt: opt}
	for _, file := range toolFiles {
		w.tool("root/omut/"+file, filepath.Join(pack.MetaDir, file))
	}
	for _, file := range rootMeta {
		w.meta("root/omut/"+file, filepath.Join(pack.MetaDir, file))
	}
	w.block("root/block.md")
	w.claudeLink()
	return res, w.err
}

// Start adds a project to the pack: creates the directory when missing,
// otherwise only adds what is missing of .omut/ and the pointer block.
func Start(p *pack.Pack, name string, opt Options) (*Result, error) {
	if err := pack.ValidName(name); err != nil {
		return nil, err
	}
	dir := p.Dir(name)
	if d, err := meta.Read(filepath.Join(dir, pack.MetaDir, pack.Brief)); err == nil && d.Front["status"] == "frozen" {
		return nil, fmt.Errorf("%s is frozen: omut thaw %s", name, name)
	}
	info, err := os.Stat(dir)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		if err := os.Mkdir(dir, 0o755); err != nil {
			return nil, err
		}
	case err != nil:
		return nil, err
	case !info.IsDir():
		return nil, fmt.Errorf("%s is a file, not a directory", dir)
	}
	return zeroProject(dir, opt)
}

// ZeroProject repairs a project's scaffold without touching its meta.
func ZeroProject(p *pack.Pack, name string, opt Options) (*Result, error) {
	dir := p.Dir(name)
	if _, err := os.Stat(dir); err != nil {
		return nil, fmt.Errorf("no project %s", name)
	}
	return zeroProject(dir, opt)
}

func zeroProject(dir string, opt Options) (*Result, error) {
	res := &Result{}
	w := writer{dir: dir, res: res, opt: opt}
	for _, file := range projectMeta {
		w.meta("project/omut/"+file, filepath.Join(pack.MetaDir, file))
	}
	w.mkdir(filepath.Join(pack.MetaDir, pack.Threads))
	w.block("project/block.md")
	w.claudeLink()
	return res, w.err
}

// writer collects the first error so the steps read as a plain list.
type writer struct {
	dir string
	res *Result
	opt Options
	err error
}

func (w *writer) render(tmpl string) string {
	b, err := templates.FS.ReadFile(tmpl)
	if err != nil {
		w.err = err
		return ""
	}
	s := strings.ReplaceAll(string(b), "{{date}}", w.opt.date())
	return strings.ReplaceAll(s, "{{by}}", w.opt.By)
}

func (w *writer) path(rel string) string { return filepath.Join(w.dir, rel) }

func (w *writer) mkdir(rel string) {
	if w.err != nil {
		return
	}
	if _, err := os.Stat(w.path(rel)); err == nil {
		return
	}
	if w.err = os.MkdirAll(w.path(rel), 0o755); w.err == nil {
		w.res.Created = append(w.res.Created, rel+"/")
	}
}

func (w *writer) tool(tmpl, rel string) {
	if w.err != nil {
		return
	}
	want := w.render(tmpl)
	have, err := os.ReadFile(w.path(rel))
	if err == nil && string(have) == want {
		return
	}
	w.mkdir(filepath.Dir(rel))
	if w.err = os.WriteFile(w.path(rel), []byte(want), 0o644); w.err != nil {
		return
	}
	if err == nil {
		w.res.Updated = append(w.res.Updated, rel)
	} else {
		w.res.Created = append(w.res.Created, rel)
	}
}

func (w *writer) meta(tmpl, rel string) {
	if w.err != nil {
		return
	}
	if _, err := os.Stat(w.path(rel)); err == nil {
		return
	}
	w.mkdir(filepath.Dir(rel))
	if w.err = os.WriteFile(w.path(rel), []byte(w.render(tmpl)), 0o644); w.err == nil {
		w.res.Created = append(w.res.Created, rel)
	}
}

func (w *writer) block(tmpl string) {
	if w.err != nil {
		return
	}
	created, changed, err := UpsertBlock(w.path(pack.Agents), w.render(tmpl))
	switch {
	case err != nil:
		w.err = err
	case created:
		w.res.Created = append(w.res.Created, pack.Agents)
	case changed:
		w.res.Updated = append(w.res.Updated, pack.Agents+" (omut block)")
	}
}

// claudeLink points CLAUDE.md at AGENTS.md when there is no CLAUDE.md, so
// Claude Code reads the same rules as every other agent. An existing
// CLAUDE.md is the user's and stays as it is; check warns about it.
func (w *writer) claudeLink() {
	if w.err != nil {
		return
	}
	if _, err := os.Lstat(w.path(pack.Claude)); err == nil {
		return
	}
	if w.err = os.Symlink(pack.Agents, w.path(pack.Claude)); w.err == nil {
		w.res.Created = append(w.res.Created, pack.Claude+" → "+pack.Agents)
	}
}

// UpsertBlock puts body between the omut markers of an AGENTS.md:
// replaces the block when it is there, appends it otherwise, creates the
// file when it is missing. Everything outside the markers stays as is.
func UpsertBlock(path, body string) (created, changed bool, err error) {
	block := pack.BlockStart + "\n" + strings.TrimRight(body, "\n") + "\n" + pack.BlockEnd + "\n"
	b, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return true, true, os.WriteFile(path, []byte(block), 0o644)
	}
	if err != nil {
		return false, false, err
	}
	text := string(b)
	var next string
	start := strings.Index(text, pack.BlockStart)
	end := strings.Index(text, pack.BlockEnd)
	switch {
	case start >= 0 && end > start:
		rest := text[end+len(pack.BlockEnd):]
		rest = strings.TrimPrefix(rest, "\n")
		next = text[:start] + block + rest
	case start >= 0 || end >= 0:
		return false, false, fmt.Errorf("%s: the omut block is broken, one marker has no pair; fix it by hand", path)
	default:
		sep := "\n"
		if !strings.HasSuffix(text, "\n") {
			sep = "\n\n"
		}
		if text == "" {
			sep = ""
		}
		next = text + sep + block
	}
	if next == text {
		return false, false, nil
	}
	info, err := os.Stat(path)
	if err != nil {
		return false, false, err
	}
	return false, true, os.WriteFile(path, []byte(next), info.Mode().Perm())
}
