package main

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/metravod/omut/internal/hook"
	"github.com/metravod/omut/templates"
)

func gitRepo(t *testing.T, dir string) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("no git")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	git(t, dir, "init", "-q")
	write(t, filepath.Join(dir, "README.md"), "# x\n")
	git(t, dir, "add", "-A")
	git(t, dir, "commit", "-qm", "init")
}

func exists(path string) bool {
	_, err := os.Lstat(path)
	return err == nil
}

// Private meta is the clone's: every worktree finds it, none gets its own.
func TestPrivateMetaInAWorktree(t *testing.T) {
	base := t.TempDir()
	repo, wt := filepath.Join(base, "repo"), filepath.Join(base, "wt")
	gitRepo(t, repo)
	must(t, repo, "start", "--private")
	git(t, repo, "worktree", "add", "-q", wt)
	brief := filepath.Join(repo, ".omut/brief.md")

	out, _, _ := hookRun(t, wt, "", "start")
	if !strings.Contains(out, "the meta of repo") || !strings.Contains(out, "linked git worktree") {
		t.Errorf("hook start in a worktree:\n%s", out)
	}
	if !strings.Contains(out, filepath.Join(repo, ".omut")) && !strings.Contains(out, "/repo/.omut") {
		t.Errorf("hook start doesn't give the full path:\n%s", out)
	}
	if fs := findings(t, wt); !has(fs, "repo", "unfilled") {
		t.Errorf("check from a worktree: %+v", fs)
	}
	must(t, wt, "zero")
	if out := must(t, wt, "start", "--private"); !strings.Contains(out, "has its meta already") {
		t.Errorf("start --private in a worktree said:\n%s", out)
	}
	if out := must(t, wt, "freeze"); !strings.Contains(out, "repo: status: frozen") {
		t.Errorf("freeze from a worktree said:\n%s", out)
	}
	if !strings.Contains(read(t, brief), "status: frozen") {
		t.Error("freeze from a worktree did not reach the meta")
	}
	if exists(filepath.Join(wt, ".omut")) {
		t.Error("a worktree got its own .omut/")
	}
	if st := gitOut(t, wt, "status", "--porcelain", "--untracked-files=all"); st != "" {
		t.Errorf("the worktree changed:\n%s", st)
	}

	// Asking for it explicitly still works.
	must(t, wt, "start", ".")
	if !exists(filepath.Join(wt, ".omut/brief.md")) {
		t.Error("start . in a worktree laid nothing down")
	}
}

func TestPrivateMetaStartedInAWorktreeGoesToTheMainCheckout(t *testing.T) {
	base := t.TempDir()
	repo, wt := filepath.Join(base, "repo"), filepath.Join(base, "wt")
	gitRepo(t, repo)
	git(t, repo, "worktree", "add", "-q", wt)
	must(t, wt, "start", "--private")
	if !exists(filepath.Join(repo, ".omut/brief.md")) || exists(filepath.Join(wt, ".omut")) {
		t.Error("private meta should be in the main checkout and not in the worktree")
	}
	if st := gitOut(t, repo, "status", "--porcelain", "--untracked-files=all"); st != "" {
		t.Errorf("the repository changed:\n%s", st)
	}
}

// Committed meta follows the branch; this copy's layer stays in the main checkout.
func TestCommittedMetaInAWorktree(t *testing.T) {
	base := t.TempDir()
	repo, wt := filepath.Join(base, "repo"), filepath.Join(base, "wt")
	gitRepo(t, repo)
	must(t, repo, "start")
	instance := filepath.Join(repo, ".omut/local/instance.md")
	write(t, instance, strings.Replace(read(t, instance), "## Access", "- server only-here · how: ssh only-here · seen 2020-01-01\n\n## Access", 1))
	git(t, repo, "add", "-A")
	git(t, repo, "commit", "-qm", "meta")
	git(t, repo, "worktree", "add", "-q", wt)

	out, _, _ := hookRun(t, wt, "", "start")
	if !strings.Contains(out, "the meta of wt") || !strings.Contains(out, "follows this branch") {
		t.Errorf("hook start in a worktree:\n%s", out)
	}
	if strings.Count(out, "only-here") < 2 { // the layer, and the claim offered for checking
		t.Errorf("hook start doesn't show this copy's memory and its due claim:\n%s", out)
	}
	if !strings.Contains(out, "/repo/.omut/local") {
		t.Errorf("hook start doesn't say where this copy's memory is:\n%s", out)
	}
	must(t, wt, "zero")
	if exists(filepath.Join(wt, ".omut/local")) {
		t.Error("zero gave the worktree a memory of its own")
	}
	if !exists(filepath.Join(wt, ".omut/threads")) {
		t.Error("zero did not repair the worktree's own meta")
	}
}

