package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/metravod/omut/internal/check"
	"github.com/metravod/omut/templates"
)

// omut runs one command in dir and returns its output and error.
func omut(t *testing.T, dir string, args ...string) (string, error) {
	t.Helper()
	t.Chdir(dir)
	var out bytes.Buffer
	err := run(args, &out)
	return out.String(), err
}

func must(t *testing.T, dir string, args ...string) string {
	t.Helper()
	out, err := omut(t, dir, args...)
	if err != nil {
		t.Fatalf("omut %v: %v\n%s", args, err, out)
	}
	return out
}

func findings(t *testing.T, dir string, args ...string) []check.Finding {
	t.Helper()
	out, err := omut(t, dir, append([]string{"check", "--json"}, args...)...)
	var exit exitCode
	if err != nil && !errors.As(err, &exit) {
		t.Fatalf("check: %v", err)
	}
	var fs []check.Finding
	if err := json.Unmarshal([]byte(out), &fs); err != nil {
		t.Fatalf("check --json: %v\n%s", err, out)
	}
	return fs
}

func has(fs []check.Finding, project, rule string) bool {
	for _, f := range fs {
		if f.Project == project && f.Rule == rule {
			return true
		}
	}
	return false
}

func write(t *testing.T, path, text string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
}

func read(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

const filledBrief = `---
status: active
updated: %s
verified: %s
by: codex/gpt-5
---
## Why
An HN parser for a week.

## What it is
A Python script → docs/notes.md

## Where it lives
%s

## Now
Works.
**Verified:** a run
**Not verified:** nothing

## Next
- turn it off
`

func fill(updated, verified, lives string) string {
	s := filledBrief
	for _, v := range []string{updated, verified, lives} {
		s = strings.Replace(s, "%s", v, 1)
	}
	return s
}

func TestInitIsCleanAndRefusesTwice(t *testing.T) {
	root := t.TempDir()
	must(t, root, "init", "--by", "test")
	for _, f := range []string{"AGENTS.md", ".omut/LEAD.md", ".omut/brief.md", ".omut/decisions.md", ".omut/graveyard.md"} {
		if _, err := os.Stat(filepath.Join(root, f)); err != nil {
			t.Errorf("init did not create %s", f)
		}
	}
	if target, err := os.Readlink(filepath.Join(root, "CLAUDE.md")); err != nil || target != "AGENTS.md" {
		t.Errorf("CLAUDE.md → %q, %v", target, err)
	}
	if fs := findings(t, root); len(fs) != 0 {
		t.Errorf("fresh pack has findings: %+v", fs)
	}
	if _, err := omut(t, root, "init"); err == nil || !strings.Contains(err.Error(), "zero") {
		t.Errorf("second init: %v, want refusal pointing at zero", err)
	}
}

func TestStartAdoptsWithoutTouchingFiles(t *testing.T) {
	root := t.TempDir()
	must(t, root, "init")
	proj := filepath.Join(root, "site")
	write(t, filepath.Join(proj, "index.html"), "<h1>hi</h1>\n")
	write(t, filepath.Join(proj, "AGENTS.md"), "# Site\n\nOwn rules.\n")
	write(t, filepath.Join(proj, "CLAUDE.md"), "# old rules for Claude only\n")
	write(t, filepath.Join(root, "untracked", "x.txt"), "x")

	fs := findings(t, root)
	if !has(fs, "untracked", "no-meta") {
		t.Errorf("no-meta not reported: %+v", fs)
	}

	must(t, root, "start", "site", "--by", "codex/gpt-5")
	agents := read(t, filepath.Join(proj, "AGENTS.md"))
	if !strings.HasPrefix(agents, "# Site\n\nOwn rules.\n\n<!-- omut:start -->") {
		t.Errorf("AGENTS.md after start:\n%s", agents)
	}
	if read(t, filepath.Join(proj, "CLAUDE.md")) != "# old rules for Claude only\n" {
		t.Error("start rewrote the user's CLAUDE.md")
	}
	if !strings.Contains(read(t, filepath.Join(proj, ".omut/brief.md")), "by: codex/gpt-5") {
		t.Error("--by did not reach brief.md")
	}

	fs = findings(t, root, "site")
	for _, rule := range []string{"pointer", "unfilled", "unverified"} {
		if !has(fs, "site", rule) {
			t.Errorf("fresh template: %s not reported: %+v", rule, fs)
		}
	}
	if check.HasErrors(fs) {
		t.Errorf("a fresh template must not be a form error: %+v", fs)
	}
	if out := must(t, root, "start", "site"); !strings.Contains(out, "in the pack already") {
		t.Errorf("second start: %q", out)
	}
	// Paths work as names, from inside the project too.
	if out := must(t, proj, "start", "."); !strings.Contains(out, "in the pack already") {
		t.Errorf("start . from inside: %q", out)
	}
}

func TestStaleAndFrozenButLive(t *testing.T) {
	root := t.TempDir()
	must(t, root, "init")
	must(t, root, "start", "hn")
	proj := filepath.Join(root, "hn")
	today := time.Now().Format("2006-01-02")
	write(t, filepath.Join(proj, ".omut/brief.md"), fill("2020-01-01", today, "- a VPS, cron since 2026-06-01"))
	write(t, filepath.Join(proj, "docs/notes.md"), "notes\n")
	write(t, filepath.Join(proj, "main.py"), "print(1)\n")

	fs := findings(t, root, "hn")
	if !has(fs, "hn", "stale-brief") {
		t.Errorf("stale-brief not reported: %+v", fs)
	}
	if check.HasErrors(fs) {
		t.Errorf("unexpected errors: %+v", fs)
	}

	write(t, filepath.Join(proj, ".omut/brief.md"), fill(today, today, "- a VPS, cron since 2026-06-01"))
	out := must(t, root, "freeze", "hn")
	if !strings.Contains(out, "Where it lives") {
		t.Errorf("freeze did not warn about a live footprint: %q", out)
	}
	fs = findings(t, root, "hn")
	if !has(fs, "hn", "frozen-but-live") || has(fs, "hn", "stale-brief") {
		t.Errorf("after freeze: %+v", fs)
	}
	if _, err := omut(t, root, "start", "hn"); err == nil || !strings.Contains(err.Error(), "thaw") {
		t.Errorf("start on frozen: %v", err)
	}
	must(t, root, "thaw", "hn")
	if !strings.Contains(read(t, filepath.Join(proj, ".omut/brief.md")), "status: active") {
		t.Error("thaw did not set active")
	}
}

func TestShapeErrors(t *testing.T) {
	root := t.TempDir()
	must(t, root, "init")
	must(t, root, "start", "p")
	proj := filepath.Join(root, "p")
	today := time.Now().Format("2006-01-02")
	write(t, filepath.Join(proj, ".omut/brief.md"), fill(today, today, ""))
	write(t, filepath.Join(proj, ".omut/decisions.md"), "# Decisions\n\n- D1 [accepted] a → threads/gone.md\n- D1 [open] a duplicate\n- D2 [superseded → D9] no D9\n")
	write(t, filepath.Join(proj, ".omut/threads/x.md"), "---\nstatus: parked\nupdated: "+today+"\nby: me\nparked_because: tired\n---\n# X\n**Next step:** y\n")
	write(t, filepath.Join(proj, ".omut/log.md"), "- 2026-10-01 · me · old\n- 2026-10-31 · me · new below old\n")

	fs := findings(t, root, "p")
	var msgs []string
	for _, f := range fs {
		if f.Rule == "shape" {
			msgs = append(msgs, f.Message)
		}
	}
	joined := strings.Join(msgs, "\n")
	for _, want := range []string{"broken link: threads/gone.md", "D1 is already", "superseded by D9", "parked_because \"tired\"", "newest entries go on top"} {
		if !strings.Contains(joined, want) {
			t.Errorf("missing shape finding %q in:\n%s", want, joined)
		}
	}
	if _, err := omut(t, root, "check", "p"); !errors.As(err, new(exitCode)) {
		t.Errorf("check with errors must exit non-zero, got %v", err)
	}
}

func TestZeroRestoresToolFilesAndKeepsMeta(t *testing.T) {
	root := t.TempDir()
	must(t, root, "init")
	write(t, filepath.Join(root, ".omut/LEAD.md"), "broken\n")
	write(t, filepath.Join(root, ".omut/brief.md"), "---\nupdated: 2026-10-31\nby: me\nignore: []\n---\n## Focus\nmine\n")
	os.Remove(filepath.Join(root, ".omut/graveyard.md"))
	write(t, filepath.Join(root, ".omut/conventions.md"), "<!-- Tool file: an old version -->\n")
	write(t, filepath.Join(root, "AGENTS.md"), "# My pack\n\n<!-- omut:start -->\nold block\n<!-- omut:end -->\n\ntail\n")

	fs := findings(t, root)
	if !has(fs, "", "outdated") || !has(fs, "", "scaffold") {
		t.Errorf("before zero: %+v", fs)
	}
	if out := must(t, root, "zero"); !strings.Contains(out, "removed  .omut/conventions.md") {
		t.Errorf("zero did not retire conventions.md: %q", out)
	}
	if strings.Contains(read(t, filepath.Join(root, ".omut/LEAD.md")), "broken") {
		t.Error("zero did not restore LEAD.md")
	}
	if !strings.Contains(read(t, filepath.Join(root, ".omut/brief.md")), "mine") {
		t.Error("zero overwrote the pack brief")
	}
	agents := read(t, filepath.Join(root, "AGENTS.md"))
	if strings.Contains(agents, "old block") || !strings.HasPrefix(agents, "# My pack\n\n") || !strings.HasSuffix(agents, "\n\ntail\n") {
		t.Errorf("block not replaced in place:\n%s", agents)
	}
	if out := must(t, root, "zero"); !strings.Contains(out, "The scaffold is fine") {
		t.Errorf("second zero: %q", out)
	}
}

func git(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir, "-c", "user.name=t", "-c", "user.email=t@t", "-c", "commit.gpgsign=false"}, args...)...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func TestKillGuardsTombstoneAndGhosts(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("no git")
	}
	root := t.TempDir()
	must(t, root, "init")
	must(t, root, "start", "hn")
	must(t, root, "start", "site")
	today := time.Now().Format("2006-01-02")
	write(t, filepath.Join(root, "hn/.omut/brief.md"), fill(today, today, ""))
	write(t, filepath.Join(root, "hn/docs/notes.md"), "n\n")
	write(t, filepath.Join(root, "site/.omut/brief.md"), strings.Replace(fill(today, today, ""), "- turn it off", "- take the data from hn", 1))
	if _, err := omut(t, root, "kill", "hn", "--yes", "--reason", "x"); err == nil || !strings.Contains(err.Error(), "no git") {
		t.Errorf("kill of a project without git: %v", err)
	}
	git(t, filepath.Join(root, "hn"), "init", "-q")

	if _, err := omut(t, root, "kill", "hn", "--reason", "x"); err == nil || !strings.Contains(err.Error(), "--yes") {
		t.Errorf("kill without --yes: %v", err)
	}
	if _, err := omut(t, root, "kill", "hn", "--yes"); err == nil || !strings.Contains(err.Error(), "--reason") {
		t.Errorf("kill without --reason: %v", err)
	}
	if _, err := omut(t, root, "kill", "--yes", "hn", "--reason", "x"); err == nil || !strings.Contains(err.Error(), "uncommitted") {
		t.Errorf("kill of a dirty repo: %v", err)
	}
	git(t, filepath.Join(root, "hn"), "add", "-A")
	git(t, filepath.Join(root, "hn"), "commit", "-qm", "init")
	if _, err := omut(t, root, "kill", "hn", "--yes", "--reason", "x"); err == nil || !strings.Contains(err.Error(), "remote") {
		t.Errorf("kill of an unpushed repo: %v", err)
	}
	out := must(t, root, "kill", "hn", "--yes", "--force", "--reason", "the week is over")
	if !strings.Contains(out, "site/.omut/brief.md") {
		t.Errorf("kill did not list the mention:\n%s", out)
	}
	if _, err := os.Stat(filepath.Join(root, "hn")); err == nil {
		t.Error("project still there")
	}
	tomb := read(t, filepath.Join(root, ".omut/graveyard.md"))
	if !strings.Contains(tomb, "- hn — removed "+today+" · the week is over · An HN parser for a week.") {
		t.Errorf("tombstone:\n%s", tomb)
	}
	if fs := findings(t, root); !has(fs, "site", "ghost") {
		t.Errorf("ghost not reported: %+v", fs)
	}
}

