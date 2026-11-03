package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
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
	write(t, filepath.Join(proj, ".omut/decisions.md"), "# Decisions\n\n- D1 [open] Keep the cron or drop it\n")
	if fs := findings(t, root, "hn"); !has(fs, "hn", "open-decision") {
		t.Errorf("open decision not listed: %+v", fs)
	}
	out := must(t, root, "freeze", "hn")
	if !strings.Contains(out, "Where it lives") {
		t.Errorf("freeze did not warn about a live footprint: %q", out)
	}
	fs = findings(t, root, "hn")
	if !has(fs, "hn", "frozen-but-live") || has(fs, "hn", "stale-brief") || has(fs, "hn", "open-decision") {
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

func TestKillSeesIgnoredFiles(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("no git")
	}
	root := t.TempDir()
	must(t, root, "init")
	must(t, root, "start", "hn")
	dir := filepath.Join(root, "hn")
	remote := filepath.Join(t.TempDir(), "hn.git")
	git(t, root, "init", "-q", "--bare", remote)
	write(t, filepath.Join(dir, ".gitignore"), ".env\ntarget/\n")
	git(t, dir, "init", "-q")
	git(t, dir, "add", "-A")
	git(t, dir, "commit", "-qm", "init")
	git(t, dir, "remote", "add", "origin", remote)
	git(t, dir, "push", "-q", "origin", "HEAD")
	write(t, filepath.Join(dir, "target/debug/hn"), "bin\n")
	write(t, filepath.Join(dir, ".env"), "KEY=1\n")

	_, err := omut(t, root, "kill", "hn", "--yes", "--reason", "x")
	if err == nil || !strings.Contains(err.Error(), "gitignored, exists only here: .env") || strings.Contains(err.Error(), "target") {
		t.Errorf("kill with a gitignored .env: %v", err)
	}
	os.Remove(filepath.Join(dir, ".env"))
	write(t, filepath.Join(dir, ".omut/local/log.md"), "- 2026-11-03 · human · deployed to vps1\n")
	if _, err := omut(t, root, "kill", "hn", "--yes", "--reason", "x"); err == nil || !strings.Contains(err.Error(), ".omut/local/") {
		t.Errorf("kill with a filled instance layer: %v", err)
	}
	write(t, filepath.Join(dir, ".omut/local/log.md"), "<!-- newest first -->\n")
	must(t, root, "kill", "hn", "--yes", "--reason", "x")
}

func TestStartOffersTheBadge(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("no git")
	}
	for remote, want := range map[string]string{
		"git@github.com:acme/hn.git":   "github.com/acme/hn/actions/workflows/omut.yml/badge.svg?branch=main)](.omut/)",
		"https://github.com/acme/hn":   "github.com/acme/hn/actions/workflows/omut.yml/badge.svg?branch=main)](.omut/)",
		"https://gitlab.com/acme/hn":   "",
		"git@github.com:acme/my.site/": "github.com/acme/my.site/actions",
	} {
		dir := t.TempDir()
		git(t, dir, "init", "-q")
		git(t, dir, "remote", "add", "origin", remote)
		out := must(t, dir, "start")
		if want == "" && strings.Contains(out, "badge") || want != "" && !strings.Contains(out, want) {
			t.Errorf("start with origin %s:\n%s", remote, out)
		}
	}
	dir := t.TempDir()
	git(t, dir, "init", "-q")
	git(t, dir, "remote", "add", "origin", "git@github.com:acme/mono.git")
	sub := filepath.Join(dir, "tools", "hn")
	write(t, filepath.Join(sub, "main.go"), "package main\n")
	if out := must(t, sub, "start"); strings.Contains(out, "badge") {
		t.Errorf("a subdirectory of a repository got the badge:\n%s", out)
	}
}

