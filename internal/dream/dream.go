// Package dream counts the memory written since the last dream and assembles
// what the dream agent works from: the records newer than dreamed, the check
// findings, and the contract that says what a dream is. It only reads; the
// agent does the consolidating and does the writing.
package dream

import (
	"cmp"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/metravod/omut/internal/check"
	"github.com/metravod/omut/internal/meta"
	"github.com/metravod/omut/internal/pack"
)

// DefaultThreshold is how many records make a dream due, when the brief sets
// none (dream: N in the frontmatter).
const DefaultThreshold = 10

// shownInText is how many records the markdown digest lists; JSON keeps all
// of them. The cap is a selection policy, not truncation of a record.
const shownInText = 20

// archiveRe is the lenient date prefix of a log line: log-archive/ holds
// rotated logs moved as they are, and check does not parse them.
var archiveRe = regexp.MustCompile(`^- (\d{4}-\d{2}-\d{2})`)

// anyDate takes the first date on a line: a decision's or a tombstone's own.
var anyDate = regexp.MustCompile(`\d{4}-\d{2}-\d{2}`)

// Record is one dated piece of memory newer than the last dream. Text is the
// record's own line (a log entry, a decision) or the thread's title.
type Record struct {
	Kind string `json:"kind"` // log | archive | thread | decision | graveyard
	Date string `json:"date"`
	File string `json:"file"` // relative to the scope directory, slash-separated
	Line int    `json:"line"`
	Text string `json:"text"`
}

// Count is the volume of memory since the last dream, with the threshold the
// brief sets. A missing or malformed dreamed: counts the whole memory; the
// nag extinguishes when the dream agent writes the date.
type Count struct {
	Dreamed   string   `json:"dreamed,omitempty"`
	Threshold int      `json:"threshold"`
	Records   []Record `json:"records"`
}

func (c Count) Total() int { return len(c.Records) }

// Due reports whether the reminder should fire: records at or over the
// threshold. A small memory that has never been dreamed stays silent.
func (c Count) Due() bool { return len(c.Records) >= c.Threshold }

// Problem is a file that could not be read while counting.
type Problem struct {
	File    string `json:"file"`
	Message string `json:"message"`
}

// CountDir counts one scope's memory: a project's or a pack root's .omut/.
// Dated log entries (log.md and log-archive/), thread updates and dated
// decisions count; records strictly after dreamed: do, so a dream's own log
// line does not recount itself.
func CountDir(dir string) (Count, []Problem) {
	c := Count{Threshold: DefaultThreshold, Records: []Record{}}
	var problems []Problem
	fail := func(path string, err error) {
		problems = append(problems, Problem{File: rel(dir, path), Message: err.Error()})
	}
	lead := filepath.Join(dir, pack.MetaDir)
	dreamedOK := false
	if brief, err := meta.Read(filepath.Join(lead, pack.Brief)); err != nil {
		if _, statErr := os.Lstat(filepath.Join(lead, pack.Brief)); !os.IsNotExist(err) || !os.IsNotExist(statErr) {
			fail(filepath.Join(lead, pack.Brief), err)
		}
	} else {
		c.Dreamed = brief.Front["dreamed"]
		_, dreamedOK = meta.ParseDate(c.Dreamed)
		if n, err := strconv.Atoi(brief.Front["dream"]); brief.Front["dream"] != "" && err == nil && n > 0 {
			c.Threshold = n
		}
	}
	newer := func(date string) bool {
		return c.Dreamed == "" || !dreamedOK || date > c.Dreamed
	}
	add := func(kind, date, path string, line int, text string) {
		if !newer(date) {
			return
		}
		c.Records = append(c.Records, Record{Kind: kind, Date: date, File: rel(dir, path), Line: line, Text: text})
	}

	// The log and its rotated parts; the archive is read leniently, since it
	// may hold records moved as they are, in no guaranteed format.
	logs := []string{filepath.Join(lead, pack.Log)}
	archive, _ := filepath.Glob(filepath.Join(lead, "log-archive", "*.md"))
	logs = append(logs, archive...)
	for _, path := range logs {
		d, err := meta.Read(path)
		if err != nil {
			if !os.IsNotExist(err) {
				fail(path, err)
			}
			continue
		}
		if filepath.Base(path) == pack.Log {
			entries, bad := d.LogEntries()
			for _, e := range entries {
				add("log", e.Date, path, e.Line, e.Text)
			}
			for _, line := range bad {
				if t, ok := archiveDate(d.Clean[line-1]); ok {
					add("log", t, path, line, short(d.Clean[line-1]))
				}
			}
			continue
		}
		for i, line := range d.Clean {
			if t, ok := archiveDate(line); ok {
				add("archive", t, path, i+1, short(line))
			}
		}
	}

	threads, _ := filepath.Glob(filepath.Join(lead, pack.Threads, "*.md"))
	for _, path := range threads {
		d, err := meta.Read(path)
		if err != nil {
			fail(path, err)
			continue
		}
		updated := d.Front["updated"]
		if !newer(updated) {
			continue
		}
		title := "`" + filepath.Base(path) + "` updated"
		for _, line := range d.Clean[d.BodyStart:] {
			if t, ok := strings.CutPrefix(line, "# "); ok {
				title = strings.TrimSpace(t)
				break
			}
		}
		c.Records = append(c.Records, Record{Kind: "thread", Date: updated, File: rel(dir, path), Line: d.FrontLine["updated"], Text: title})
	}

	if d, err := meta.Read(filepath.Join(lead, pack.Decisions)); err == nil {
		decisions := filepath.Join(lead, pack.Decisions)
		for i := d.BodyStart; i < len(d.Clean); i++ {
			line := d.Clean[i]
			if !strings.HasPrefix(line, "- D") {
				continue
			}
			if t := anyDate.FindString(line); t != "" && newer(t) {
				add("decision", t, decisions, i+1, short(line))
			}
		}
	} else if !os.IsNotExist(err) {
		fail(filepath.Join(lead, pack.Decisions), err)
	}

	if d, err := meta.Read(filepath.Join(lead, pack.Graveyard)); err == nil {
		graveyard := filepath.Join(lead, pack.Graveyard)
		for i := d.BodyStart; i < len(d.Clean); i++ {
			line := d.Clean[i]
			if !strings.HasPrefix(line, "- ") {
				continue
			}
			if t := anyDate.FindString(line); t != "" && newer(t) {
				add("graveyard", t, graveyard, i+1, short(line))
			}
		}
	} else if !os.IsNotExist(err) {
		fail(filepath.Join(lead, pack.Graveyard), err)
	}

	slices.SortFunc(c.Records, func(a, b Record) int {
		return cmp.Or(cmp.Compare(b.Date, a.Date), cmp.Compare(a.File, b.File), cmp.Compare(a.Line, b.Line))
	})
	return c, problems
}

