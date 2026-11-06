// Package context assembles recorded project state without interpreting it.
// Selection follows fields and statuses; every excerpt points to its source.
package context

import (
	"cmp"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/metravod/omut/internal/check"
	"github.com/metravod/omut/internal/meta"
	"github.com/metravod/omut/internal/pack"
)

const packSelection = "Pack brief and decisions; each project's Why, What it is, Now and Next; open/doubted decisions and waiting ADR titles; unfinished thread titles, Goal, Open and Next step. Frozen projects: brief and Git observation only. Project footprints, local memory, logs, completed threads and decided project records are omitted; use omut context <project> for current project records. Git observations include branch/upstream, local ahead/behind and changed paths, with no fetch or full diffs."
const projectSelection = "Full brief and decisions; full active/parked threads and waiting ADRs; filled local instance memory and the latest five entries of each log. Completed/dropped threads, decided ADR bodies and older history are omitted. Invalid selected records and read failures are reported. Git observations include branch/upstream, local ahead/behind and changed paths, with no fetch or full diffs."

// Source is a literal excerpt. Line is 1-based; only line endings are
// normalized. File is relative to Report.Root, possibly outside it for local
// memory in a linked worktree. No text is shortened to fit an output budget.
type Source struct {
	Kind          string `json:"kind"`
	File          string `json:"file"`
	Line          int    `json:"line"`
	Text          string `json:"text"`
	Section       string `json:"section,omitempty"`
	Status        string `json:"status,omitempty"`
	Updated       string `json:"updated,omitempty"`
	By            string `json:"by,omitempty"`
	ParkedBecause string `json:"parked_because,omitempty"`
}

type Project struct {
	Name     string    `json:"name"`
	Dir      string    `json:"dir"`
	Instance string    `json:"instance_dir"`
	Worktree string    `json:"worktree,omitempty"`
	Private  bool      `json:"private"`
	Status   string    `json:"status"`
	Updated  string    `json:"updated"`
	Verified string    `json:"verified"`
	Sources  []Source  `json:"sources"`
	Git      *GitState `json:"git"`
}

type Problem struct {
	File    string `json:"file"`
	Message string `json:"message"`
}

type Report struct {
	Scope      string          `json:"scope"`
	Root       string          `json:"root"`
	AsOf       string          `json:"as_of"`
	Selection  string          `json:"selection"`
	Pack       *Project        `json:"pack,omitempty"`
	Projects   []Project       `json:"projects"`
	Findings   []check.Finding `json:"findings"`
	Problems   []Problem       `json:"problems"`
	Incomplete bool            `json:"incomplete"`
}

func newReport(scope, root, selection string, opt check.Options) *Report {
	root, _ = filepath.Abs(root)
	root = resolved(root)
	now := opt.Now
	if now.IsZero() {
		now = time.Now()
	}
	return &Report{Scope: scope, Root: root, AsOf: now.Format(meta.DateLayout), Selection: selection,
		Projects: []Project{}, Findings: []check.Finding{}, Problems: []Problem{}}
}

// ForPack collects an overview, not a second store or an agent-written summary.
// A project with missing or unreadable meta still appears in the report.
func ForPack(p *pack.Pack, opt check.Options) *Report {
	r := newReport("pack", p.Root, packSelection, opt)
	p = &pack.Pack{Root: r.Root}
	r.Pack = &Project{Name: filepath.Base(r.Root), Dir: r.Root, Instance: r.Root, Sources: []Source{}}
	r.Pack.Git = r.git(r.Root)
	if d := r.read(p.Lead(pack.Brief), false); d != nil {
		r.Pack.Status, r.Pack.Updated, r.Pack.Verified = d.Front["status"], d.Front["updated"], d.Front["verified"]
		r.Pack.Sources = append(r.Pack.Sources, r.source("brief", d, 0, len(d.Lines)))
	}
	if d := r.read(p.Lead(pack.Decisions), false); d != nil {
		r.Pack.Sources = append(r.Pack.Sources, r.source("decisions", d, 0, len(d.Lines)))
	}
	names, problems, err := p.ScanProjects()
	if err != nil {
		r.problem(p.Root, err)
	}
	for _, problem := range problems {
		r.problem(problem.Path, problem.Err)
	}
	for _, name := range names {
		dir := p.Dir(name)
		pl := pack.Locate(dir)
		if pl.Dir == "" || pl.Foreign || pack.IsRoot(pl.Dir) {
			pl = pack.Place{Dir: dir, Instance: dir}
		}
		r.Projects = append(r.Projects, r.project(name, pl, true))
	}
	r.Findings, err = check.Pack(p, opt)
	if err != nil {
		r.problem(p.Root, err)
	}
	r.finish()
	return r
}