func TestHookCommitInAWorktreeOfPrivateMeta(t *testing.T) {
	base := t.TempDir()
	repo, wt := filepath.Join(base, "repo"), filepath.Join(base, "wt")
	gitRepo(t, repo)
	must(t, repo, "start", "--private")
	git(t, repo, "worktree", "add", "-q", wt)
	past := time.Now().Add(-time.Hour)
	os.Chtimes(filepath.Join(repo, ".omut/log.md"), past, past)
	write(t, filepath.Join(wt, "code.go"), "package x\n")
	git(t, wt, "add", "-A")
	if res := hook.Commit(wt, false); !strings.Contains(res.Message, "code.go") {
		t.Errorf("no reminder from a worktree: %+v", res)
	}
}

func TestInsideASingleRepository(t *testing.T) {
	repo := filepath.Join(t.TempDir(), "repo")
	gitRepo(t, repo)
	must(t, repo, "start")
	deep := filepath.Join(repo, "src", "pkg")
	write(t, filepath.Join(deep, "x.go"), "package x\n")

	if fs := findings(t, deep); !has(fs, "repo", "unfilled") {
		t.Errorf("check from a subdirectory: %+v", fs)
	}
	if out := must(t, deep, "start"); !strings.Contains(out, "To give this directory its own: omut start .") {
		t.Errorf("start from a subdirectory said:\n%s", out)
	}
	if exists(filepath.Join(deep, ".omut")) {
		t.Error("start from a subdirectory laid down a second meta")
	}
	must(t, deep, "zero")
	if exists(filepath.Join(deep, ".omut")) {
		t.Error("zero from a subdirectory laid down a second meta")
	}
	if out := must(t, deep, "freeze"); !strings.Contains(out, "repo: status: frozen") {
		t.Errorf("freeze from a subdirectory said:\n%s", out)
	}
	must(t, deep, "thaw")

	// A monorepo asks for it.
	must(t, deep, "start", ".")
	if !exists(filepath.Join(deep, ".omut/brief.md")) {
		t.Error("start . laid nothing down")
	}
}

// Another repository inside a project is not that project.
func TestARepositoryInsideAProjectIsItsOwn(t *testing.T) {
	outer := filepath.Join(t.TempDir(), "outer")
	gitRepo(t, outer)
	must(t, outer, "start")
	inner := filepath.Join(outer, "vendor", "inner")
	gitRepo(t, inner)

	if _, err := omut(t, inner, "check"); !errors.Is(err, errNoMeta) {
		t.Errorf("check in a repository without meta: %v", err)
	}
	must(t, inner, "start")
	if !exists(filepath.Join(inner, ".omut/brief.md")) {
		t.Error("start in the inner repository laid nothing down")
	}
}

func TestNoMetaSaysWhatToDo(t *testing.T) {
	_, err := omut(t, t.TempDir(), "check")
	if err == nil || !strings.Contains(err.Error(), "omut start") || strings.Contains(err.Error(), "LEAD.md") {
		t.Errorf("check without meta: %v", err)
	}
}

func TestAWorktreeBesideAProjectOfAPack(t *testing.T) {
	root := t.TempDir()
	must(t, root, "init")
	repo, wt := filepath.Join(root, "repo"), filepath.Join(root, "repo-wt")
	gitRepo(t, repo)
	must(t, root, "start", "repo", "--private")
	git(t, repo, "worktree", "add", "-q", wt)

	if fs := findings(t, root); has(fs, "repo-wt", "no-meta") {
		t.Errorf("a worktree counted as a project: %+v", fs)
	}
	if fs := findings(t, wt); !has(fs, "repo", "unfilled") || has(fs, "repo-wt", "no-meta") {
		t.Errorf("check from the worktree: %+v", fs)
	}
	must(t, wt, "zero")
	if exists(filepath.Join(wt, ".omut")) {
		t.Error("zero gave the worktree its own .omut/")
	}
}

func TestOpenDecisionMaySaySinceWhen(t *testing.T) {
	dir := t.TempDir()
	must(t, dir, "start")
	path := filepath.Join(dir, ".omut/decisions.md")
	write(t, path, read(t, path)+"\n- D1 [open 2026-11-04] Which way? a or b\n")
	name := filepath.Base(dir)
	fs := findings(t, dir)
	if !has(fs, name, "open-decision") || has(fs, name, "shape") {
		t.Errorf("a dated open decision: %+v", fs)
	}
}

// The human's choice about agents' memory is a line zero leaves alone, and the
// rules tell the agent to ask for it.
func TestMemoryChoiceSurvivesZero(t *testing.T) {
	b, err := templates.FS.ReadFile("project/omut/README.md")
	if err != nil {
		t.Fatal(err)
	}
	const choice = "Agents' memory holds only the human's preferences"
	if !strings.Contains(string(b), choice) || !strings.Contains(string(b), "Ask the human once") {
		t.Error("the rules don't offer the choice")
	}
	dir := t.TempDir()
	must(t, dir, "start")
	agents := filepath.Join(dir, "AGENTS.md")
	write(t, agents, read(t, agents)+"\n"+choice+"\n")
	write(t, agents, strings.Replace(read(t, agents), "## Project meta", "## Old text", 1))
	must(t, dir, "zero")
	got := read(t, agents)
	if !strings.Contains(got, choice) || strings.Contains(got, "## Old text") {
		t.Errorf("after zero:\n%s", got)
	}
}
