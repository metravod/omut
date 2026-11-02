// Package check compares the meta with its form and with what can be
// observed on disk. It reports and never changes anything.
package check

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/metravod/omut/internal/meta"
	"github.com/metravod/omut/internal/pack"
	"github.com/metravod/omut/templates"
)

type Level string

const (
	Error Level = "error"
	Warn  Level = "warn"
	Info  Level = "info"
)

// Thresholds from the spec. They are fixed for now; see spec §12.
const (
	BriefLines    = 60
	ThreadLines   = 80
	LogEntries    = 50
	VerifyDays    = 30
	StalledDays   = 14
	maxWalkedFile = 200_000
)

type Finding struct {
	Project string `json:"project"` // "" for the pack itself
	Rule    string `json:"rule"`
	Level   Level  `json:"level"`
	File    string `json:"file,omitempty"` // relative to the pack root
	Line    int    `json:"line,omitempty"`
	Message string `json:"message"`
}

type Options struct {
	Now time.Time
}

func (o Options) today() time.Time {
	now := o.Now
	if now.IsZero() {
		now = time.Now()
	}
	y, m, d := now.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.Local)
}

// HasWarnings reports whether any finding is a warning.
func HasWarnings(fs []Finding) bool {
	return slices.ContainsFunc(fs, func(f Finding) bool { return f.Level == Warn })
}

// HasErrors reports whether any finding is a form error.
func HasErrors(fs []Finding) bool {
	return slices.ContainsFunc(fs, func(f Finding) bool { return f.Level == Error })
}

type report struct {
	base    string // paths in findings are relative to it
	project string
	out     []Finding
}

func (r *report) add(rule string, level Level, path string, line int, format string, args ...any) {
	rel := path
	if path != "" {
		if p, err := filepath.Rel(r.base, path); err == nil {
			rel = p
		}
	}
	r.out = append(r.out, Finding{Project: r.project, Rule: rule, Level: level, File: rel, Line: line, Message: fmt.Sprintf(format, args...)})
}

// Pack checks the root and every project.
func Pack(p *pack.Pack, opt Options) ([]Finding, error) {
	r := &report{base: p.Root}
	checkRoot(r, p)
	names, err := p.Projects()
	if err != nil {
		return nil, err
	}
	dead := p.Graveyard()
	for _, name := range names {
		r.out = append(r.out, project(p.Root, p.Dir(name), name, opt, dead)...)
	}
	return r.out, nil
}

// Project checks one project of the pack.
func Project(p *pack.Pack, name string, opt Options) []Finding {
	return project(p.Root, p.Dir(name), name, opt, p.Graveyard())
}

// Standalone checks a project that sits outside any pack, such as a
// colleague's clone.
func Standalone(dir string, opt Options) []Finding {
	return project(filepath.Dir(dir), dir, filepath.Base(dir), opt, nil)
}

// Dir checks the project at dir, as a project of its pack when it sits
// right under a pack root, on its own otherwise.
func Dir(dir string, opt Options) []Finding {
	if p, err := pack.Find(filepath.Dir(dir)); err == nil && p.Root == filepath.Dir(dir) {
		return Project(p, filepath.Base(dir), opt)
	}
	return Standalone(dir, opt)
}

func checkRoot(r *report, p *pack.Pack) {
	for _, file := range []string{pack.LeadFile} {
		path := p.Lead(file)
		b, err := os.ReadFile(path)
		if err != nil {
			r.add("scaffold", Error, path, 0, "tool file missing: omut zero restores it")
			continue
		}
		if tmpl, err := templates.FS.ReadFile("root/omut/" + file); err == nil && string(tmpl) != string(b) {
			r.add("outdated", Info, path, 0, "differs from this version's template: omut zero updates it")
		}
	}
	for _, file := range []string{pack.Brief, pack.Decisions, pack.Graveyard} {
		if _, err := os.Stat(p.Lead(file)); err != nil {
			r.add("scaffold", Error, p.Lead(file), 0, "pack meta file missing: omut zero creates a template")
		}
	}
	if d, err := meta.Read(p.Lead(pack.Brief)); err == nil {
		if !d.HasFront {
			r.add("shape", Error, d.Path, 1, "no frontmatter")
		}
		for _, name := range d.Lists["ignore"] {
			if _, err := os.Stat(p.Dir(name)); err != nil {
				r.add("shape", Warn, d.Path, d.FrontLine["ignore"], "ignore lists %q, there is no such directory", name)
			}
		}
		checkRefs(r, d, p.Root)
	}
	if d, err := meta.Read(p.Lead(pack.Decisions)); err == nil {
		checkDecisions(r, d)
		checkRefs(r, d, p.Root)
	}
	checkPointer(r, p.Root)
}