func TestADRsAreTheRegistry(t *testing.T) {
	dir := t.TempDir()
	today := time.Now().Format("2006-01-02")
	standalone(t, dir, today)
	brief := filepath.Join(dir, ".omut/brief.md")
	write(t, brief, strings.Replace(read(t, brief), "by: codex/gpt-5\n", "by: codex/gpt-5\nadr: docs/adr\n", 1))
	if fs := findings(t, dir); !has(fs, filepath.Base(dir), "shape") {
		t.Errorf("adr: names a missing directory, no shape error: %+v", fs)
	}
	write(t, filepath.Join(dir, "docs/adr/0001-sqlite.md"), "# SQLite\n\nStatus: Accepted\n")
	write(t, filepath.Join(dir, "docs/adr/0002-cache.md"), "# Cache pages\n\nStatus: Proposed\n")
	write(t, filepath.Join(dir, "docs/adr/0003-timer.md"), "# A systemd timer\n\nFirmness: tentative\nRevisit if: it misses runs\n")
	write(t, filepath.Join(dir, "docs/adr/README.md"), "# ADRs\n\nStatus: proposed\n")

	var open []string
	for _, f := range findings(t, dir) {
		if f.Level != check.Info {
			t.Errorf("unexpected: %+v", f)
		}
		if f.Rule == "open-decision" {
			open = append(open, f.Message)
		}
	}
	if strings.Join(open, "|") != "0002-cache [open] Cache pages|0003-timer [doubted] A systemd timer" {
		t.Errorf("ADRs waiting: %q", open)
	}
	out, _, err := hookRun(t, dir, "", "start")
	if err != nil || !strings.Contains(out, "- `docs/adr/0002-cache.md` [open] Cache pages") {
		t.Errorf("hook start does not list the proposed ADR (%v):\n%s", err, out)
	}

	write(t, filepath.Join(dir, "docs/adr/0004-newer.md"), "# Newer than the registry\n")
	write(t, brief, strings.Replace(read(t, brief), "status: active", "status: frozen", 1))
	for _, f := range findings(t, dir) {
		if f.Rule == "open-decision" || f.Rule == "adr-ahead" {
			t.Errorf("frozen, or no copy in decisions.md, yet: %+v", f)
		}
	}
}

func TestInstanceLayerStaysOutOfGit(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("no git")
	}
	dir := t.TempDir()
	git(t, dir, "init", "-q")
	must(t, dir, "start")
	for _, f := range []string{".omut/.gitignore", ".omut/local/instance.md", ".omut/local/log.md"} {
		if _, err := os.Stat(filepath.Join(dir, f)); err != nil {
			t.Errorf("start did not lay down %s", f)
		}
	}
	git(t, dir, "add", "-A")
	out, err := exec.Command("git", "-C", dir, "diff", "--cached", "--name-only").Output()
	if err != nil || strings.Contains(string(out), "local/") || !strings.Contains(string(out), ".omut/.gitignore") {
		t.Errorf("git add -A staged:\n%s", out)
	}
	if fs := findings(t, dir); has(fs, filepath.Base(dir), "local-not-ignored") || has(fs, filepath.Base(dir), "local-in-git") {
		t.Errorf("a fresh start: %+v", fs)
	}

	write(t, filepath.Join(dir, ".omut/local/instance.md"), "---\nupdated: 2026-11-03\nverified:\nby: human\n---\n## Where it lives\n- container hn on vps1, image hn:abc\n\n## Access\n- token in ~/hn/.env\n\n## Runbook\n")
	write(t, filepath.Join(dir, ".omut/local/log.md"), "- 2026-11-03 · human · deployed abc to vps1\n")
	hookOut, _, _ := hookRun(t, dir, "", "start")
	for _, want := range []string{"## .omut/local/instance.md (this copy only, never in git)", "container hn on vps1", "deployed abc to vps1"} {
		if !strings.Contains(hookOut, want) {
			t.Errorf("hook start lacks %q:\n%s", want, hookOut)
		}
	}
	if fs := findings(t, dir); !has(fs, filepath.Base(dir), "unchecked") {
		t.Errorf("claims without a way to check them not counted: %+v", fs)
	}

	os.RemoveAll(filepath.Join(dir, ".omut/local"))
	write(t, filepath.Join(dir, ".omut/decisions.md"), "# Decisions\n\n- D1 [accepted 2026-11-03] Deploy by hand → local/instance.md\n")
	if fs := findings(t, dir); has(fs, filepath.Base(dir), "shape") {
		t.Errorf("a link into local/ on a clone without it is not broken: %+v", fs)
	}
	must(t, dir, "zero")
	os.Remove(filepath.Join(dir, ".omut/.gitignore"))
	if fs := findings(t, dir); !has(fs, filepath.Base(dir), "local-not-ignored") {
		t.Errorf("no .omut/.gitignore, no warning: %+v", fs)
	}
	git(t, dir, "add", "-f", ".omut/local/instance.md")
	if fs := findings(t, dir); !has(fs, filepath.Base(dir), "local-in-git") {
		t.Errorf("the instance layer is staged, no error: %+v", fs)
	}
}

