// Package hook is what agents' and git's hooks call: a session-start
// context and a pre-commit reminder. It only reads the meta; a directory
// without .omut/ gets silence, so a hook can be installed globally.
package hook

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/metravod/omut/internal/check"
	"github.com/metravod/omut/internal/meta"
	"github.com/metravod/omut/internal/pack"
)

// Locate walks up from dir to the nearest directory with .omut/. It is a
// pack root when .omut/LEAD.md is there. root is "" when there is none.
func Locate(dir string) (root string, isPack bool) {
	dir, err := filepath.Abs(dir)
	if err != nil {
		return "", false
	}
	for {
		if info, err := os.Stat(filepath.Join(dir, pack.MetaDir)); err == nil && info.IsDir() {
			return dir, pack.IsRoot(dir)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}

// Start is the context an agent gets when a session starts in dir: the
// brief, what waits for a decision, the unfinished threads and what check
// says. "" when dir is not under any meta.
func Start(dir string) string {
	root, isPack := Locate(dir)
	switch {
	case root == "":
		return ""
	case isPack:
		return startPack(root)
	}
	return startProject(root)
}

func startPack(root string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# omut: the pack %s\n\n", root)
	b.WriteString("You were started in a pack root, so you are the lead: read `.omut/LEAD.md` before anything else.\n")
	p := &pack.Pack{Root: root}
	if fs, err := check.Pack(p, check.Options{}); err == nil {
		writeFindings(&b, fs, true)
	}
	return b.String()
}

func startProject(dir string) string {
	lead := filepath.Join(dir, pack.MetaDir)
	var b strings.Builder
	fmt.Fprintf(&b, "# omut: the meta of %s\n\n", filepath.Base(dir))
	b.WriteString("This project keeps its meta in `.omut/` (rules: `.omut/README.md`). Below is what to read at the start. When a piece of work is done, record it before you tell the human it is done: a line at the top of `.omut/log.md`, \"Now\" in `.omut/brief.md`, a thread for what is left. You won't see the session end, so don't wait for it.\n")

	writeUnrecorded(&b, dir)
	writeFindings(&b, check.Dir(dir, check.Options{}), false)

	if text, err := os.ReadFile(filepath.Join(lead, pack.Brief)); err == nil {
		b.WriteString("\n## .omut/brief.md\n\n")
		b.WriteString(strings.TrimRight(string(text), "\n") + "\n")
	}
	if d, err := meta.Read(filepath.Join(lead, pack.Decisions)); err == nil {
		decs, _ := d.Decisions()
		var waiting []string
		for _, dec := range decs {
			if dec.Status == "open" || dec.Status == "doubted" {
				waiting = append(waiting, "- "+strings.TrimSpace(d.Lines[dec.Line-1][2:]))
			}
		}
		if len(waiting) > 0 {
			b.WriteString("\n## Decisions waiting for the human (.omut/decisions.md)\n\n")
			b.WriteString(strings.Join(waiting, "\n") + "\n")
		}
	}
	threads, _ := filepath.Glob(filepath.Join(lead, pack.Threads, "*.md"))
	var open []string
	for _, path := range threads {
		d, err := meta.Read(path)
		if err != nil {
			continue
		}
		status := d.Front["status"]
		if status != "active" && status != "parked" {
			continue
		}
		line := fmt.Sprintf("- `.omut/threads/%s` (%s", filepath.Base(path), status)
		if why := d.Front["parked_because"]; status == "parked" && why != "" {
			line += ": " + why
		}
		line += ")"
		for _, l := range d.Clean {
			if _, next, ok := strings.Cut(l, "**Next step:**"); ok {
				line += " next step: " + strings.TrimSpace(next)
				break
			}
		}
		open = append(open, line)
	}
	if len(open) > 0 {
		b.WriteString("\n## Unfinished threads\n\n")
		b.WriteString(strings.Join(open, "\n") + "\n")
	}
	return b.String()
}

// writeUnrecorded tells about work newer than the last write to the log
// that no commit holds: a session that ended without recording it, or this
// session's own work before a resume or a compaction. Without git it is
// the only signal that a piece of work ended unrecorded.
func writeUnrecorded(b *strings.Builder, dir string) {
	logInfo, err := os.Stat(filepath.Join(dir, pack.MetaDir, pack.Log))
	if err != nil {
		return
	}
	when, what := check.NewestUnsaved(dir)
	if what == "" || !when.After(logInfo.ModTime()) {
		return
	}
	b.WriteString("\n## Unrecorded work\n\n")
	fmt.Fprintf(b, "`%s` changed at %s, after the last write to `.omut/log.md` (%s), and no commit holds it. Either a session ended without recording its work, or this is your own session's work before a resume. Find out what was done (the files, `git diff` if there is git) and record it before going on; if it is unclear whose work it is, ask the human.\n",
		what, when.Format(timeLayout), logInfo.ModTime().Format(timeLayout))
}

const timeLayout = "2006-01-02 15:04"

// writeFindings lists errors and warnings; open decisions are listed
// separately for a project, and only counted for a pack.
func writeFindings(b *strings.Builder, fs []check.Finding, withInfo bool) {
	var lines []string
	info := 0
	for _, f := range fs {
		if f.Level == check.Info {
			info++
			if !withInfo {
				continue
			}
		}
		loc := f.File
		if f.Line > 0 {
			loc = fmt.Sprintf("%s:%d", f.File, f.Line)
		}
		line := fmt.Sprintf("- %s %s: %s (%s)", f.Level, f.Rule, f.Message, loc)
		if f.Rule == "stale-brief" {
			line += ". A session may have ended without recording its work: compare with git and ask the human before trusting \"Now\""
		}
		lines = append(lines, line)
	}
	if len(lines) == 0 {
		return
	}
	b.WriteString("\n## omut check\n\n")
	b.WriteString(strings.Join(lines, "\n") + "\n")
}

// CommitResult is what the pre-commit hook says. Block means the commit
// should not go through.
type CommitResult struct {
	Message string
	Block   bool
}

// Commit looks at a commit about to be made in the repository at dir: shape
// errors in staged meta block it; work staged without a log entry written
// after it is reminded of, and blocked when strict.
func Commit(dir string, strict bool) CommitResult {
	out, err := exec.Command("git", "-C", dir, "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return CommitResult{}
	}
	root := strings.TrimSpace(string(out))
	lead := filepath.Join(root, pack.MetaDir)
	if _, err := os.Stat(filepath.Join(lead, pack.Brief)); err != nil || pack.IsRoot(root) {
		return CommitResult{}
	}
	out, err = exec.Command("git", "-C", root, "diff", "--cached", "--name-only", "-z", "--diff-filter=ACMR").Output()
	if err != nil {
		return CommitResult{}
	}
	staged := slices.DeleteFunc(strings.Split(string(out), "\x00"), func(s string) bool { return s == "" })
	isMeta := func(rel string) bool { return rel == pack.MetaDir || strings.HasPrefix(rel, pack.MetaDir+"/") }

	var b strings.Builder
	res := CommitResult{}
	var errs []check.Finding
	for _, f := range check.Dir(root, check.Options{}) {
		if f.Level == check.Error {
			errs = append(errs, f)
		}
	}
	if len(errs) > 0 {
		if slices.ContainsFunc(staged, isMeta) {
			res.Block = true
			b.WriteString("omut: the meta in this commit has shape errors, fix them first (omut check):\n")
		} else {
			b.WriteString("omut: the meta has shape errors (omut check):\n")
		}
		for _, f := range errs {
			fmt.Fprintf(&b, "  %s:%d  %s\n", f.File, f.Line, f.Message)
		}
	}

	var newest time.Time
	what := ""
	for _, rel := range staged {
		if isMeta(rel) {
			continue
		}
		if info, err := os.Lstat(filepath.Join(root, rel)); err == nil && info.ModTime().After(newest) {
			newest, what = info.ModTime(), rel
		}
	}
	logInfo, err := os.Stat(filepath.Join(lead, pack.Log))
	if what != "" && (err != nil || newest.After(logInfo.ModTime())) {
		if strict {
			res.Block = true
		}
		fmt.Fprintf(&b, "omut: %s changed after the last write to .omut/log.md. If this commit finishes a piece of work, add a line to the top of the log (or extend today's), update \"Now\" in .omut/brief.md, and record unfinished work as a thread.\n", what)
		if strict {
			b.WriteString("This repository asks for that before a commit (omut hook commit --strict); git commit --no-verify skips the check.\n")
		}
	}
	res.Message = b.String()
	return res
}