func TestStandaloneProject(t *testing.T) {
	proj := t.TempDir()
	today := time.Now().Format("2006-01-02")
	write(t, filepath.Join(proj, ".omut/brief.md"), fill(today, today, ""))
	write(t, filepath.Join(proj, ".omut/decisions.md"), "# Decisions\n")
	write(t, filepath.Join(proj, ".omut/log.md"), "")
	write(t, filepath.Join(proj, "docs/notes.md"), "n\n")
	write(t, filepath.Join(proj, "AGENTS.md"), "<!-- omut:start -->\nx\n<!-- omut:end -->\n")
	writeReadme(t, proj)
	if fs := findings(t, proj); len(fs) != 0 {
		t.Errorf("standalone project: %+v", fs)
	}
}

// standalone lays out a filled project with the given brief date that
// check accepts as clean.
func standalone(t *testing.T, dir, updated string) {
	t.Helper()
	today := time.Now().Format("2006-01-02")
	write(t, filepath.Join(dir, ".omut/brief.md"), fill(updated, today, ""))
	write(t, filepath.Join(dir, ".omut/decisions.md"), "# Decisions\n")
	write(t, filepath.Join(dir, ".omut/log.md"), "")
	write(t, filepath.Join(dir, "docs/notes.md"), "n\n")
	write(t, filepath.Join(dir, "AGENTS.md"), "<!-- omut:start -->\nx\n<!-- omut:end -->\n")
	writeReadme(t, dir)
}