// ForProject includes current records in full, including accepted decisions.
// Place preserves branch-owned meta and main-checkout instance memory.
func ForProject(pl pack.Place, opt check.Options) *Report {
	name := filepath.Base(pl.Dir)
	pl.Dir, pl.Instance = resolved(pl.Dir), resolved(pl.Instance)
	if pl.Worktree != "" {
		pl.Worktree = resolved(pl.Worktree)
	}
	if pl.WorkDir != "" {
		pl.WorkDir = resolved(pl.WorkDir)
	}
	r := newReport("project", pl.Dir, projectSelection, opt)
	r.Projects = append(r.Projects, r.project(name, pl, false))
	r.Findings = check.Dir(pl.Dir, opt)
	// check uses paths relative to the project's parent; context sources use
	// the project itself. Normalize so both refer to the same report root.
	for i := range r.Findings {
		f := &r.Findings[i]
		if f.Project != "" {
			f.Project = name
		}
		if f.File != "" {
			f.File = r.relative(filepath.Join(filepath.Dir(pl.Dir), filepath.FromSlash(f.File)))
		}
	}
	r.finish()
	return r
}

func resolved(path string) string {
	if real, err := filepath.EvalSymlinks(path); err == nil {
		return real
	}
	return path
}

func (r *Report) finish() {
	if r.Findings == nil {
		r.Findings = []check.Finding{}
	}
	slices.SortStableFunc(r.Findings, func(a, b check.Finding) int {
		return cmp.Or(cmp.Compare(a.Project, b.Project), cmp.Compare(a.File, b.File),
			cmp.Compare(a.Line, b.Line), cmp.Compare(a.Rule, b.Rule), cmp.Compare(a.Message, b.Message))
	})
	r.Incomplete = len(r.Problems) > 0 || check.HasErrors(r.Findings)
}