func archiveDate(line string) (string, bool) {
	m := archiveRe.FindStringSubmatch(line)
	if m == nil {
		return "", false
	}
	if _, ok := meta.ParseDate(m[1]); !ok {
		return "", false
	}
	return m[1], true
}

func rel(dir, path string) string {
	r, err := filepath.Rel(dir, path)
	if err != nil {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(r)
}

func short(s string) string {
	if r := []rune(s); len(r) > 90 {
		return string(r[:89]) + "…"
	}
	return s
}

// Report is the digest and the contract, the same in markdown and JSON.
type Report struct {
	Scope      string          `json:"scope"` // project | pack
	Root       string          `json:"root"`
	AsOf       string          `json:"as_of"`
	Mode       string          `json:"mode"` // report | apply
	Dreamed    string          `json:"dreamed,omitempty"`
	Threshold  int             `json:"threshold"`
	Records    []Record        `json:"records"`
	Findings   []check.Finding `json:"findings"`
	Problems   []Problem       `json:"problems"`
	Contract   string          `json:"contract"`
	Incomplete bool            `json:"incomplete"`
}

// ForProject assembles the dream for one project's memory.
func ForProject(pl pack.Place, opt check.Options, apply bool) *Report {
	r := newReport("project", pl.Dir, opt, apply)
	count, problems := CountDir(pl.Dir)
	r.take(count, problems)
	r.Findings = check.Dir(pl.Dir, opt)
	r.finish()
	return r
}

// ForPack assembles the dream for the pack's own memory: its brief,
// decisions and graveyard. Projects keep their own dreams: the findings
// below say which of them rot.
func ForPack(p *pack.Pack, opt check.Options, apply bool) *Report {
	root, err := filepath.Abs(p.Root)
	if err != nil {
		root = p.Root
	}
	r := newReport("pack", root, opt, apply)
	count, problems := CountDir(root)
	r.take(count, problems)
	r.Findings, err = check.Pack(&pack.Pack{Root: root}, opt)
	if err != nil {
		r.Problems = append(r.Problems, Problem{File: ".", Message: err.Error()})
	}
	r.finish()
	return r
}

func newReport(scope, root string, opt check.Options, apply bool) *Report {
	now := opt.Now
	if now.IsZero() {
		now = time.Now()
	}
	mode := "report"
	if apply {
		mode = "apply"
	}
	return &Report{Scope: scope, Root: root, AsOf: now.Format(meta.DateLayout), Mode: mode,
		Records: []Record{}, Findings: []check.Finding{}, Problems: []Problem{}}
}

func (r *Report) take(count Count, problems []Problem) {
	r.Dreamed, r.Threshold, r.Records = count.Dreamed, count.Threshold, count.Records
	r.Problems = append(r.Problems, problems...)
}

func (r *Report) finish() {
	if r.Records == nil {
		r.Records = []Record{}
	}
	if r.Findings == nil {
		r.Findings = []check.Finding{}
	}
	slices.SortStableFunc(r.Findings, func(a, b check.Finding) int {
		return cmp.Or(cmp.Compare(a.Project, b.Project), cmp.Compare(a.File, b.File),
			cmp.Compare(a.Line, b.Line), cmp.Compare(a.Rule, b.Rule), cmp.Compare(a.Message, b.Message))
	})
	r.Incomplete = len(r.Problems) > 0 || check.HasErrors(r.Findings)
	r.Contract = contract(r.Mode == "apply", r.Scope == "pack", r.AsOf)
}

// contract is the instructions the dream agent follows. Only its words
// change with --apply; the CLI writes nothing either way.
func contract(apply, packScope bool, asOf string) string {
	var b strings.Builder
	if !apply {
		b.WriteString(`## The dream contract

You are the dream agent for this memory; nothing has been changed yet. The
digest above is the floor, not the ceiling: read the whole memory — the brief,
the decisions, the threads, the recent log and the selected work plan — and
compare it with reality: git, the claims' ` + "`how`" + ` (run under your own
permissions, asking the human first where a check touches servers or anything
outside), the places the rules say footprints hide.

Consolidate, in the memory only:
- merge what "Now", the threads and the plans repeat; retire what the work has settled;
- a decision reality has outgrown becomes ` + "`superseded → Dn`" + `; lines are never deleted;
- verify the claims you can, and propose their ` + "`seen`" + ` updates;
- date what can rot; fold finished threads into decisions and the log;
- stay inside the budgets: brief 60 lines / 6000 characters, threads and plans 80 / 8000.

Then report to the human what you would change, in at most 15 lines. Change
nothing until they approve.
`)
		if packScope {
			b.WriteString("This is the pack's own memory. A project's memory dreams separately: `omut dream <project>`. Route by the findings above.\n")
		}
		return b.String()
	}
	fmt.Fprintf(&b, `## The dream contract (the human has approved)

Apply the consolidation now, in the memory only: never touch code, and never
write outside %s. Make exactly the changes worth making:
- merge, supersede, date, and set `+"`seen`"+` for the claims you verified;
- keep the budgets; don't take another writer's active thread or rewrite what
  another agent changed today without a handoff;
- when you finish, set `+"`dreamed: %s`"+` in the brief's frontmatter and record
  the dream the way the rules say: a line at the top of the log for a project,
  the pack brief for a pack;
- show the human a short summary of what changed.
`, "`"+pack.MetaDir+"/`", asOf)
	if packScope {
		b.WriteString("This is the pack's own memory; a project's memory dreams separately: `omut dream <project>`.\n")
	}
	return b.String()
}

// Markdown is the digest and the contract for a human or an agent.
func (r *Report) Markdown() string {
	var b strings.Builder
	fmt.Fprintf(&b, "# omut dream: %s (%s)\n\n", filepath.Base(r.Root), r.Mode)
	switch {
	case r.Dreamed == "":
		fmt.Fprintf(&b, "Memory: %d records; no dream is recorded yet (no `dreamed:` in the brief). Threshold: %d records (`dream:` in the brief, %d when unset).\n", r.Total(), r.Threshold, DefaultThreshold)
	default:
		fmt.Fprintf(&b, "Memory: %d records since the last dream (%s). Threshold: %d records (`dream:` in the brief, %d when unset).\n", r.Total(), r.Dreamed, r.Threshold, DefaultThreshold)
	}
	b.WriteString("Counted: dated log entries (log.md and log-archive/), thread updates and dated decisions strictly after the last dream.\n\n")
	if r.Total() == 0 {
		b.WriteString("No records after the last dream; the findings below are the digest.\n")
	} else {
		fmt.Fprintf(&b, "## Records since the last dream (newest first; %d of %d shown)\n\n", min(shownInText, r.Total()), r.Total())
		for _, rec := range r.Records[:min(shownInText, len(r.Records))] {
			fmt.Fprintf(&b, "- %s · %s · `%s:%d` · %s\n", rec.Date, rec.Kind, rec.File, rec.Line, rec.Text)
		}
	}
	b.WriteString("\n## What check found\n\n")
	if len(r.Findings) == 0 {
		b.WriteString("Nothing.\n")
	}
	for _, f := range r.Findings {
		loc := f.File
		if f.Line > 0 {
			loc = fmt.Sprintf("%s:%d", f.File, f.Line)
		}
		fmt.Fprintf(&b, "- %s %s: %s (`%s`)\n", f.Level, f.Rule, f.Message, loc)
	}
	b.WriteString("\n" + r.Contract)
	if len(r.Problems) > 0 {
		b.WriteString("\n## Incomplete\n\nRead failures below mean the count may miss records; missing data does not mean nothing is pending.\n\n")
		for _, p := range r.Problems {
			fmt.Fprintf(&b, "- `%s`: %s\n", p.File, p.Message)
		}
	}
	fmt.Fprintf(&b, "\nAs of %s. The CLI read all of this; it executes nothing and writes nothing.\n", r.AsOf)
	return b.String()
}

// Total is the record count for the digest line.
func (r *Report) Total() int { return len(r.Records) }
