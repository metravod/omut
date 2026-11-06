package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/metravod/omut/internal/context"
)

func contextReport(t *testing.T, dir string, args ...string) (context.Report, string, error) {
	t.Helper()
	out, err := omut(t, dir, append([]string{"context", "--json"}, args...)...)
	var exit exitCode
	if err != nil && !errors.As(err, &exit) {
		t.Fatalf("context: %v\n%s", err, out)
	}
	var report context.Report
	if e := json.Unmarshal([]byte(out), &report); e != nil {
		t.Fatalf("context JSON: %v\n%s", e, out)
	}
	return report, out, err
}

func contextProject(t *testing.T, root, name string) string {
	t.Helper()
	must(t, root, "start", name)
	dir := filepath.Join(root, name)
	today := time.Now().Format("2006-01-02")
	write(t, filepath.Join(dir, ".omut/brief.md"), fill(today, today, "PUBLIC-FOOTPRINT"))
	write(t, filepath.Join(dir, "docs/notes.md"), "# notes\n")
	return dir
}

func contextThread(status, reason string) string {
	return fmt.Sprintf("---\nstatus: %s\nupdated: 2020-01-01\nby: test/model\nparked_because: %s\n---\n# Shipping\n**Goal:** SHIP-GOAL\n**Where we got:** DETAIL-ONLY\n**Verified:** tested once\n**Not verified:** delivery\n**Open:** CHOOSE-TRANSPORT\n  include a multiline question\n**Next step:** SEND-RELEASE\n", status, reason)
}

func contextText(p context.Project) string {
	var b strings.Builder
	for _, s := range p.Sources {
		b.WriteString(s.Text + "\n")
	}
	return b.String()
}

// Check actual source contents and line numbers, rather than reconstructing the
// collector's selection logic. JSON excerpts must also reach the text output.
func assertContextSources(t *testing.T, r context.Report, markdown string) {
	t.Helper()
	projects := append([]context.Project{}, r.Projects...)
	if r.Pack != nil {
		projects = append(projects, *r.Pack)
	}
	for _, p := range projects {
		for _, s := range p.Sources {
			text := strings.ReplaceAll(read(t, filepath.Join(r.Root, filepath.FromSlash(s.File))), "\r\n", "\n")
			lines := strings.Split(strings.TrimSuffix(text, "\n"), "\n")
			end := s.Line + strings.Count(s.Text, "\n")
			if s.Line < 1 || end > len(lines) || strings.Join(lines[s.Line-1:end], "\n") != s.Text {
				t.Errorf("excerpt differs from %s:%d: %q", s.File, s.Line, s.Text)
			}
			if !strings.Contains(markdown, s.Text) || !strings.Contains(markdown, fmt.Sprintf("%s:%d", s.File, s.Line)) {
				t.Errorf("text output lost source %s:%d", s.File, s.Line)
			}
		}
	}
}