func project(base, dir, name string, opt Options, dead []string) []Finding {
	r := &report{base: base, project: name}
	lead := filepath.Join(dir, pack.MetaDir)
	if _, err := os.Stat(lead); err != nil {
		r.add("no-meta", Error, dir, 0, "project without meta: omut start %s, or add it to the pack's ignore", name)
		return r.out
	}
	brief, err := meta.Read(filepath.Join(lead, pack.Brief))
	if err != nil {
		r.add("shape", Error, filepath.Join(lead, pack.Brief), 0, "no brief.md: omut zero %s creates a template", name)
	} else {
		checkBrief(r, brief, dir, opt)
		checkRefs(r, brief, dir)
	}
	if d, err := meta.Read(filepath.Join(lead, pack.Decisions)); err == nil {
		checkDecisions(r, d)
		checkRefs(r, d, dir)
		if brief != nil {
			checkADR(r, brief, d, dir)
		}
	} else {
		r.add("shape", Error, filepath.Join(lead, pack.Decisions), 0, "no decisions.md: omut zero %s creates a template", name)
	}
	if d, err := meta.Read(filepath.Join(lead, pack.Log)); err == nil {
		checkLog(r, d)
		checkRefs(r, d, dir)
	} else {
		r.add("shape", Error, filepath.Join(lead, pack.Log), 0, "no log.md: omut zero %s creates a template", name)
	}
	threads, _ := filepath.Glob(filepath.Join(lead, pack.Threads, "*.md"))
	for _, path := range threads {
		if d, err := meta.Read(path); err == nil {
			checkThread(r, d, opt)
			checkRefs(r, d, dir)
		}
	}
	checkReadme(r, lead, name)
	checkPointer(r, dir)
	if len(dead) > 0 {
		checkGhosts(r, lead, name, dead)
	}
	return r.out
}

var briefSections = []string{"Why", "What it is", "Where it lives", "Now", "Next"}

func checkBrief(r *report, d *meta.Doc, dir string, opt Options) {
	if d.LineCount() > BriefLines {
		r.add("size", Warn, d.Path, 0, "%d lines, budget %d", d.LineCount(), BriefLines)
	}
	if !d.HasFront {
		r.add("shape", Error, d.Path, 1, "no frontmatter")
		return
	}
	for _, key := range []string{"status", "updated", "verified", "by"} {
		if _, ok := d.Front[key]; !ok {
			r.add("shape", Error, d.Path, 1, "no %s in frontmatter", key)
		}
	}
	status := d.Front["status"]
	if _, ok := d.Front["status"]; ok && status != "active" && status != "frozen" {
		r.add("shape", Error, d.Path, d.FrontLine["status"], "status %q: allowed are active, frozen", status)
	}
	updated, updatedOK := meta.ParseDate(d.Front["updated"])
	if _, ok := d.Front["updated"]; ok && !updatedOK {
		r.add("shape", Error, d.Path, d.FrontLine["updated"], "updated %q: want a YYYY-MM-DD date", d.Front["updated"])
	}
	if _, ok := d.Front["by"]; ok && d.Front["by"] == "" {
		r.add("unfilled", Warn, d.Path, d.FrontLine["by"], "by is empty: who wrote the brief (agent/model or human)")
	}
	for _, title := range briefSections {
		s, ok := d.Section(title)
		if !ok {
			r.add("shape", Error, d.Path, 0, "no section %q", title)
			continue
		}
		if title != "Where it lives" && title != "Next" && !d.Filled(s) {
			r.add("unfilled", Warn, d.Path, s.Line, "section %q is empty", title)
		}
	}

	today := opt.today()
	if v, ok := d.Front["verified"]; ok {
		if v == "" {
			if status == "active" {
				r.add("unverified", Warn, d.Path, d.FrontLine["verified"], "the brief was never checked against reality")
			}
		} else if t, ok := meta.ParseDate(v); !ok {
			r.add("shape", Error, d.Path, d.FrontLine["verified"], "verified %q: want a YYYY-MM-DD date", v)
		} else if status == "active" && today.Sub(t) > VerifyDays*24*time.Hour {
			r.add("unverified", Warn, d.Path, d.FrontLine["verified"], "last checked %s, more than %d days ago", v, VerifyDays)
		}
	}
	if live, ok := d.Section("Where it lives"); ok && status == "frozen" && d.Filled(live) {
		r.add("frozen-but-live", Warn, d.Path, live.Line, "the project is frozen, but \"Where it lives\" is not empty: make sure nothing is still running")
	}
	if updatedOK {
		if when, what := newestChange(dir); !when.IsZero() && dayAfter(when, updated) {
			r.add("stale-brief", Warn, d.Path, d.FrontLine["updated"], "the project changed on %s (%s), the brief was updated %s", when.Format(meta.DateLayout), what, d.Front["updated"])
		}
	}
}