func (r *Report) relative(path string) string {
	rel, err := filepath.Rel(r.Root, path)
	if err != nil {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(rel)
}

func (r *Report) problem(path string, err error) {
	r.Problems = append(r.Problems, Problem{File: r.relative(path), Message: err.Error()})
}

func (r *Report) read(path string, optional bool) *meta.Doc {
	d, err := meta.Read(path)
	if err != nil {
		_, statErr := os.Lstat(path)
		if !(optional && os.IsNotExist(err) && os.IsNotExist(statErr)) {
			r.problem(path, err)
		}
	}
	return d
}

func (r *Report) source(kind string, d *meta.Doc, start, end int) Source {
	return Source{Kind: kind, File: r.relative(d.Path), Line: start + 1, Text: strings.Join(d.Lines[start:end], "\n")}
}

func (r *Report) project(name string, pl pack.Place, overview bool) Project {
	p := Project{Name: name, Dir: pl.Dir, Instance: pl.Instance, Worktree: pl.Worktree,
		Private: pack.Private(pl.Dir), Sources: []Source{}}
	work := pl.Dir
	if pl.WorkDir != "" {
		work = pl.WorkDir
	}
	p.Git = r.git(work)
	lead := filepath.Join(pl.Dir, pack.MetaDir)
	brief := r.read(filepath.Join(lead, pack.Brief), false)
	if brief != nil {
		p.Status, p.Updated, p.Verified = brief.Front["status"], brief.Front["updated"], brief.Front["verified"]
		if overview {
			for _, s := range brief.Sections() {
				if !slices.Contains([]string{"Why", "What it is", "Now", "Next"}, s.Title) {
					continue
				}
				source := r.source("brief", brief, s.Line-1, s.End)
				source.Section = s.Title
				p.Sources = append(p.Sources, source)
			}
		} else {
			p.Sources = append(p.Sources, r.source("brief", brief, 0, len(brief.Lines)))
		}
	}
	if overview && p.Status == "frozen" {
		return p
	}
	if d := r.read(filepath.Join(lead, pack.Decisions), false); d != nil {
		if overview {
			decisions, bad := d.Decisions()
			for _, dec := range decisions {
				if dec.Status == "open" || dec.Status == "doubted" {
					s := r.source("decision", d, dec.Line-1, dec.Line)
					s.Status = dec.Status
					p.Sources = append(p.Sources, s)
				}
			}
			for _, line := range bad {
				p.Sources = append(p.Sources, r.source("invalid-decision", d, line-1, line))
			}
		} else {
			p.Sources = append(p.Sources, r.source("decisions", d, 0, len(d.Lines)))
		}
	}
	r.threads(&p, lead, overview)
	if brief != nil {
		r.adrs(&p, brief, overview)
	}
	if !overview {
		local := filepath.Join(pl.Instance, pack.MetaDir, pack.Local)
		if _, err := os.Stat(local); err != nil {
			if _, statErr := os.Lstat(local); !os.IsNotExist(err) || !os.IsNotExist(statErr) {
				r.problem(local, err)
			}
		}
		if d := r.read(filepath.Join(local, pack.Instance), true); d != nil {
			if slices.ContainsFunc(d.Sections(), d.Filled) {
				p.Sources = append(p.Sources, r.source("instance", d, 0, len(d.Lines)))
			}
		}
		r.log(&p, filepath.Join(lead, pack.Log), false)
		r.log(&p, filepath.Join(local, pack.Log), true)
	}
	return p
}

func (r *Report) threads(p *Project, lead string, overview bool) {
	dir := filepath.Join(lead, pack.Threads)
	entries, err := os.ReadDir(dir)
	if err != nil {
		// Git does not preserve an empty threads/ directory. Its absence
		// means no threads; an existing but unreadable/broken path is an error.
		if _, statErr := os.Lstat(dir); !os.IsNotExist(err) || !os.IsNotExist(statErr) {
			r.problem(dir, err)
		}
		return
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		d := r.read(filepath.Join(dir, e.Name()), false)
		if d == nil || d.Front["status"] == "done" || d.Front["status"] == "dropped" {
			continue
		}
		appendSource := func(start, end int) {
			s := r.source("thread", d, start, end)
			s.Status, s.Updated, s.By, s.ParkedBecause = d.Front["status"], d.Front["updated"], d.Front["by"], d.Front["parked_because"]
			if n := len(p.Sources); n > 0 {
				prev := &p.Sources[n-1]
				if prev.File == s.File && prev.Line+strings.Count(prev.Text, "\n")+1 == s.Line {
					prev.Text += "\n" + s.Text
					return
				}
			}
			p.Sources = append(p.Sources, s)
		}
		if !overview || (d.Front["status"] != "active" && d.Front["status"] != "parked") {
			appendSource(0, len(d.Lines))
			continue
		}
		before := len(p.Sources)
		for i := d.BodyStart; i < len(d.Clean); i++ {
			line := d.Clean[i]
			if strings.HasPrefix(line, "# ") {
				appendSource(i, i+1)
			} else if strings.HasPrefix(line, "**Goal:**") || strings.HasPrefix(line, "**Open:**") || strings.HasPrefix(line, "**Next step:**") {
				end := i + 1
				for end < len(d.Clean) && !strings.HasPrefix(d.Clean[end], "**") && !strings.HasPrefix(d.Clean[end], "#") {
					end++
				}
				appendSource(i, end)
				i = end - 1
			}
		}
		if before == len(p.Sources) {
			appendSource(0, len(d.Lines))
		}
	}
}

func (r *Report) adrs(p *Project, brief *meta.Doc, overview bool) {
	if brief.Front["adr"] == "" {
		return
	}
	dir := filepath.Join(p.Dir, brief.Front["adr"])
	paths, err := meta.ADRFiles(dir)
	if err != nil {
		r.problem(dir, err)
		return
	}
	for _, path := range paths {
		d := r.read(path, false)
		if d == nil {
			continue
		}
		a := meta.ParseADR(d)
		if a.Waiting() == "" {
			continue
		}
		s := r.source("adr", d, 0, len(d.Lines))
		if overview {
			start := max(0, a.Line-1)
			for i, line := range d.Clean {
				if strings.HasPrefix(strings.TrimSpace(line), "# ") {
					start = i
					break
				}
			}
			s = r.source("adr", d, start, start+1)
		}
		s.Status = a.Waiting()
		p.Sources = append(p.Sources, s)
	}
}

func (r *Report) log(p *Project, path string, optional bool) {
	d := r.read(path, optional)
	if d == nil {
		return
	}
	entries, bad := d.LogEntries()
	for _, e := range entries[:min(5, len(entries))] {
		p.Sources = append(p.Sources, r.source("log", d, e.Line-1, e.Line))
	}
	for _, line := range bad {
		r.problem(path, fmt.Errorf("invalid log entry at line %d", line))
		p.Sources = append(p.Sources, r.source("invalid-log", d, line-1, line))
	}
}

// Markdown is the same selection as JSON, with explicit source locations.
func (r *Report) Markdown() string {
	var b strings.Builder
	fmt.Fprintf(&b, "# omut context: %s\n\nRoot: `%s` · as of %s\n\nSelection: %s\n\nMeta excerpts are recorded state, not a verification of their truth. Their paths are relative to Root. Git observations are current local reads; divergence uses cached local refs, not a server check. Full diffs are omitted.\n", r.Scope, r.Root, r.AsOf, r.Selection)
	writeProject := func(p Project, isPack bool) {
		fmt.Fprintf(&b, "\n## %s\n\nDirectory: `%s`\n", p.Name, p.Dir)
		if isPack {
			fmt.Fprintf(&b, "Updated: %s\n", p.Updated)
		} else {
			fmt.Fprintf(&b, "Status: %s · updated: %s · verified: %s\n", p.Status, p.Updated, p.Verified)
		}
		if p.Private {
			b.WriteString("Private meta: keep it out of commits.\n")
		}
		if p.Worktree != "" {
			fmt.Fprintf(&b, "Working copy: `%s`; project memory is in `%s`.\n", p.Worktree, p.Dir)
		}
		if p.Instance != p.Dir {
			fmt.Fprintf(&b, "This copy's memory: `%s` (main checkout).\n", filepath.Join(p.Instance, pack.MetaDir, pack.Local))
		}
		if p.Git != nil {
			b.WriteString(p.Git.markdown())
		}
		for _, s := range p.Sources {
			fmt.Fprintf(&b, "\n### %s — `%s:%d`", s.Kind, s.File, s.Line)
			if s.Status != "" {
				fmt.Fprintf(&b, " [%s", s.Status)
				if s.ParkedBecause != "" {
					fmt.Fprintf(&b, ": %s", s.ParkedBecause)
				}
				b.WriteString("]")
			}
			if s.Updated != "" {
				fmt.Fprintf(&b, " updated %s", s.Updated)
			}
			if s.By != "" {
				fmt.Fprintf(&b, " · by %s", s.By)
			}
			fmt.Fprintf(&b, "\n\n%s\n", s.Text)
		}
	}
	if r.Pack != nil {
		writeProject(*r.Pack, true)
	}
	for _, p := range r.Projects {
		writeProject(p, false)
	}
	b.WriteString("\n## Findings (omut check)\n\n")
	if len(r.Findings) == 0 {
		b.WriteString("None.\n")
	}
	for _, f := range r.Findings {
		fmt.Fprintf(&b, "- %s %s: %s (`%s:%d`)\n", f.Level, f.Rule, f.Message, f.File, f.Line)
	}
	if r.Incomplete {
		b.WriteString("\n## Incomplete context\n\nRead/Git failures or shape errors require attention; missing data does not mean nothing is pending.\n")
	}
	for _, p := range r.Problems {
		fmt.Fprintf(&b, "- `%s`: %s\n", p.File, p.Message)
	}
	return b.String()
}
