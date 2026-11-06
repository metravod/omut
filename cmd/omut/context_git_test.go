package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/metravod/omut/internal/context"
)

func gitContextProject(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	gitRepo(t, repo)
	git(t, repo, "branch", "-M", "main")
	must(t, repo, "start", "--private")
	today := time.Now().Format("2006-01-02")
	write(t, filepath.Join(repo, ".omut/brief.md"), fill(today, today, ""))
	write(t, filepath.Join(repo, "docs/notes.md"), "# notes\n")
	git(t, repo, "add", "docs")
	git(t, repo, "commit", "-qm", "project docs")
	return repo
}

func gitChanges(t *testing.T, g *context.GitState) map[string]context.GitChange {
	t.Helper()
	if g == nil || g.State != "available" || g.Clean == nil {
		t.Fatalf("Git observation unavailable: %+v", g)
	}
	out := map[string]context.GitChange{}
	for _, c := range g.Changes {
		out[c.Path] = c
	}
	return out
}

func TestContextGitStatusAndLocalDivergence(t *testing.T) {
	repo := gitContextProject(t)
	for _, name := range []string{"tracked.txt", "deleted.txt", "old \tname.txt"} {
		write(t, filepath.Join(repo, name), "base content\n")
	}
	git(t, repo, "add", "-A")
	git(t, repo, "commit", "-qm", "files")
	git(t, repo, "branch", "remote-side")
	write(t, filepath.Join(repo, "tracked.txt"), "local commit\n")
	git(t, repo, "commit", "-qam", "local")
	git(t, repo, "checkout", "-q", "remote-side")
	write(t, filepath.Join(repo, "remote-only.txt"), "remote commit\n")
	git(t, repo, "add", "remote-only.txt")
	git(t, repo, "commit", "-qm", "remote")
	remote := strings.TrimSpace(gitOut(t, repo, "rev-parse", "HEAD"))
	git(t, repo, "update-ref", "refs/remotes/origin/main", remote)
	git(t, repo, "checkout", "-q", "main")
	git(t, repo, "remote", "add", "origin", "https://must-not-contact.invalid/repository")
	git(t, repo, "config", "branch.main.remote", "origin")
	git(t, repo, "config", "branch.main.merge", "refs/heads/main")

	write(t, filepath.Join(repo, "tracked.txt"), "staged change\n")
	git(t, repo, "add", "tracked.txt")
	write(t, filepath.Join(repo, "tracked.txt"), "unstaged too\n")
	if err := os.Remove(filepath.Join(repo, "deleted.txt")); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(filepath.Join(repo, "old \tname.txt"), filepath.Join(repo, "new \nname.txt")); err != nil {
		t.Fatal(err)
	}
	git(t, repo, "add", "--", "old \tname.txt", "new \nname.txt")
	write(t, filepath.Join(repo, "а б\t\n.txt"), "untracked\n")
	r, first, err := contextReport(t, repo)
	if err != nil || r.Incomplete {
		t.Fatalf("context: %+v, %v", r, err)
	}
	g := r.Projects[0].Git
	changes := gitChanges(t, g)
	if g.Branch != "main" || g.Upstream != "origin/main" || g.Ahead == nil || *g.Ahead != 1 || g.Behind == nil || *g.Behind != 1 || *g.Clean || g.Conflicts != 0 {
		t.Errorf("branch or local divergence: %+v", g)
	}
	if len(changes) != 4 || changes["tracked.txt"].Status != "MM" || changes["deleted.txt"].Status != " D" || changes["а б\t\n.txt"].Status != "??" || changes["new \nname.txt"].Status != "R " || changes["new \nname.txt"].OriginalPath != "old \tname.txt" {
		t.Errorf("paths/status/rename lost: %+v", changes)
	}
	text := must(t, repo, "context")
	for _, want := range []string{"ahead 1, behind 1", "local refs; no fetch", "XY = index/worktree", `"new \nname.txt"`, `"old \tname.txt"`} {
		if !strings.Contains(text, want) {
			t.Errorf("text output lost %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, "unstaged too") || exists(filepath.Join(repo, ".git/FETCH_HEAD")) {
		t.Error("context included full diff content or fetched from a remote")
	}
	_, second, err := contextReport(t, repo)
	if err != nil || first != second {
		t.Error("unchanged Git state produced different JSON")
	}
	git(t, repo, "update-ref", "-d", "refs/remotes/origin/main")
	r, _, err = contextReport(t, repo)
	g = r.Projects[0].Git
	if err != nil || g.Upstream != "origin/main" || g.Ahead != nil || g.Behind != nil {
		t.Errorf("missing local upstream ref was reported as synchronized: %+v, %v", g, err)
	}
}

func TestContextGitHeadStatesAndFailure(t *testing.T) {
	repo := gitContextProject(t)
	r, _, err := contextReport(t, repo)
	g := r.Projects[0].Git
	gitChanges(t, g)
	if err != nil || !*g.Clean || g.Upstream != "" || g.Ahead != nil || g.Behind != nil {
		t.Errorf("clean branch without upstream: %+v, %v", g, err)
	}
	git(t, repo, "checkout", "-q", "--detach", "HEAD")
	r, _, err = contextReport(t, repo)
	g = r.Projects[0].Git
	if err != nil || !g.Detached || g.Branch != "" || g.Head == "" || g.Unborn {
		t.Errorf("detached HEAD: %+v, %v", g, err)
	}
	empty := t.TempDir()
	git(t, empty, "init", "-q")
	git(t, empty, "symbolic-ref", "HEAD", "refs/heads/main")
	must(t, empty, "start", "--private")
	r, _, err = contextReport(t, empty)
	g = r.Projects[0].Git
	if err != nil || g.State != "available" || !g.Unborn || g.Branch != "main" || g.Head != "" {
		t.Errorf("unborn branch: %+v, %v", g, err)
	}
	plain := t.TempDir()
	must(t, plain, "start")
	r, _, err = contextReport(t, plain)
	g = r.Projects[0].Git
	if err != nil || g.State != "not-repository" || g.Clean != nil || len(g.Changes) != 0 {
		t.Errorf("non-Git project: %+v, %v", g, err)
	}
	if err := os.Rename(filepath.Join(empty, ".git"), filepath.Join(empty, ".git-backup")); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(empty, ".git"), "gitdir: missing-git-dir\n")
	r, _, err = contextReport(t, empty)
	g = r.Projects[0].Git
	if err == nil || !r.Incomplete || g.State != "unavailable" || g.Clean != nil || g.Error == "" || len(r.Problems) == 0 || len(r.Projects[0].Sources) == 0 {
		t.Errorf("Git failure hid data or claimed cleanliness: %+v, %v", r, err)
	}
}

func TestContextGitMergeConflict(t *testing.T) {
	repo := gitContextProject(t)
	git(t, repo, "checkout", "-qb", "other")
	write(t, filepath.Join(repo, "README.md"), "other\n")
	git(t, repo, "commit", "-qam", "other")
	git(t, repo, "checkout", "-q", "main")
	write(t, filepath.Join(repo, "README.md"), "main\n")
	git(t, repo, "commit", "-qam", "main")
	if out, err := exec.Command("git", "-C", repo, "-c", "commit.gpgsign=false", "merge", "--no-edit", "other").CombinedOutput(); err == nil {
		t.Fatalf("expected a merge conflict: %s", out)
	}
	r, _, err := contextReport(t, repo)
	g := r.Projects[0].Git
	changes := gitChanges(t, g)
	if err != nil || g.Conflicts != 1 || !changes["README.md"].Conflict || changes["README.md"].Status != "UU" || *g.Clean {
		t.Errorf("merge conflict was lost: %+v, %v", g, err)
	}
}

func TestContextGitScopeAndPrivateSubprojectWorktree(t *testing.T) {
	base := t.TempDir()
	repo, wt := filepath.Join(base, "repo"), filepath.Join(base, "wt")
	gitRepo(t, repo)
	write(t, filepath.Join(repo, "one/a.txt"), "one\n")
	write(t, filepath.Join(repo, "two/b.txt"), "two\n")
	git(t, repo, "add", "-A")
	git(t, repo, "commit", "-qm", "subprojects")
	must(t, repo, "start", "one", "--private")
	git(t, repo, "worktree", "add", "-q", wt)
	repo, _ = filepath.EvalSymlinks(repo)
	wt, _ = filepath.EvalSymlinks(wt)
	write(t, filepath.Join(repo, "one/main-only.txt"), "main checkout\n")
	write(t, filepath.Join(wt, "one/a.txt"), "worktree change\n")
	write(t, filepath.Join(wt, "two/b.txt"), "outside project\n")
	write(t, filepath.Join(wt, "one/worktree-only.txt"), "worktree\n")
	r, _, err := contextReport(t, filepath.Join(wt, "one"))
	g := r.Projects[0].Git
	changes := gitChanges(t, g)
	if err != nil || !r.Projects[0].Private || r.Projects[0].Dir != filepath.Join(repo, "one") || g.Directory != filepath.Join(wt, "one") || g.Root != wt {
		t.Fatalf("metadata and observed working copy were mixed: %+v, %v", r.Projects[0], err)
	}
	if len(changes) != 2 || changes["one/a.txt"].Status != " M" || changes["one/worktree-only.txt"].Status != "??" {
		t.Errorf("wrong project scope or wrong worktree: %+v", changes)
	}
}

func TestContextGitDoesNotRefreshIndexOrRunFsmonitor(t *testing.T) {
	repo := gitContextProject(t)
	// The bytes match the index, but timestamps do not. Ordinary git status
	// would refresh the index's stat cache; context must keep it untouched.
	when := time.Now().Add(-time.Hour)
	if err := os.Chtimes(filepath.Join(repo, "README.md"), when, when); err != nil {
		t.Fatal(err)
	}
	monitor := filepath.Join(repo, ".git/fsmonitor-test")
	write(t, monitor, "#!/bin/sh\n: > .git/fsmonitor-was-run\nexit 1\n")
	if err := os.Chmod(monitor, 0o755); err != nil {
		t.Fatal(err)
	}
	git(t, repo, "config", "core.fsmonitor", monitor)
	index := filepath.Join(repo, ".git/index")
	before := read(t, index)
	info, err := os.Stat(index)
	if err != nil {
		t.Fatal(err)
	}
	r, _, err := contextReport(t, repo)
	if err != nil || r.Projects[0].Git.Clean == nil || !*r.Projects[0].Git.Clean {
		t.Fatalf("unchanged file content was reported dirty: %+v, %v", r, err)
	}
	must(t, repo, "context")
	after, err := os.Stat(index)
	if err != nil || !reflect.DeepEqual(before, read(t, index)) || !info.ModTime().Equal(after.ModTime()) {
		t.Error("context refreshed or rewrote the Git index")
	}
	if exists(filepath.Join(repo, ".git/fsmonitor-was-run")) {
		t.Error("context invoked the configured fsmonitor hook")
	}
}