func dayAfter(t, day time.Time) bool {
	return t.Format(meta.DateLayout) > day.Format(meta.DateLayout)
}

// newestChange finds the latest change in the project outside .omut/. In a
// git repository it is the last commit that touched anything outside .omut/
// plus the mtimes of files git reports as changed or untracked, so a fresh
// clone, where every mtime is the clone date, is not a change. Without git
// it is the newest file mtime outside hidden directories. A symlinked
// project is resolved first: the walk doesn't descend into a link.
func newestChange(dir string) (time.Time, string) {
	dir = resolve(dir)
	if isRepo(dir) {
		if when, what, ok := dirtyChange(dir); ok {
			if commit := lastCommit(dir); commit.After(when) {
				return commit, "last commit"
			}
			return when, what
		}
	}
	return walkChange(dir)
}

// NewestUnsaved finds the latest change outside .omut/ that no commit
// holds: in a git repository the changed and untracked files, without git
// any file, since nothing else keeps it.
func NewestUnsaved(dir string) (time.Time, string) {
	dir = resolve(dir)
	if isRepo(dir) {
		if when, what, ok := dirtyChange(dir); ok {
			return when, what
		}
	}
	return walkChange(dir)
}

func resolve(dir string) string {
	if real, err := filepath.EvalSymlinks(dir); err == nil {
		return real
	}
	return dir
}

func isRepo(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, ".git"))
	return err == nil
}

// lastCommit is the date of the last commit that touched anything outside
// .omut/; zero in a repository without commits. It keeps the committer's
// own time zone: `updated` was written in it, and CI runs in UTC.
func lastCommit(dir string) time.Time {
	out, _ := exec.Command("git", "-C", dir, "log", "-1", "--format=%cI", "--", ".", ":(exclude)"+pack.MetaDir).Output()
	if t, err := time.Parse(time.RFC3339, strings.TrimSpace(string(out))); err == nil {
		return t
	}
	return time.Time{}
}

// dirtyChange is the newest mtime among the files git reports as changed
// or untracked, .omut/ aside. ok is false when git can't answer.
func dirtyChange(dir string) (when time.Time, what string, ok bool) {
	out, err := exec.Command("git", "-C", dir, "status", "--porcelain", "-z", "--untracked-files=all").Output()
	if err != nil {
		return when, what, false
	}
	entries := strings.Split(string(out), "\x00")
	for i := 0; i < len(entries); i++ {
		e := entries[i]
		if len(e) < 4 {
			continue
		}
		if e[0] == 'R' || e[0] == 'C' {
			i++ // the next entry is the rename's source
		}
		rel := e[3:]
		if rel == pack.MetaDir || strings.HasPrefix(rel, pack.MetaDir+"/") {
			continue
		}
		if info, err := os.Lstat(filepath.Join(dir, rel)); err == nil && info.ModTime().After(when) {
			when, what = info.ModTime(), rel
		}
	}
	return when, what, true
}

// skipDirs are never part of "the project changed": tool caches and
// dependency trees. Hidden directories are skipped as well.
var skipDirs = []string{"node_modules", "__pycache__", "vendor", "target", "dist", "build"}

