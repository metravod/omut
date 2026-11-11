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
	"github.com/metravod/omut/internal/context"
	"github.com/metravod/omut/internal/dream"
	"github.com/metravod/omut/internal/meta"
	"github.com/metravod/omut/internal/pack"
)

// Start is the context an agent gets when a session starts in dir: the
// brief, what waits for a decision, the unfinished threads and what check
// says. "" when dir is not under any meta.
func Start(dir string) string {
	pl := pack.Locate(dir)
	switch {
	case pl.Dir == "":
		return ""
	case pack.IsRoot(pl.Dir):
		return startPack(pl.Dir)
	}
	return startProject(pl)
}

func startPack(root string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# omut: the pack %s\n\n", root)
	b.WriteString("You were started in a pack root, so you are the lead: read `.omut/LEAD.md` before anything else.\n")
	b.WriteString("The pack overview is assembled below; do not reread those sources just to assemble it again. Use `omut context <project>` when you need the full current project records.\n\n")
	b.WriteString(context.ForPack(&pack.Pack{Root: root}, check.Options{}).Markdown())
	writeDreamReminder(&b, root)
	return b.String()
}

// writeDreamReminder adds one line when the memory grew past the brief's
// threshold since the last dream: consolidation is due by volume, not by the
// calendar, and the nag extinguishes when the dream agent writes dreamed:.
// Nothing for a frozen project, whose memory is not growing.
func writeDreamReminder(b *strings.Builder, dir string) {
	if brief, err := meta.Read(filepath.Join(dir, pack.MetaDir, pack.Brief)); err != nil || brief.Front["status"] == "frozen" {
		return
	}
	count, _ := dream.CountDir(dir)
	if !count.Due() {
		return
	}
	since := "the memory has never been recorded as dreamed"
	if count.Dreamed != "" {
		since = fmt.Sprintf("the last dream was recorded %s", count.Dreamed)
	}
	fmt.Fprintf(b, "\n## Memory wants a dream\n\n%d records are newer than %s; the threshold is %d (`dream:` in the brief, %d when unset). Tell the human the memory wants consolidating, and offer it; on their approval run `omut dream` and follow its contract.\n",
		count.Total(), since, count.Threshold, dream.DefaultThreshold)
}