func TestLeakOfInstanceFacts(t *testing.T) {
	dir := t.TempDir()
	today := time.Now().Format("2006-01-02")
	standalone(t, dir, today)
	brief := filepath.Join(dir, ".omut/brief.md")
	clean := read(t, brief)
	name := filepath.Base(dir)

	write(t, brief, strings.Replace(clean, "Works.\n", "Works on Python 3.12.1 and 1.2.3.4-beta; rules in ~/.claude/settings.json.\n", 1))
	write(t, filepath.Join(dir, ".omut/local/instance.md"), "---\nupdated: "+today+"\nverified: "+today+"\nby: human\n---\n## Where it lives\n- ~/hn, 10.0.0.5, /Users/bob/hn\n")
	if fs := findings(t, dir); has(fs, name, "leak") {
		t.Errorf("versions, a general ~/ path or the instance layer reported as a leak: %+v", fs)
	}
	for what, text := range map[string]string{
		"~/ where it lives": strings.Replace(clean, "## Where it lives\n", "## Where it lives\n- symlink ~/.claude/skills/hn → here\n", 1),
		"an IP":             strings.Replace(clean, "Works.\n", "Works on 10.0.0.5.\n", 1),
		"a home path":       strings.Replace(clean, "Works.\n", "Works from /home/bob/hn.\n", 1),
	} {
		write(t, brief, text)
		if fs := findings(t, dir); !has(fs, name, "leak") {
			t.Errorf("%s not reported: %+v", what, fs)
		}
	}
}

func TestClaimsAreCheckedByAgents(t *testing.T) {
	dir := t.TempDir()
	today := time.Now()
	day := func(ago int) string { return today.AddDate(0, 0, -ago).Format("2006-01-02") }
	standalone(t, dir, day(0))
	name := filepath.Base(dir)
	brief := filepath.Join(dir, ".omut/brief.md")
	write(t, brief, strings.Replace(read(t, brief), "## Where it lives\n", "## Where it lives\n- PyPI: hn 1.0 · how: pip index versions hn · seen "+day(2)+"\n- GitHub: public\n", 1))
	write(t, filepath.Join(dir, ".omut/local/instance.md"), "---\nupdated: "+day(0)+"\nby: human\n---\n## Where it lives\n"+
		"- skill hn → this checkout · how: readlink ~/.claude/skills/hn · seen "+day(40)+"\n"+
		"- cron on vps1 · how: ssh vps1 crontab -l\n"+
		"- data in ~/hn-data · how: ls ~/hn-data · seen "+day(10)+"\n"+
		"## Access\n- token in ~/hn/.env · how: test -f ~/hn/.env · seen yesterday\n")

	fs := findings(t, dir)
	var unseen []int
	for _, f := range fs {
		if f.Rule == "unseen" {
			unseen = append(unseen, f.Line)
		}
	}
	if fmt.Sprint(unseen) != "[6 7]" || !has(fs, name, "unchecked") || !has(fs, name, "shape") {
		t.Errorf("unseen on lines %v (want the 40-day-old and the never-seen claims), all: %+v", unseen, fs)
	}

	out, _, _ := hookRun(t, dir, "", "start")
	_, out, _ = strings.Cut(out, "## Check while you are here")
	i, j := strings.Index(out, "`.omut/local/instance.md:7` cron on vps1"), strings.Index(out, "`.omut/local/instance.md:6` skill hn")
	if i < 0 || j < i || strings.Count(out, "· how: `") != 2 || strings.Contains(out, "PyPI: hn 1.0") {
		t.Errorf("hook start should offer the never-seen claim, then the oldest, two at most, not the fresh one:\n%s", out)
	}
	if !strings.Contains(out, "1 claim about where the project lives says no way to check it") {
		t.Errorf("hook start does not count the unchecked claim:\n%s", out)
	}

	write(t, brief, strings.Replace(read(t, brief), "status: active", "status: frozen", 1))
	if fs := findings(t, dir); has(fs, name, "unseen") {
		t.Errorf("a frozen project's claims: %+v", fs)
	}
	if out, _, _ := hookRun(t, dir, "", "start"); strings.Contains(out, "Check while you are here") {
		t.Errorf("a frozen project is asked to check claims:\n%s", out)
	}
}

func gitOut(t *testing.T, dir string, args ...string) string {
	t.Helper()
	out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return string(out)
}