func TestContextPackSelectionAndProjectExpansion(t *testing.T) {
	root := t.TempDir()
	must(t, root, "init")
	dir := contextProject(t, root, "alpha")
	write(t, filepath.Join(root, ".omut/decisions.md"), "# Decisions\n- D1 [accepted ?] PACK-CONSTRAINT\n- D2 [open] PACK-QUESTION\n")
	write(t, filepath.Join(dir, ".omut/decisions.md"), "# Decisions\n- D1 [accepted ?] ACCEPTED-DETAIL\n- D2 [open 2026-11-01] PENDING-CHOICE\n- D3 [doubted] REVISIT-CHOICE\n<!-- - D4 [open] COMMENT-ONLY -->\n")
	write(t, filepath.Join(dir, ".omut/threads/ship.md"), contextThread("parked", "blocked"))
	write(t, filepath.Join(dir, ".omut/threads/active.md"), contextThread("active", ""))
	write(t, filepath.Join(dir, ".omut/threads/done.md"), strings.ReplaceAll(contextThread("done", ""), "SHIP-GOAL", "DONE-ONLY"))
	write(t, filepath.Join(dir, ".omut/threads/dropped.md"), strings.ReplaceAll(contextThread("dropped", ""), "SHIP-GOAL", "DROPPED-ONLY"))
	brief := filepath.Join(dir, ".omut/brief.md")
	write(t, brief, strings.Replace(read(t, brief), "status: active", "status: active\nadr: docs/adr", 1))
	write(t, filepath.Join(dir, "docs/adr/ADR-001-proposed.md"), "# Transport\n\nStatus: Proposed\n\nADR-REASON\n")
	write(t, filepath.Join(dir, "docs/adr/002-tentative.md"), "# Retry\n\nStatus: Accepted\nFirmness: tentative\n")
	write(t, filepath.Join(dir, "docs/adr/003-accepted.md"), "# ACCEPTED-ADR-ONLY\nStatus: Accepted\n")
	write(t, filepath.Join(dir, "docs/adr/README.md"), "# NON-ADR-ONLY\nStatus: Proposed\n")
	instance := filepath.Join(dir, ".omut/local/instance.md")
	write(t, instance, strings.Replace(read(t, instance), "## Runbook", "## Runbook\nLOCAL-ONLY", 1))
	var log strings.Builder
	for i := 7; i >= 1; i-- {
		fmt.Fprintf(&log, "- 2026-11-0%d · test/model · LOG-%d\n", i, i)
	}
	write(t, filepath.Join(dir, ".omut/log.md"), log.String())
	write(t, filepath.Join(dir, ".omut/local/log.md"), "- 2026-11-01 · test/model · LOCAL-EVENT\n")
	frozen := contextProject(t, root, "zeta")
	must(t, frozen, "freeze")
	write(t, filepath.Join(frozen, ".omut/decisions.md"), "# Decisions\n- D1 [open] FROZEN-CHOICE\n")
	write(t, filepath.Join(frozen, ".omut/threads/old.md"), contextThread("active", ""))

	r, first, err := contextReport(t, root)
	if err != nil || r.Incomplete || r.Scope != "pack" || r.Pack == nil || len(r.Projects) != 2 {
		t.Fatalf("pack: %+v, %v", r, err)
	}
	if r.Projects[0].Name != "alpha" || r.Projects[1].Name != "zeta" || r.Projects[1].Status != "frozen" {
		t.Errorf("project order or frozen state: %+v", r.Projects)
	}
	for _, want := range []string{"PACK-CONSTRAINT", "PACK-QUESTION", "PENDING-CHOICE", "REVISIT-CHOICE", "SHIP-GOAL", "CHOOSE-TRANSPORT", "include a multiline question", "SEND-RELEASE", "# Transport", "# Retry", "stalled-thread"} {
		if !strings.Contains(first, want) {
			t.Errorf("overview lost %q", want)
		}
	}
	for _, unwanted := range []string{"ACCEPTED-DETAIL", "COMMENT-ONLY", "DETAIL-ONLY", "DONE-ONLY", "DROPPED-ONLY", "ADR-REASON", "ACCEPTED-ADR-ONLY", "NON-ADR-ONLY", "LOCAL-ONLY", "LOCAL-EVENT", "LOG-7", "PUBLIC-FOOTPRINT", "FROZEN-CHOICE"} {
		if strings.Contains(first, unwanted) {
			t.Errorf("overview includes %q", unwanted)
		}
	}
	_, second, _ := contextReport(t, root)
	if first != second {
		t.Error("unchanged input produced different JSON")
	}
	markdown := must(t, root, "context")
	assertContextSources(t, r, markdown)
	if !strings.Contains(markdown, "parked: blocked") || !strings.Contains(markdown, "test/model") || !strings.Contains(markdown, "omitted") {
		t.Error("overview lost selection policy, thread owner or blocker")
	}
	full, jsonText, err := contextReport(t, root, "alpha")
	if err != nil || full.Scope != "project" || len(full.Projects) != 1 {
		t.Fatalf("expanded: %+v, %v", full, err)
	}
	for _, want := range []string{"PUBLIC-FOOTPRINT", "ACCEPTED-DETAIL", "DETAIL-ONLY", "ADR-REASON", "LOCAL-ONLY", "LOCAL-EVENT", "LOG-7", "LOG-3"} {
		if !strings.Contains(jsonText, want) {
			t.Errorf("expanded context lost %q", want)
		}
	}
	for _, unwanted := range []string{"DONE-ONLY", "DROPPED-ONLY", "ACCEPTED-ADR-ONLY", "LOG-2", "LOG-1"} {
		if strings.Contains(jsonText, unwanted) {
			t.Errorf("expanded context includes %q", unwanted)
		}
	}
	assertContextSources(t, full, must(t, root, "context", "alpha"))
	deep := filepath.Join(dir, "src/pkg")
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatal(err)
	}
	nested, _, err := contextReport(t, deep)
	if err != nil || nested.Root != full.Root || contextText(nested.Projects[0]) != contextText(full.Projects[0]) {
		t.Errorf("context from a nested directory changed scope: %v", err)
	}
	whole, _, err := contextReport(t, dir, "..")
	if err != nil || whole.Scope != "pack" || whole.Root != r.Root {
		t.Errorf(".. did not select the pack: %+v, %v", whole, err)
	}
	hook, _, err := hookRun(t, root, "", "start")
	if err != nil || !strings.Contains(hook, must(t, root, "context")) {
		t.Error("pack hook did not supply the command's assembled overview")
	}
}

