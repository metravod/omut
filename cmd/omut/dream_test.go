package main

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// dreamLog is a log with n entries dated after an old dreamed:.
func dreamLog(n int) string {
	s := ""
	for i := 0; i < n; i++ {
		s += "- 2026-11-10 · t/t · work done\n"
	}
	return s
}

func TestDreamStandalone(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "hn")
	standalone(t, dir, time.Now().Format("2006-01-02"))
	write(t, filepath.Join(dir, ".omut/brief.md"), strings.Replace(fill(time.Now().Format("2006-01-02"), time.Now().Format("2006-01-02"), ""), "---\n## Why", "dreamed: 2026-11-01\n---\n## Why", 1))
	write(t, filepath.Join(dir, ".omut/log.md"), dreamLog(3))

	out, err := omut(t, dir, "dream")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"# omut dream: hn (report)", "Memory: 3 records since the last dream (2026-11-01)",
		"- 2026-11-10 · log · `.omut/log.md:", "## The dream contract", "Change\nnothing until they approve",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("dream has no %q:\n%s", want, out)
		}
	}

	out, err = omut(t, dir, "dream", "--apply")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "(apply)") || !strings.Contains(out, "the human has approved") {
		t.Errorf("--apply contract:\n%s", out)
	}

	out, err = omut(t, dir, "dream", "--json")
	if err != nil {
		t.Fatal(err)
	}
	var report struct {
		Scope     string   `json:"scope"`
		Mode      string   `json:"mode"`
		Dreamed   string   `json:"dreamed"`
		Threshold int      `json:"threshold"`
		Records   []struct{ Kind, Date string } `json:"records"`
		Contract  string   `json:"contract"`
	}
	if err := json.Unmarshal([]byte(out), &report); err != nil {
		t.Fatal(err)
	}
	if report.Scope != "project" || report.Mode != "report" || report.Dreamed != "2026-11-01" || report.Threshold != 10 {
		t.Errorf("JSON: %+v", report)
	}
	if len(report.Records) != 3 || report.Records[0].Kind != "log" {
		t.Errorf("JSON records: %+v", report.Records)
	}
	if !strings.Contains(report.Contract, "15 lines") {
		t.Errorf("JSON contract: %s", report.Contract)
	}

	// The dream never writes.
	before := entries(t, dir)
	if _, err := omut(t, dir, "dream", "--apply", "--json"); err != nil {
		t.Fatal(err)
	}
	if got := entries(t, dir); len(got) != len(before) {
		t.Errorf("dream wrote: %v", got)
	}
}

func TestDreamScopes(t *testing.T) {
	root := t.TempDir()
	must(t, root, "init", "--by", "t/t")
	must(t, root, "start", "hn", "--by", "t/t")
	proj := filepath.Join(root, "hn")
	write(t, filepath.Join(proj, ".omut/brief.md"), fill(time.Now().Format("2006-01-02"), time.Now().Format("2006-01-02"), ""))
	write(t, filepath.Join(proj, "docs/notes.md"), "n\n")
	write(t, filepath.Join(proj, ".omut/log.md"), dreamLog(2))
	write(t, filepath.Join(root, ".omut/decisions.md"), "# Decisions\n\n- D1 [open 2026-11-10] pack question\n")

	out, err := omut(t, proj, "dream")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "# omut dream: hn") || !strings.Contains(out, "2 records") {
		t.Errorf("project dream from inside:\n%s", out)
	}
	out, err = omut(t, proj, "dream", "..")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "pack") || !strings.Contains(out, "D1 [open 2026-11-10]") {
		t.Errorf("pack dream from a project:\n%s", out)
	}
	out, err = omut(t, root, "dream")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "dream <project>") {
		t.Errorf("pack dream routes nowhere:\n%s", out)
	}

	// A shape error makes the report incomplete: partial output, exit 1.
	write(t, filepath.Join(root, ".omut/decisions.md"), "# Decisions\n\n- D1 [maybe] broken\n")
	if _, err := omut(t, root, "dream"); err == nil {
		t.Error("broken meta did not fail the dream")
	}
	if _, err := omut(t, t.TempDir(), "dream"); err == nil {
		t.Error("no meta: dream must refuse")
	}
}

func TestHookDreamReminderAndPreview(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "hn")
	standalone(t, dir, time.Now().Format("2006-01-02"))
	// Below the threshold: silent.
	if out, _, _ := hookRun(t, dir, "", "start"); strings.Contains(out, "Memory wants a dream") {
		t.Errorf("reminder under threshold:\n%s", out)
	}
	write(t, filepath.Join(dir, ".omut/log.md"), dreamLog(10))
	if out, _, err := hookRun(t, dir, "", "start"); err != nil || !strings.Contains(out, "## Memory wants a dream") || !strings.Contains(out, "10 records are newer than the memory has never been recorded as dreamed") {
		t.Errorf("no reminder at threshold:\n%s", out)
	}
	// The project's own threshold, and the extinguishing.
	brief := strings.Replace(fill(time.Now().Format("2006-01-02"), time.Now().Format("2006-01-02"), ""), "---\n## Why", "dreamed: 2026-11-05\ndream: 3\n---\n## Why", 1)
	write(t, filepath.Join(dir, ".omut/brief.md"), brief)
	if out, _, _ := hookRun(t, dir, "", "start"); !strings.Contains(out, "## Memory wants a dream") || !strings.Contains(out, "the last dream was recorded 2026-11-05") {
		t.Errorf("reminder with dreamed/dream:\n%s", out)
	}
	write(t, filepath.Join(dir, ".omut/brief.md"), strings.Replace(brief, "dreamed: 2026-11-05", "dreamed: 2026-11-11", 1))
	if out, _, _ := hookRun(t, dir, "", "start"); strings.Contains(out, "Memory wants a dream") {
		t.Errorf("reminder did not extinguish:\n%s", out)
	}
	// A frozen project's memory is not growing.
	write(t, filepath.Join(dir, ".omut/brief.md"), strings.Replace(brief, "status: active", "status: frozen", 1))
	if out, _, _ := hookRun(t, dir, "", "start"); strings.Contains(out, "Memory wants a dream") {
		t.Errorf("frozen project reminded:\n%s", out)
	}

	// --preview prints what a session would get, without a payload, and
	// ignores one when given anyway.
	out, _, err := hookRun(t, dir, "", "start", "--preview")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "# omut: the meta of hn") || strings.Contains(out, "Memory wants a dream") {
		t.Errorf("preview: %s", out)
	}
	if out, _, err := hookRun(t, filepath.Join(dir, "docs"), `{"cwd": "`+dir+`"}`, "start", "--agent", "claude", "--preview"); err != nil || !strings.Contains(out, "the meta of hn") {
		t.Errorf("preview with a payload:\n%s", out)
	}
	if _, _, err := hookRun(t, dir, "", "commit", "--preview"); err == nil {
		t.Error("commit --preview must be refused")
	}
}