func startProject(pl pack.Place) string {
	dir := pl.Dir
	lead := filepath.Join(dir, pack.MetaDir)
	var b strings.Builder
	fmt.Fprintf(&b, "# omut: the meta of %s\n\n", filepath.Base(dir))
	b.WriteString("This project keeps its meta in `.omut/` (rules: `.omut/README.md`). Below is what to read at the start. When a piece of work is done, record it before you tell the human it is done: a log line, \"Now\" in the brief, a thread for what is left. You won't see the session end, so don't wait for it.\n")

	writeWorktree(&b, pl)
	if pack.Private(dir) {
		b.WriteString("\nThe meta is this copy's only: git ignores `.omut/` (`omut start --private`), and it stays out of commits. Don't `git add -f` it.\n")
	}
	// Stable first, volatile last: what the memory holds changes rarely and
	// legitimately; what this run observes (mtimes, findings, counts) would
	// break an inference prefix cache if it came first.
	if text, err := os.ReadFile(filepath.Join(lead, pack.Brief)); err == nil {
		b.WriteString("\n## .omut/brief.md\n\n")
		b.WriteString(strings.TrimRight(string(text), "\n") + "\n")
		writePlan(&b, dir, meta.Parse(filepath.Join(lead, pack.Brief), string(text)), false)
	}
	writeInstance(&b, filepath.Join(pl.Instance, pack.MetaDir))

	var waiting []string
	if d, err := meta.Read(filepath.Join(lead, pack.Decisions)); err == nil {
		decs, _ := d.Decisions()
		for _, dec := range decs {
			if dec.Status == "open" || dec.Status == "doubted" {
				waiting = append(waiting, "- "+strings.TrimSpace(d.Lines[dec.Line-1][2:]))
			}
		}
	}
	if brief, err := meta.Read(filepath.Join(lead, pack.Brief)); err == nil && brief.Front["status"] != "frozen" {
		adrs, _ := check.ADRs(dir, brief)
		for _, a := range adrs {
			if w := a.Waiting(); w != "" {
				rel, _ := filepath.Rel(dir, a.Path)
				waiting = append(waiting, fmt.Sprintf("- `%s` [%s] %s", filepath.ToSlash(rel), w, a.Title))
			}
		}
	}
	if len(waiting) > 0 {
		b.WriteString("\n## Decisions waiting for the human (.omut/decisions.md, ADRs)\n\n")
		b.WriteString(strings.Join(waiting, "\n") + "\n")
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
	work := dir
	if pl.Worktree != "" {
		work = pl.Worktree
	}
	writeUnrecorded(&b, work, dir)
	writeFindings(&b, check.Dir(dir, check.Options{}), false)
	writeDue(&b, dir, pl.Instance)
	writeDreamReminder(&b, dir)
	return b.String()
}

// writeWorktree tells an agent in a linked worktree where the clone's memory
// is (D17): relative paths would write a second one, gone with the worktree.
func writeWorktree(b *strings.Builder, pl pack.Place) {
	switch {
	case pl.Worktree != "":
		fmt.Fprintf(b, "\nYou are in a linked git worktree, and a clone keeps one meta: it is in the main checkout, `%s`. Read and write it there by that full path; the paths below are relative to `%s`. Don't create `.omut/` in this worktree.\n", filepath.Join(pl.Dir, pack.MetaDir), pl.Dir)
	case pl.Instance != pl.Dir:
		fmt.Fprintf(b, "\nYou are in a linked git worktree. `.omut/` here follows this branch, but this copy's memory is not here: it is `%s`, in the main checkout. Write what is true only for this copy there by that full path; don't create `.omut/local/` in this worktree.\n", filepath.Join(pl.Instance, pack.MetaDir, pack.Local))
	}
}

// writeUnrecorded tells about work newer than the last write to the log
// that no commit holds: a session that ended without recording it, or this
// session's own work before a resume or a compaction. Without git it is
// the only signal that a piece of work ended unrecorded.
func writeUnrecorded(b *strings.Builder, work, dir string) {
	logInfo, err := os.Stat(filepath.Join(dir, pack.MetaDir, pack.Log))
	if err != nil {
		return
	}
	when, what := check.NewestUnsaved(work)
	if what == "" || !when.After(logInfo.ModTime()) {
		return
	}
	b.WriteString("\n## Unrecorded work\n\n")
	fmt.Fprintf(b, "`%s` changed at %s, after the last write to `.omut/log.md` (%s), and no commit holds it. Either a session ended without recording its work, or this is your own session's work before a resume. Find out what was done (the files, `git diff` if there is git) and record it before going on; if it is unclear whose work it is, ask the human.\n",
		what, when.Format(timeLayout), logInfo.ModTime().Format(timeLayout))
}

const timeLayout = "2006-01-02 15:04"

// dueClaims is how many claims an agent is asked to check per session.
const dueClaims = 2

// writeDue asks the agent to check the claims about where the project
// lives that were seen longest ago (D11): omut remembers how to check
// them and when it was done, the agent does it.
func writeDue(b *strings.Builder, dir, instance string) {
	due, unchecked := check.Due(dir, instance, dueClaims, check.Options{})
	if len(due) == 0 && unchecked == 0 {
		return
	}
	b.WriteString("\n## Check while you are here\n\n")
	if len(due) > 0 {
		b.WriteString("Claims about where the project lives, the longest unchecked first; the lines with their `how` are in the memory above. Run each `how` (ask the human first where it touches servers or anything outside this directory), compare, and set `seen` to today if it holds. Note what you saw, not just \"checked\", in `.omut/local/log.md`.\n")
		for _, c := range due {
			seen := "never"
			if c.Seen != "" {
				seen = c.Seen
			}
			fmt.Fprintf(b, "- `%s:%d` · seen %s\n", c.File, c.Line, seen)
		}
	}
	if unchecked > 0 {
		what := fmt.Sprintf("%d claims about where the project lives say no way to check them", unchecked)
		if unchecked == 1 {
			what = "1 claim about where the project lives says no way to check it"
		}
		fmt.Fprintf(b, "%s. When you know how, add ` · how: <command or place to look>` to the line.\n", what)
	}
}

// writeInstance adds the instance layer (D15) when it says anything: this
// copy's footprint with its claims, and a pointer to the rest. Access and
// Runbook serve deploy-type sessions; every session pays only the pointer.
func writeInstance(b *strings.Builder, lead string) {
	local := filepath.Join(lead, pack.Local)
	instance, err := meta.Read(filepath.Join(local, pack.Instance))
	if err != nil {
		return
	}
	live, hasLive := instance.Section("Where it lives")
	if (!hasLive || !instance.Filled(live)) && instance.Front["plan"] == "" {
		return
	}
	b.WriteString("\n## .omut/local/instance.md (this copy only, never in git)\n\n")
	if instance.HasFront {
		b.WriteString(strings.TrimRight(strings.Join(instance.Lines[:instance.BodyStart], "\n"), "\n") + "\n")
	}
	if hasLive && instance.Filled(live) {
		b.WriteString(strings.TrimRight(strings.Join(instance.Lines[live.Line-1:live.End], "\n"), "\n") + "\n")
	}
	if access, ok := instance.Section("Access"); ok && instance.Filled(access) {
		b.WriteString("\nAccess (secrets, accounts) stays in that file; read it when this session touches them.\n")
	}
	if runbook, ok := instance.Section("Runbook"); ok && instance.Filled(runbook) {
		b.WriteString("\nRunbook (deploy, roll back, restart, move) stays in that file; read it when this session deploys or moves anything.\n")
	}
	writePlan(b, filepath.Dir(lead), instance, true)
	if d, err := meta.Read(filepath.Join(local, pack.Log)); err == nil {
		entries, _ := d.LogEntries()
		if len(entries) > 3 {
			entries = entries[:3]
		}
		if len(entries) > 0 {
			b.WriteString("\n## This copy's latest events (.omut/local/log.md)\n\n")
			for _, e := range entries {
				b.WriteString(d.Lines[e.Line-1] + "\n")
			}
		}
	}
}

func writePlan(b *strings.Builder, dir string, owner *meta.Doc, local bool) {
	d, err := pack.ReadPlan(dir, owner, local)
	if err != nil {
		fmt.Fprintf(b, "\n## Incomplete plan context\n\n`%s:%d`: %v\n", owner.Path, owner.FrontLine["plan"], err)
		return
	}
	if d == nil {
		return
	}
	// A plan that is parked or dropped is not this session's work: one
	// pointer line instead of the body. No frontmatter means active.
	switch d.Front["status"] {
	case "parked", "dropped":
		fmt.Fprintf(b, "\n## Work plan — `%s:1` (%s)\n\nNot this session's work; the full index is in the file and in `omut context`.\n", d.Path, d.Front["status"])
	default:
		fmt.Fprintf(b, "\n## Work plan — `%s:1`\n\n%s\n", d.Path, strings.Join(d.Lines, "\n"))
	}
}

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
	// In a linked worktree without meta of its own, the clone's is the main
	// checkout's (D17).
	metaRoot := root
	if pl := pack.Locate(root); pl.Worktree != "" {
		metaRoot = pl.Dir
	}
	lead := filepath.Join(metaRoot, pack.MetaDir)
	if _, err := os.Stat(filepath.Join(lead, pack.Brief)); err != nil || pack.IsRoot(metaRoot) {
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
	for _, f := range check.Dir(metaRoot, check.Options{}) {
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