func walkChange(dir string) (time.Time, string) {
	var when time.Time
	what := ""
	seen := 0
	filepath.WalkDir(dir, func(path string, e fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		name := e.Name()
		if e.IsDir() {
			if path != dir && (strings.HasPrefix(name, ".") || slices.Contains(skipDirs, name)) {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasPrefix(name, ".") {
			return nil
		}
		if seen++; seen > maxWalkedFile {
			return filepath.SkipAll
		}
		if info, err := e.Info(); err == nil && info.ModTime().After(when) {
			when = info.ModTime()
			what, _ = filepath.Rel(dir, path)
		}
		return nil
	})
	return when, what
}

var (
	threadStatuses = []string{"active", "parked", "done", "dropped"}
	parkReasons    = []string{"stamina", "doubt", "blocked", "limits", "priority", "unknown"}
)

func checkThread(r *report, d *meta.Doc, opt Options) {
	if d.LineCount() > ThreadLines {
		r.add("size", Warn, d.Path, 0, "%d lines, thread budget %d", d.LineCount(), ThreadLines)
	}
	if !d.HasFront {
		r.add("shape", Error, d.Path, 1, "no frontmatter")
		return
	}
	status := d.Front["status"]
	if !slices.Contains(threadStatuses, status) {
		r.add("shape", Error, d.Path, d.FrontLine["status"], "status %q: allowed are %s", status, strings.Join(threadStatuses, ", "))
	}
	if status == "parked" {
		if why := d.Front["parked_because"]; !slices.Contains(parkReasons, why) {
			r.add("shape", Error, d.Path, d.FrontLine["parked_because"], "parked_because %q: allowed are %s", why, strings.Join(parkReasons, ", "))
		}
	}
	updated, ok := meta.ParseDate(d.Front["updated"])
	if !ok {
		r.add("shape", Error, d.Path, d.FrontLine["updated"], "updated %q: want a YYYY-MM-DD date", d.Front["updated"])
	} else if status == "active" && opt.today().Sub(updated) > StalledDays*24*time.Hour {
		r.add("stalled-thread", Info, d.Path, d.FrontLine["updated"], "thread is active but not updated since %s", d.Front["updated"])
	}
	if d.Front["by"] == "" {
		r.add("shape", Error, d.Path, 1, "no by in frontmatter")
	}
	if status == "active" || status == "parked" {
		if !slices.ContainsFunc(d.Clean, func(l string) bool { return strings.Contains(l, "**Next step:**") }) {
			r.add("unfilled", Warn, d.Path, 0, "no \"Next step\": where to pick up")
		}
	}
}

func checkDecisions(r *report, d *meta.Doc) {
	decs, bad := d.Decisions()
	for _, line := range bad {
		r.add("shape", Error, d.Path, line, "decision line out of format: - Dn [open|accepted …|doubted …|superseded → Dn] text")
	}
	seen := map[int]int{}
	for _, dec := range decs {
		if prev, ok := seen[dec.N]; ok {
			r.add("shape", Error, d.Path, dec.Line, "D%d is already on line %d", dec.N, prev)
		}
		seen[dec.N] = dec.Line
	}
	for _, dec := range decs {
		switch dec.Status {
		case "superseded":
			if _, ok := seen[dec.Supersedes]; !ok {
				r.add("shape", Error, d.Path, dec.Line, "D%d is superseded by D%d, which does not exist", dec.N, dec.Supersedes)
			}
		case "open", "doubted":
			r.add("open-decision", Info, d.Path, dec.Line, "D%d [%s] %s", dec.N, dec.Status, short(dec.Text))
		}
	}
}

func checkADR(r *report, brief, decisions *meta.Doc, dir string) {
	rel := brief.Front["adr"]
	if rel == "" {
		return
	}
	adr := filepath.Join(dir, rel)
	info, err := os.Stat(decisions.Path)
	if err != nil {
		return
	}
	var newer []string
	err = filepath.WalkDir(adr, func(path string, e fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if fi, err := e.Info(); err == nil && !e.IsDir() && fi.ModTime().After(info.ModTime()) {
			newer = append(newer, e.Name())
		}
		return nil
	})
	if err != nil {
		r.add("shape", Error, brief.Path, brief.FrontLine["adr"], "adr: %s, no such directory", rel)
		return
	}
	if len(newer) > 0 {
		sort.Strings(newer)
		r.add("adr-ahead", Warn, decisions.Path, 0, "%s has ADRs newer than the registry: %s", rel, strings.Join(newer, ", "))
	}
}

func checkLog(r *report, d *meta.Doc) {
	entries, bad := d.LogEntries()
	for _, line := range bad {
		r.add("shape", Error, d.Path, line, "log entry out of format: - YYYY-MM-DD · agent/model · text")
	}
	if len(entries) > LogEntries {
		r.add("size", Warn, d.Path, 0, "%d entries, time to move old ones to log-archive/", len(entries))
	}
	for i := 1; i < len(entries); i++ {
		if entries[i].Date > entries[i-1].Date {
			r.add("shape", Warn, d.Path, entries[i].Line, "newest entries go on top: %s is below %s", entries[i].Date, entries[i-1].Date)
			break
		}
	}
}

// checkRefs resolves each path the meta points at against the file's own
// directory first, then against the project (or pack) root.
func checkRefs(r *report, d *meta.Doc, root string) {
	for _, ref := range d.Refs() {
		if exists(filepath.Join(filepath.Dir(d.Path), ref.Target)) || exists(filepath.Join(root, ref.Target)) {
			continue
		}
		r.add("shape", Error, d.Path, ref.Line, "broken link: %s", ref.Target)
	}
}

func checkPointer(r *report, dir string) {
	agents := filepath.Join(dir, pack.Agents)
	b, err := os.ReadFile(agents)
	if err != nil {
		r.add("pointer", Warn, agents, 0, "no AGENTS.md, an agent will not learn about the meta; in a pack omut zero creates it")
		return
	}
	if !strings.Contains(string(b), pack.BlockStart) || !strings.Contains(string(b), pack.BlockEnd) {
		r.add("pointer", Warn, agents, 0, "AGENTS.md has no omut block, an agent will not learn about the meta; in a pack omut zero adds it")
	}
	claude := filepath.Join(dir, pack.Claude)
	info, err := os.Lstat(claude)
	if err != nil {
		return
	}
	if info.Mode()&os.ModeSymlink != 0 {
		if target, err := os.Readlink(claude); err == nil && target == pack.Agents {
			return
		}
	}
	// Claude Code reads CLAUDE.local.md as well, so a gitignored local file
	// with @AGENTS.md is a valid way in.
	if leadsToAgents(claude) || leadsToAgents(filepath.Join(dir, pack.ClaudeLocal)) {
		return
	}
	r.add("pointer", Warn, claude, 0, "Claude Code reads CLAUDE.md and will not see AGENTS.md: move the rules to AGENTS.md and make CLAUDE.md a symlink, or add an @AGENTS.md line to CLAUDE.md or CLAUDE.local.md")
}

func leadsToAgents(path string) bool {
	c, err := os.ReadFile(path)
	return err == nil && (strings.Contains(string(c), "@AGENTS.md") || strings.Contains(string(c), pack.BlockStart))
}

// checkReadme looks at the project's copy of the rules: an agent that gets
// the project without its pack learns the format from it.
func checkReadme(r *report, lead, name string) {
	path := filepath.Join(lead, pack.Readme)
	b, err := os.ReadFile(path)
	if err != nil {
		r.add("pointer", Warn, path, 0, "no .omut/README.md, an agent without the pack will not know the rules: omut zero %s creates it", name)
		return
	}
	if tmpl, err := templates.FS.ReadFile("project/omut/" + pack.Readme); err == nil && string(tmpl) != string(b) {
		r.add("outdated", Info, path, 0, "differs from this version's template: omut zero %s updates it", name)
	}
}

func checkGhosts(r *report, lead, self string, dead []string) {
	filepath.WalkDir(lead, func(path string, e fs.DirEntry, err error) error {
		if err != nil || e.IsDir() || !strings.HasSuffix(path, ".md") || e.Name() == pack.Log || path == filepath.Join(lead, pack.Readme) || strings.Contains(path, "log-archive") {
			return nil
		}
		d, err := meta.Read(path)
		if err != nil {
			return nil
		}
		for _, name := range dead {
			if name == self {
				continue
			}
			for _, line := range d.Mentions(name) {
				r.add("ghost", Warn, path, line, "mentions %s, which is in the graveyard", name)
			}
		}
		return nil
	})
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func short(s string) string {
	if r := []rune(s); len(r) > 90 {
		return string(r[:89]) + "…"
	}
	return s
}
