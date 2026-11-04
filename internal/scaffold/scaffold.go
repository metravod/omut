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
	Removed []string
	Notes   []string
}

type Options struct {
	By      string // who lays the files down, goes into `by:`
	Now     time.Time
	Private bool // start: keep the whole meta out of git (D16)
}

func (o Options) date() string {
	if o.Now.IsZero() {
		return time.Now().Format(meta.DateLayout)
	}
	return o.Now.Format(meta.DateLayout)
}

// toolFiles belong to the tool: zero rewrites them to the current template.
var toolFiles = []string{pack.LeadFile}

// retiredFiles were tool files of the root in earlier versions; zero
// removes them when they still carry the tool-file mark. conventions.md
// moved into every project as .omut/README.md, so a project read without
// its pack still carries its rules.
var retiredFiles = []string{"conventions.md"}

const toolMark = "<!-- Tool file:"

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
	for _, file := range retiredFiles {
		w.retire(filepath.Join(pack.MetaDir, file))
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
	return StartAt(p.Dir(name), opt)
}

// StartAt does what Start does for any directory, inside a pack or not: a
// single repository carries its meta and rules without a pack around it.
func StartAt(dir string, opt Options) (*Result, error) {
	name := filepath.Base(dir)
	if d, err := meta.Read(filepath.Join(dir, pack.MetaDir, pack.Brief)); err == nil && d.Front["status"] == "frozen" {
		return nil, fmt.Errorf("%s is frozen: omut thaw %s", name, name)
	}
	info, err := os.Stat(dir)
	switch {
	case errors.Is(err, fs.ErrNotExist) && opt.Private:
		return nil, fmt.Errorf("no directory %s: --private is for a repository that already exists", dir)
	case errors.Is(err, fs.ErrNotExist):
		if err := os.Mkdir(dir, 0o755); err != nil {
			return nil, err
		}
	case err != nil:
		return nil, err
	case !info.IsDir():
		return nil, fmt.Errorf("%s is a file, not a directory", dir)
	}
	if opt.Private {
		if pack.MetaTracked(dir) {
			return nil, fmt.Errorf("%s has its meta in git already: --private would hide it from this copy only, while the repository keeps it; remove .omut/ from git first if that is what you want", filepath.Base(dir))
		}
		changed, err := pack.ExcludePrivate(dir)
		if err != nil {
			return nil, err
		}
		res, err := zeroProject(dir, opt)
		if res != nil && changed {
			res.Updated = append(res.Updated, ".git/info/exclude (.omut/, CLAUDE.local.md)")
			// Meta laid down earlier without --private becomes private as it
			// is; what that start did to the repository's own files is the
			// user's to undo.
			if hasBlock(filepath.Join(dir, pack.Agents)) {
				if seen := pack.GitSees(dir, pack.Agents, pack.Claude); len(seen) > 0 {
					res.Notes = append(res.Notes, fmt.Sprintf("git still sees %s from an earlier omut start: to leave the repository unchanged, undo them (the omut block, the CLAUDE.md link), then run omut zero.", strings.Join(seen, " and ")))
				}
			}
		}
		return res, err
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

// ZeroAt repairs the scaffold of a project outside any pack.
func ZeroAt(dir string, opt Options) (*Result, error) { return zeroProject(dir, opt) }

func zeroProject(dir string, opt Options) (*Result, error) {
	res := &Result{}
	w := writer{dir: dir, res: res, opt: opt}
	w.tool("project/omut/"+pack.Readme, filepath.Join(pack.MetaDir, pack.Readme))
	// The template is "gitignore": go:embed leaves out files whose name
	// starts with a dot.
	w.tool("project/omut/gitignore", filepath.Join(pack.MetaDir, pack.Ignore))
	w.tool("project/omut/gitattributes", filepath.Join(pack.MetaDir, pack.Attributes))
	for _, file := range projectMeta {
		w.meta("project/omut/"+file, filepath.Join(pack.MetaDir, file))
	}
	for _, file := range []string{pack.Instance, pack.Log} {
		w.meta("project/omut/local/"+file, filepath.Join(pack.MetaDir, pack.Local, file))
	}
	w.mkdir(filepath.Join(pack.MetaDir, pack.Threads))
	// Private meta (D16) leaves the repository's own files alone: the
	// pointer goes to CLAUDE.local.md, which git ignores too, unless
	// AGENTS.md carries the omut block already.
	if pack.Private(dir) && !hasBlock(w.path(pack.Agents)) {
		w.blockIn("project/block.md", pack.ClaudeLocal)
		return res, w.err
	}
	w.block("project/block.md")
	w.claudeLink()
	return res, w.err
}

func hasBlock(path string) bool {
	b, err := os.ReadFile(path)
	return err == nil && strings.Contains(string(b), pack.BlockStart)
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

func (w *writer) retire(rel string) {
	if w.err != nil {
		return
	}
	b, err := os.ReadFile(w.path(rel))
	if err != nil || !strings.HasPrefix(string(b), toolMark) {
		return
	}
	if w.err = os.Remove(w.path(rel)); w.err == nil {
		w.res.Removed = append(w.res.Removed, rel)
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

func (w *writer) block(tmpl string) { w.blockIn(tmpl, pack.Agents) }

func (w *writer) blockIn(tmpl, file string) {
	if w.err != nil {
		return
	}
	created, changed, err := UpsertBlock(w.path(file), w.render(tmpl))
	switch {
	case err != nil:
		w.err = err
	case created:
		w.res.Created = append(w.res.Created, file)
	case changed:
		w.res.Updated = append(w.res.Updated, file+" (omut block)")
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
