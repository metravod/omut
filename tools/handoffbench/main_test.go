package main

import (
	"archive/tar"
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestSnapshotExcludesAnswersAndProjectMemory(t *testing.T) {
	var b bytes.Buffer
	w := tar.NewWriter(&b)
	for _, name := range []string{"go.mod", "internal/check/check.go", "docs/spec-v0.md", "templates/project/block.md", ".omut/brief.md", "docs/handoff.md", "docs/research/results.md", "tools/handoffbench/oracle.go", "AGENTS.md"} {
		data := []byte("fixture")
		if err := w.WriteHeader(&tar.Header{Name: name, Mode: 0600, Size: int64(len(data)), Typeflag: tar.TypeReg}); err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write(data); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	files, err := snapshot(b.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 4 {
		t.Fatalf("answer leakage or missing source: %v", files)
	}
	for _, name := range []string{"go.mod", "internal/check/check.go", "docs/spec-v0.md", "templates/project/block.md"} {
		if files[name] == nil {
			t.Errorf("missing %s", name)
		}
	}
}

func TestMutationRequiresAnAssertionFailure(t *testing.T) {
	for _, tc := range []struct {
		name, log string
		failed    bool
	}{
		{"assertion", "{\"Action\":\"fail\",\"Test\":\"TestBudget/CRLF\"}\n", true},
		{"compile error", "{\"Action\":\"fail\",\"Package\":\"fixture\"}\n", false},
		{"pass", "{\"Action\":\"pass\",\"Test\":\"TestBudget\"}\n", false},
		{"process error", "sandbox unavailable\n", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := assertionFailed([]byte(tc.log)); got != tc.failed {
				t.Fatalf("assertionFailed=%v, want %v", got, tc.failed)
			}
		})
	}
}

func TestAccountingRejectsIncompleteSessionsAndCountsCacheAsSubset(t *testing.T) {
	for _, tc := range []struct {
		name, suffix string
		complete     bool
	}{
		{"completed", "", true},
		{"failed after completion", "{\"type\":\"turn.failed\"}\n", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "events.jsonl")
			log := `{"type":"item.completed","item":{"type":"command_execution"}}
{"type":"turn.completed","usage":{"input_tokens":100,"cached_input_tokens":80,"output_tokens":10,"reasoning_output_tokens":3}}
` + tc.suffix
			if err := os.WriteFile(path, []byte(log), 0600); err != nil {
				t.Fatal(err)
			}
			var s stage
			if err := readEvents(path, &s); err != nil {
				t.Fatal(err)
			}
			if s.Completed != tc.complete || s.Commands != 1 || s.Usage != (usage{100, 80, 10, 3}) {
				t.Fatalf("incorrect stage: %+v", s)
			}
			v := cell{Status: "scored", Source: &s, Receiver: &s, Checks: []checkResult{{Name: "gate", Passed: true}}}
			finishCell(&v)
			if v.Success != tc.complete {
				t.Fatalf("incomplete session counted as success: %+v", v)
			}
			if v.TotalUsage != (usage{200, 160, 20, 6}) {
				t.Fatalf("incorrect combined accounting: %+v", v.TotalUsage)
			}
		})
	}
	path := filepath.Join(t.TempDir(), "incomplete.jsonl")
	if err := os.WriteFile(path, []byte("{\"type\":\"thread.started\"}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	var s stage
	if err := readEvents(path, &s); err != nil || s.Completed {
		t.Fatalf("missing usage/completion must not look successful: %+v, %v", s, err)
	}
}

func TestScopeRejectsWeakeningExistingTestsAndCrossConditionMemory(t *testing.T) {
	dir := t.TempDir()
	for name, body := range map[string]string{"cmd/omut/main_test.go": "original", "cmd/omut/size_test.go": "original", "internal/check/check.go": "original"} {
		if err := write(dir, name, []byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	before, err := tree(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := write(dir, "cmd/omut/size_test.go", []byte("new coverage")); err != nil {
		t.Fatal(err)
	}
	if !unchanged(before, dir, "scope", "tests").Passed {
		t.Fatal("legitimate task edits rejected")
	}
	if !unchanged(before, dir, "scope", "repair").Passed {
		t.Fatal("repair's legitimate regression coverage rejected")
	}
	if unchanged(before, dir, "source scope", "").Passed {
		t.Fatal("outgoing review was allowed to change product tests")
	}
	if err := write(dir, "internal/check/check.go", []byte("repaired")); err != nil {
		t.Fatal(err)
	}
	if !unchanged(before, dir, "scope", "repair").Passed || unchanged(before, dir, "scope", "tests").Passed {
		t.Fatal("repair and test-only production scopes were confused")
	}
	if err := write(dir, "cmd/omut/main_test.go", []byte("weakened")); err != nil {
		t.Fatal(err)
	}
	if unchanged(before, dir, "scope", "tests").Passed || unchanged(before, dir, "scope", "repair").Passed {
		t.Fatal("unrelated test edits were accepted")
	}
	if err := write(dir, "handoff.md", []byte("carry context")); err != nil {
		t.Fatal(err)
	}
	if memoryScope(dir, "none").Passed || !memoryScope(dir, "note").Passed {
		t.Fatal("condition boundary not enforced")
	}
}

func TestRescoreRefusesUnfinishedRunWithoutTouchingIt(t *testing.T) {
	dir := t.TempDir()
	m := manifest{Cells: []cell{{ID: "cell-001", Status: "source-complete"}}}
	path := filepath.Join(dir, "manifest.json")
	if err := jsonFile(path, &m); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(dir, "rescore")
	if err := rescoreRun(dir, output); err == nil {
		t.Fatal("accepted an actively running experiment")
	}
	if _, err := os.Stat(output); !os.IsNotExist(err) {
		t.Fatal("created outputs for an unfinished run")
	}
	after, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("changed original evidence")
	}
}

func TestRescoreUsesOriginalInstructions(t *testing.T) {
	dir := t.TempDir()
	if err := write(dir, "AGENTS.md", []byte("original instructions\n")); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"init", "-q"}, {"add", "."}, {"commit", "-qm", "Controlled source snapshot"}} {
		if _, err := git(dir, args...); err != nil {
			t.Fatal(err)
		}
	}
	if err := write(dir, "AGENTS.md", []byte("changed instructions\n")); err != nil {
		t.Fatal(err)
	}
	got, err := initialAgents(dir)
	if err != nil || string(got) != "original instructions\n" {
		t.Fatalf("rescore trusted changed instructions: %q, %v", got, err)
	}
}