func TestContextIncompletePreservesReadableRecords(t *testing.T) {
	root := t.TempDir()
	must(t, root, "init")
	dir := contextProject(t, root, "healthy")
	write(t, filepath.Join(root, "unadopted/README.md"), "no meta\n")
	write(t, filepath.Join(dir, ".omut/decisions.md"), "# Decisions\n- D1 [maybe] MALFORMED-CHOICE\n")
	write(t, filepath.Join(dir, ".omut/threads/unknown.md"), "---\nstatus: unknown\n---\nUNCLASSIFIED-WORK\n")
	if err := os.Symlink("gone.md", filepath.Join(dir, ".omut/threads/missing.md")); err != nil {
		t.Fatal(err)
	}
	r, out, err := contextReport(t, root)
	if err == nil || !r.Incomplete || len(r.Problems) == 0 || len(r.Projects) != 2 {
		t.Fatalf("failures were hidden: %+v, %v", r, err)
	}
	for _, want := range []string{"MALFORMED-CHOICE", "UNCLASSIFIED-WORK", "unadopted", "missing.md", "An HN parser"} {
		if !strings.Contains(out, want) {
			t.Errorf("partial report lost %q", want)
		}
	}
	text, err := omut(t, root, "context")
	if err == nil || !strings.Contains(text, "Incomplete context") {
		t.Error("text report hides incomplete collection")
	}
	// An unreadable ADR was silently ignored by the old ADR enumeration.
	brief := filepath.Join(dir, ".omut/brief.md")
	write(t, brief, strings.Replace(read(t, brief), "status: active", "status: active\nadr: docs/adr", 1))
	if err := os.MkdirAll(filepath.Join(dir, "docs/adr"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("gone.md", filepath.Join(dir, "docs/adr/001-missing.md")); err != nil {
		t.Fatal(err)
	}
	full, _, err := contextReport(t, dir)
	if err == nil || !strings.Contains(fmt.Sprint(full.Problems), "001-missing.md") {
		t.Error("ADR read failure was hidden")
	}
	if err := os.Symlink("gone", filepath.Join(root, "broken-project")); err != nil {
		t.Fatal(err)
	}
	r, _, err = contextReport(t, root)
	if err == nil || !strings.Contains(fmt.Sprint(r.Problems), "broken-project") {
		t.Error("uninspectable project entry disappeared from discovery")
	}
}

func TestContextDiscoveryAndReadOnly(t *testing.T) {
	root := t.TempDir()
	must(t, root, "init")
	contextProject(t, root, "alpha")
	write(t, filepath.Join(root, "ignored/item"), "ignored\n")
	write(t, filepath.Join(root, ".hidden/item"), "hidden\n")
	brief := filepath.Join(root, ".omut/brief.md")
	write(t, brief, strings.Replace(read(t, brief), "ignore: []", "ignore:\n  - ignored", 1))
	external := t.TempDir()
	must(t, external, "start")
	if err := os.Symlink(external, filepath.Join(root, "linked")); err != nil {
		t.Fatal(err)
	}
	snapshot := func() map[string]string {
		out := map[string]string{}
		for _, path := range entries(t, root) {
			info, err := os.Lstat(path)
			if err != nil {
				t.Fatal(err)
			}
			value := info.Mode().String() + info.ModTime().String()
			if info.Mode()&os.ModeSymlink != 0 {
				target, _ := os.Readlink(path)
				value += target
			} else if !info.IsDir() {
				value += read(t, path)
			}
			out[path] = value
		}
		return out
	}
	before := snapshot()
	r, _, err := contextReport(t, root)
	if err != nil || len(r.Projects) != 2 || r.Projects[0].Name != "alpha" || r.Projects[1].Name != "linked" {
		t.Fatalf("directory discovery: %+v, %v", r.Projects, err)
	}
	must(t, root, "context")
	linked, _, err := contextReport(t, root, "linked")
	if err != nil || linked.Projects[0].Name != "linked" {
		t.Error("an explicit symlinked project lost its pack name")
	}
	if after := snapshot(); !reflect.DeepEqual(before, after) {
		t.Error("context changed files, metadata or timestamps")
	}
	// A directory with no records must not inherit an unrelated parent's meta.
	for _, args := range [][]string{{"context", "missing"}, {"context", "alpha", "extra"}, {"context", "ignored"}} {
		if _, err := omut(t, root, args...); err == nil {
			t.Errorf("%v should fail", args)
		}
	}
	outside := t.TempDir()
	if _, err := omut(t, outside, "context"); err == nil {
		t.Error("context without meta should fail")
	}
	fromOutside, _, err := contextReport(t, outside, external)
	if err != nil || fromOutside.Scope != "project" {
		t.Errorf("explicit project path: %v", err)
	}
	if _, err := omut(t, outside, "context", filepath.Join(external, "AGENTS.md")); err == nil {
		t.Error("a file is not a project directory")
	}
	// A broken optional instance file must not be confused with an absent layer.
	if err := os.Remove(filepath.Join(external, ".omut/local/instance.md")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("gone", filepath.Join(external, ".omut/local/instance.md")); err != nil {
		t.Fatal(err)
	}
	partial, _, err := contextReport(t, external)
	if err == nil || !strings.Contains(fmt.Sprint(partial.Problems), "instance.md") {
		t.Error("a dangling optional instance file was silently omitted")
	}
}

func TestContextWorktreeLayers(t *testing.T) {
	for _, private := range []bool{false, true} {
		t.Run(fmt.Sprintf("private=%v", private), func(t *testing.T) {
			base := t.TempDir()
			repo, wt := filepath.Join(base, "repo"), filepath.Join(base, "wt")
			gitRepo(t, repo)
			args := []string{"start"}
			if private {
				args = append(args, "--private")
			}
			must(t, repo, args...)
			today := time.Now().Format("2006-01-02")
			write(t, filepath.Join(repo, ".omut/brief.md"), fill(today, today, ""))
			write(t, filepath.Join(repo, "docs/notes.md"), "# notes\n")
			instance := filepath.Join(repo, ".omut/local/instance.md")
			write(t, instance, strings.Replace(read(t, instance), "## Runbook", "## Runbook\nMAIN-CHECKOUT-ONLY", 1))
			planPointer(t, filepath.Join(repo, ".omut/brief.md"), "docs/work.md")
			write(t, filepath.Join(repo, "docs/work.md"), "MAIN-PROJECT-PLAN\n")
			planPointer(t, instance, "work/README.md")
			write(t, filepath.Join(repo, ".omut/local/work/README.md"), "MAIN-LOCAL-PLAN\n")
			git(t, repo, "add", "-A")
			git(t, repo, "commit", "-qm", "meta")
			git(t, repo, "worktree", "add", "-q", wt)
			repo, _ = filepath.EvalSymlinks(repo)
			wt, _ = filepath.EvalSymlinks(wt)
			if !private {
				path := filepath.Join(wt, ".omut/brief.md")
				write(t, path, strings.Replace(read(t, path), "Works.", "BRANCH-ONLY", 1))
				write(t, filepath.Join(wt, "docs/work.md"), "BRANCH-PROJECT-PLAN\n")
			}
			r, out, err := contextReport(t, wt)
			if err != nil || r.Scope != "project" || len(r.Projects) != 1 {
				t.Fatalf("worktree context: %+v, %v", r, err)
			}
			p := r.Projects[0]
			if p.Git == nil || p.Git.State != "available" || p.Git.Directory != wt || p.Git.Root != wt {
				t.Errorf("Git observation did not come from the actual worktree: %+v", p.Git)
			}
			if p.Private != private || !strings.Contains(out, "MAIN-CHECKOUT-ONLY") || p.Instance != repo {
				t.Errorf("lost main-checkout memory or visibility: %+v", p)
			}
			if private {
				if p.Dir != repo || p.Worktree != wt || exists(filepath.Join(wt, ".omut")) {
					t.Error("private memory was not resolved to the main checkout")
				}
			} else if p.Dir != wt || !strings.Contains(out, "BRANCH-ONLY") || exists(filepath.Join(wt, ".omut/local")) {
				t.Error("branch and instance layers were mixed")
			}
			wantPlan := "BRANCH-PROJECT-PLAN"
			if private {
				wantPlan = "MAIN-PROJECT-PLAN"
			}
			if !strings.Contains(out, wantPlan) || !strings.Contains(out, "MAIN-LOCAL-PLAN") {
				t.Errorf("plan layers mixed: %s", out)
			}
			if hook := must(t, wt, "hook", "start"); !strings.Contains(hook, wantPlan) || !strings.Contains(hook, "MAIN-LOCAL-PLAN") {
				t.Errorf("hook lost worktree plans: %s", hook)
			}
			assertContextSources(t, r, must(t, wt, "context"))
			for _, f := range r.Findings {
				if f.File != "" && !exists(filepath.Join(r.Root, filepath.FromSlash(f.File))) {
					t.Errorf("finding has the wrong base: %+v", f)
				}
			}
		})
	}
}