func writeReadme(t *testing.T, dir string) {
	t.Helper()
	b, err := templates.FS.ReadFile("project/omut/README.md")
	if err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(dir, ".omut/README.md"), string(b))
}

func TestStaleBriefInsideSymlinkedProject(t *testing.T) {
	root := t.TempDir()
	must(t, root, "init")
	outside := filepath.Join(t.TempDir(), "hn")
	today := time.Now().Format("2006-01-02")
	standalone(t, outside, today)
	if err := os.Symlink(outside, filepath.Join(root, "hn")); err != nil {
		t.Fatal(err)
	}
	if fs := findings(t, root, "hn"); has(fs, "hn", "stale-brief") {
		t.Fatalf("stale-brief before any edit: %+v", fs)
	}
	// An edit inside the linked directory, newer than the brief; the link
	// itself stays as old as it was.
	edit := filepath.Join(outside, "main.py")
	write(t, edit, "print(1)\n")
	later := time.Now().Add(48 * time.Hour)
	if err := os.Chtimes(edit, later, later); err != nil {
		t.Fatal(err)
	}
	if fs := findings(t, root, "hn"); !has(fs, "hn", "stale-brief") {
		t.Errorf("stale-brief not reported for an edit inside a symlinked project: %+v", fs)
	}
}