func TestPrivateMetaLeavesTheRepoAlone(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("no git")
	}
	if _, err := omut(t, t.TempDir(), "start", "--private"); err == nil {
		t.Error("--private outside git: no error")
	}

	dir := t.TempDir()
	name := filepath.Base(dir)
	git(t, dir, "init", "-q")
	write(t, filepath.Join(dir, "README.md"), "# theirs\n")
	git(t, dir, "add", "-A")
	git(t, dir, "commit", "-qm", "init")

	out := must(t, dir, "start", "--private")
	if !strings.Contains(out, "The meta is private") || strings.Contains(out, "badge") {
		t.Errorf("start --private said:\n%s", out)
	}
	for _, f := range []string{"AGENTS.md", "CLAUDE.md"} {
		if _, err := os.Lstat(filepath.Join(dir, f)); err == nil {
			t.Errorf("--private created %s in their repository", f)
		}
	}
	if !strings.Contains(read(t, filepath.Join(dir, "CLAUDE.local.md")), "<!-- omut:start -->") {
		t.Error("no omut block in CLAUDE.local.md")
	}
	if st := gitOut(t, dir, "status", "--porcelain", "--untracked-files=all"); st != "" {
		t.Errorf("the repository changed:\n%s", st)
	}
	if ex := read(t, filepath.Join(dir, ".git/info/exclude")); !strings.Contains(ex, "/.omut/\n") || !strings.Contains(ex, "/CLAUDE.local.md\n") {
		t.Errorf(".git/info/exclude:\n%s", ex)
	}
	must(t, dir, "start", "--private") // twice: the exclude lines are not repeated
	if n := strings.Count(read(t, filepath.Join(dir, ".git/info/exclude")), "/.omut/"); n != 1 {
		t.Errorf("/.omut/ is in .git/info/exclude %d times", n)
	}

	write(t, filepath.Join(dir, ".omut/brief.md"), fill(time.Now().Format("2006-01-02"), time.Now().Format("2006-01-02"), "- checkout in /Users/someone/work/theirs"))
	fs := findings(t, dir)
	for _, rule := range []string{"pointer", "leak"} {
		if has(fs, name, rule) {
			t.Errorf("private meta got %s: %+v", rule, fs)
		}
	}
	hookOut, _, _ := hookRun(t, dir, "", "start")
	if !strings.Contains(hookOut, "The meta is this copy's only") {
		t.Errorf("hook start does not say the meta is private:\n%s", hookOut)
	}
	must(t, dir, "zero")
	if _, err := os.Lstat(filepath.Join(dir, "AGENTS.md")); err == nil {
		t.Error("zero created AGENTS.md in a private project")
	}
	os.Remove(filepath.Join(dir, "CLAUDE.local.md"))
	if fs := findings(t, dir); !has(fs, name, "pointer") {
		t.Errorf("private meta without CLAUDE.local.md: no pointer warning: %+v", fs)
	}

	// Sharing it later: drop the exclude lines, zero, commit.
	write(t, filepath.Join(dir, ".git/info/exclude"), "")
	must(t, dir, "zero")
	if !strings.Contains(read(t, filepath.Join(dir, "AGENTS.md")), "<!-- omut:start -->") {
		t.Error("zero after sharing: no omut block in AGENTS.md")
	}
	git(t, dir, "add", "-A")
	git(t, dir, "commit", "-qm", "share the meta")
	if _, err := omut(t, dir, "start", "--private"); err == nil {
		t.Error("--private over meta already in git: no error")
	}

	// A project in a subdirectory of the repository.
	sub := filepath.Join(dir, "tools", "bot")
	write(t, filepath.Join(sub, "main.go"), "package main\n")
	must(t, sub, "start", "--private")
	if ex := read(t, filepath.Join(dir, ".git/info/exclude")); !strings.Contains(ex, "/tools/bot/.omut/\n") {
		t.Errorf("exclude for a subdirectory:\n%s", ex)
	}
}

func TestParallelLogLinesMerge(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("no git")
	}
	dir := t.TempDir()
	git(t, dir, "init", "-q")
	must(t, dir, "start")
	log := filepath.Join(dir, ".omut/log.md")
	write(t, log, "- 2026-11-01 · human · started\n")
	git(t, dir, "add", "-A")
	git(t, dir, "commit", "-qm", "start")
	main := strings.TrimSpace(gitOut(t, dir, "branch", "--show-current"))

	git(t, dir, "checkout", "-qb", "anna")
	write(t, log, "- 2026-11-03 · claude-code/opus · anna's work\n- 2026-11-01 · human · started\n")
	git(t, dir, "commit", "-qam", "anna")
	git(t, dir, "checkout", "-q", main)
	write(t, log, "- 2026-11-03 · codex/gpt-5 · boris's work\n- 2026-11-01 · human · started\n")
	git(t, dir, "commit", "-qam", "boris")

	git(t, dir, "merge", "-q", "--no-edit", "anna")
	got := read(t, log)
	for _, want := range []string{"anna's work", "boris's work", "started"} {
		if strings.Count(got, want) != 1 {
			t.Errorf("after the merge the log has %q %d times:\n%s", want, strings.Count(got, want), got)
		}
	}
	if fs := findings(t, dir); has(fs, filepath.Base(dir), "shape") {
		t.Errorf("merged log out of shape: %+v", fs)
	}
}
