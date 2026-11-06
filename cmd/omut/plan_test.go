package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func planPointer(t *testing.T, owner, ref string) {
	t.Helper()
	text := read(t, owner)
	write(t, owner, strings.Replace(text, "---\n", "---\nplan: "+ref+"\n", 1))
}

func TestPlansSelectionProvenanceAndNoWrites(t *testing.T) {
	root := t.TempDir()
	must(t, root, "init")
	dir := contextProject(t, root, "alpha")
	brief := filepath.Join(dir, ".omut/brief.md")
	instance := filepath.Join(dir, ".omut/local/instance.md")
	planPointer(t, brief, "docs/work plan.md")
	planPointer(t, instance, "work/README.md")
	shared := "# Existing plan\r\n\r\n## Goal: reliable answers\r\n| Task | State |\r\n| --- | --- |\r\n| [A](cards/a.md) | done |\r\n| B | todo |\r\n"
	local := "# LOCAL-PLAN-ONLY\n[Card](../cards/b.md)\n"
	sharedPath := filepath.Join(dir, "docs/work plan.md")
	localPath := filepath.Join(dir, ".omut/local/work/README.md")
	write(t, sharedPath, shared)
	write(t, localPath, local)
	// A plan is not a recursive include. Even a broken card link is left to
	// the existing plan's owner; cards may use another tracker or format.
	card := filepath.Join(dir, "docs/cards/a.md")
	write(t, card, "CARD-CONTENT-MUST-NOT-BE-INJECTED\n")
	before := map[string]string{}
	for _, path := range []string{sharedPath, localPath, brief, instance, card} {
		info, _ := os.Stat(path)
		before[path] = read(t, path) + info.ModTime().String()
	}
	r, out, err := contextReport(t, root)
	if err != nil || !strings.Contains(out, "reliable answers") || strings.Contains(out, "LOCAL-PLAN-ONLY") || strings.Contains(out, "CARD-CONTENT") {
		t.Fatalf("overview selection: %v\n%s", err, out)
	}
	assertContextSources(t, r, must(t, root, "context"))
	r, out, err = contextReport(t, dir)
	if err != nil || !strings.Contains(out, "LOCAL-PLAN-ONLY") || !strings.Contains(out, "reliable answers") || strings.Contains(out, "CARD-CONTENT") {
		t.Fatalf("project selection: %v\n%s", err, out)
	}
	assertContextSources(t, r, must(t, dir, "context"))
	for _, scope := range []string{root, dir} {
		hook := must(t, scope, "hook", "start")
		if !strings.Contains(hook, "reliable answers") || strings.Contains(hook, "LOCAL-PLAN-ONLY") != (scope == dir) {
			t.Fatalf("hook plan selection: %s", hook)
		}
	}
	must(t, dir, "check")
	must(t, dir, "zero")
	for path, want := range before {
		info, _ := os.Stat(path)
		if read(t, path)+info.ModTime().String() != want {
			t.Errorf("reading/zero changed %s", path)
		}
	}
	write(t, brief, strings.Replace(read(t, brief), "status: active", "status: frozen", 1))
	_, out, err = contextReport(t, root)
	if err != nil || strings.Contains(out, "reliable answers") {
		t.Fatalf("frozen plan entered overview: %v\n%s", err, out)
	}
}

func TestInvalidPlanPointersFailExplicitlyWithoutReadingTargets(t *testing.T) {
	for _, tc := range []string{"missing.md", "../outside.md", "absolute", "directory.md", "alias.md", ".omut/local/private.md", ".omut/LOCAL/private.md", "local-alias.md", "not-markdown.txt"} {
		t.Run(tc, func(t *testing.T) {
			dir := t.TempDir()
			standalone(t, dir, time.Now().Format("2006-01-02"))
			outside := filepath.Join(t.TempDir(), "outside.md")
			write(t, outside, "DO-NOT-READ-OUTSIDE\n")
			private := filepath.Join(dir, ".omut/local/private.md")
			write(t, private, "DO-NOT-READ-LOCAL\n")
			if err := os.Mkdir(filepath.Join(dir, "directory.md"), 0o755); err != nil {
				t.Fatal(err)
			}
			for alias, target := range map[string]string{"alias.md": outside, "local-alias.md": private} {
				if err := os.Symlink(target, filepath.Join(dir, alias)); err != nil {
					t.Fatal(err)
				}
			}
			ref := tc
			if tc == "absolute" {
				ref = outside
			}
			planPointer(t, filepath.Join(dir, ".omut/brief.md"), ref)
			found := false
			for _, f := range findings(t, dir) {
				found = found || (f.Rule == "shape" && f.Level == "error" && strings.HasPrefix(f.Message, "plan:"))
			}
			if !found {
				t.Fatal("bad plan pointer passed check")
			}
			r, out, err := contextReport(t, dir)
			if err == nil || !r.Incomplete || len(r.Problems) == 0 || strings.Contains(out, "DO-NOT-READ") {
				t.Fatalf("bad plan silently omitted or read: %v\n%s", err, out)
			}
			if hook := must(t, dir, "hook", "start"); !strings.Contains(hook, "Incomplete plan context") || strings.Contains(hook, "DO-NOT-READ") {
				t.Fatalf("hook hid invalid pointer: %s", hook)
			}
		})
	}
}

func TestLocalPlanCannotEscapeItsLayer(t *testing.T) {
	dir := t.TempDir()
	standalone(t, dir, time.Now().Format("2006-01-02"))
	owner := filepath.Join(dir, ".omut/local/instance.md")
	write(t, owner, "---\nupdated: 2026-11-06\nby: test/model\n---\n")
	write(t, filepath.Join(dir, "outside.md"), "OUTSIDE-LOCAL-LAYER\n")
	if err := os.Symlink(filepath.Join(dir, "outside.md"), filepath.Join(dir, ".omut/local/alias.md")); err != nil {
		t.Fatal(err)
	}
	planPointer(t, owner, "alias.md")
	r, out, err := contextReport(t, dir)
	if err == nil || !r.Incomplete || strings.Contains(out, "OUTSIDE-LOCAL-LAYER") {
		t.Fatalf("local plan escaped: %v\n%s", err, out)
	}
}

func TestPlanSizeWarnsWithoutTruncation(t *testing.T) {
	dir := t.TempDir()
	standalone(t, dir, time.Now().Format("2006-01-02"))
	planPointer(t, filepath.Join(dir, ".omut/brief.md"), "work.md")
	for _, text := range []string{strings.Repeat("я", 8001), strings.Repeat("task\n", 81)} {
		write(t, filepath.Join(dir, "work.md"), text)
		if _, err := omut(t, dir, "check", "--fail-on", "warn"); err == nil {
			t.Fatal("oversized plan passed strict check")
		}
		r, _, err := contextReport(t, dir)
		if err != nil {
			t.Fatal(err)
		}
		found := false
		for _, source := range r.Projects[0].Sources {
			if source.Kind == "plan" {
				found = source.Text == strings.TrimSuffix(text, "\n")
			}
		}
		if !found || !strings.Contains(fmt.Sprint(r.Findings), "size") {
			t.Fatal("oversized plan was truncated, omitted or not flagged")
		}
	}
}