func TestStaleBriefWithGit(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("no git")
	}
	src := filepath.Join(t.TempDir(), "hn")
	standalone(t, src, "2021-01-01")
	git(t, src, "init", "-q")
	git(t, src, "add", ".")
	t.Setenv("GIT_COMMITTER_DATE", "2020-06-01T12:00:00Z")
	git(t, src, "commit", "-q", "-m", "old work")
	os.Unsetenv("GIT_COMMITTER_DATE")

	// A fresh clone: every mtime is today, but nothing changed since 2020.
	clone := filepath.Join(t.TempDir(), "hn")
	git(t, filepath.Dir(clone), "clone", "-q", src, clone)
	if fs := findings(t, clone); has(fs, "hn", "stale-brief") {
		t.Errorf("stale-brief on a fresh clone: %+v", fs)
	}
	// A commit that touches only the meta is not a change of the project.
	write(t, filepath.Join(clone, ".omut/log.md"), "- 2021-01-01 · t/t · meta\n")
	git(t, clone, "commit", "-q", "-am", "meta")
	if fs := findings(t, clone); has(fs, "hn", "stale-brief") {
		t.Errorf("stale-brief after a meta-only commit: %+v", fs)
	}
	// An uncommitted edit, and an untracked file, are changes.
	write(t, filepath.Join(clone, "docs/notes.md"), "edited\n")
	if fs := findings(t, clone); !has(fs, "hn", "stale-brief") {
		t.Errorf("stale-brief not reported for an uncommitted edit: %+v", fs)
	}
	git(t, clone, "checkout", "-q", "--", "docs/notes.md")
	write(t, filepath.Join(clone, "new.py"), "x\n")
	if fs := findings(t, clone); !has(fs, "hn", "stale-brief") {
		t.Errorf("stale-brief not reported for an untracked file: %+v", fs)
	}
	os.Remove(filepath.Join(clone, "new.py"))
	// A commit outside the meta is a change.
	write(t, filepath.Join(clone, "main.py"), "print(1)\n")
	git(t, clone, "add", "main.py")
	git(t, clone, "commit", "-q", "-m", "work")
	if err := os.Chtimes(filepath.Join(clone, "main.py"), time.Unix(0, 0), time.Unix(0, 0)); err != nil {
		t.Fatal(err)
	}
	if fs := findings(t, clone); !has(fs, "hn", "stale-brief") {
		t.Errorf("stale-brief not reported for a new commit: %+v", fs)
	}
}

func TestPointerThroughClaudeLocal(t *testing.T) {
	proj := filepath.Join(t.TempDir(), "hn")
	standalone(t, proj, time.Now().Format("2006-01-02"))
	write(t, filepath.Join(proj, "CLAUDE.md"), "# ADR rules for Claude\n")
	if fs := findings(t, proj); !has(fs, "hn", "pointer") {
		t.Errorf("pointer not reported for a CLAUDE.md without @AGENTS.md: %+v", fs)
	}
	write(t, filepath.Join(proj, "CLAUDE.local.md"), "@AGENTS.md\n")
	if fs := findings(t, proj); has(fs, "hn", "pointer") {
		t.Errorf("pointer reported although CLAUDE.local.md has @AGENTS.md: %+v", fs)
	}
}

func TestSingleProjectWithoutPack(t *testing.T) {
	repo := filepath.Join(t.TempDir(), "repo")
	write(t, filepath.Join(repo, "main.go"), "package main\n")

	out := must(t, repo, "start", "--by", "codex/gpt-5")
	if !strings.Contains(out, "No pack around") {
		t.Errorf("start outside a pack: %q", out)
	}
	readme := read(t, filepath.Join(repo, ".omut/README.md"))
	for _, want := range []string{"## Reading it", "## Keeping it", "### Adopting a project"} {
		if !strings.Contains(readme, want) {
			t.Errorf(".omut/README.md has no %q", want)
		}
	}
	agents := read(t, filepath.Join(repo, "AGENTS.md"))
	if !strings.Contains(agents, "Full rules: `.omut/README.md`") || strings.Contains(agents, "../.omut") {
		t.Errorf("the block points outside the project:\n%s", agents)
	}
	if target, err := os.Readlink(filepath.Join(repo, "CLAUDE.md")); err != nil || target != "AGENTS.md" {
		t.Errorf("CLAUDE.md → %q, %v", target, err)
	}
	fs := findings(t, repo)
	if check.HasErrors(fs) || has(fs, "repo", "pointer") || !has(fs, "repo", "unfilled") {
		t.Errorf("fresh single project: %+v", fs)
	}

	write(t, filepath.Join(repo, ".omut/README.md"), "<!-- Tool file -->\nold\n")
	if fs := findings(t, repo); !has(fs, "repo", "outdated") {
		t.Errorf("outdated README not reported: %+v", fs)
	}
	must(t, repo, "zero")
	if read(t, filepath.Join(repo, ".omut/README.md")) != readme {
		t.Error("zero did not restore .omut/README.md")
	}

	must(t, repo, "freeze")
	if !strings.Contains(read(t, filepath.Join(repo, ".omut/brief.md")), "status: frozen") {
		t.Error("freeze outside a pack did not set frozen")
	}
	must(t, repo, "thaw")
	// A path works as well as the current directory.
	if out := must(t, filepath.Dir(repo), "start", "repo"); !strings.Contains(out, "has its meta already") {
		t.Errorf("second start: %q", out)
	}
	if _, err := omut(t, t.TempDir(), "zero"); err == nil || !strings.Contains(err.Error(), "omut start") {
		t.Errorf("zero in a directory without meta: %v", err)
	}
}

func TestPackProjectsGetTheirRules(t *testing.T) {
	root := t.TempDir()
	must(t, root, "init")
	must(t, root, "start", "hn")
	if _, err := os.Stat(filepath.Join(root, "hn/.omut/README.md")); err != nil {
		t.Fatal("start in a pack did not lay down .omut/README.md")
	}
	os.Remove(filepath.Join(root, "hn/.omut/README.md"))
	if fs := findings(t, root, "hn"); !has(fs, "hn", "pointer") {
		t.Errorf("missing README not reported: %+v", fs)
	}
	must(t, root, "zero", "hn")
	if fs := findings(t, root, "hn"); has(fs, "hn", "pointer") || has(fs, "hn", "outdated") {
		t.Errorf("after zero: %+v", fs)
	}
}

func TestCheckFailOnWarn(t *testing.T) {
	proj := filepath.Join(t.TempDir(), "hn")
	today := time.Now().Format("2006-01-02")
	standalone(t, proj, today)
	var exit exitCode
	if _, err := omut(t, proj, "check", "--fail-on", "warn"); err != nil {
		t.Errorf("clean meta: %v", err)
	}
	write(t, filepath.Join(proj, ".omut/brief.md"), fill(today, "2020-01-01", ""))
	if _, err := omut(t, proj, "check"); err != nil {
		t.Errorf("a warning must not fail by default: %v", err)
	}
	if _, err := omut(t, proj, "check", "--fail-on", "warn"); !errors.As(err, &exit) {
		t.Errorf("--fail-on warn did not fail on unverified: %v", err)
	}
	if _, err := omut(t, proj, "check", "--fail-on", "info"); err == nil {
		t.Error("--fail-on info must be refused")
	}
}

// A commit made in the evening west of UTC is already "tomorrow" in UTC,
// where CI runs. The brief's date was written in the committer's zone, so
// that is the zone to compare in.
func TestStaleBriefInCommitterTimeZone(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("no git")
	}
	local := time.Local
	time.Local = time.UTC
	t.Cleanup(func() { time.Local = local })

	repo := filepath.Join(t.TempDir(), "hn")
	standalone(t, repo, "2026-11-02")
	git(t, repo, "init", "-q")
	git(t, repo, "add", ".")
	t.Setenv("GIT_COMMITTER_DATE", "2026-11-02T18:00:00-08:00") // 02:00 UTC on 11-03
	git(t, repo, "commit", "-qm", "evening work")
	clone := filepath.Join(t.TempDir(), "hn")
	git(t, filepath.Dir(clone), "clone", "-q", repo, clone)
	if fs := findings(t, clone); has(fs, "hn", "stale-brief") {
		t.Errorf("stale-brief from the time zone, not from the work: %+v", fs)
	}
}
