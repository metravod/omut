package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// hookRun calls omut hook with the given payload on stdin and returns
// stdout, stderr and the error.
func hookRun(t *testing.T, dir, payload string, args ...string) (string, string, error) {
	t.Helper()
	t.Chdir(dir)
	in, err := os.CreateTemp(t.TempDir(), "stdin")
	if err != nil {
		t.Fatal(err)
	}
	in.WriteString(payload)
	in.Seek(0, 0)
	defer in.Close()
	var out, errOut bytes.Buffer
	err = cmdHook(args, &out, in, &errOut)
	return out.String(), errOut.String(), err
}

func TestHookStart(t *testing.T) {
	root := t.TempDir()
	must(t, root, "init")
	must(t, root, "start", "hn")
	proj := filepath.Join(root, "hn")
	today := time.Now().Format("2006-01-02")
	write(t, filepath.Join(proj, ".omut/brief.md"), fill(today, today, ""))
	write(t, filepath.Join(proj, ".omut/decisions.md"), "# Decisions\n\n- D1 [accepted 2026-10-01] Python\n- D2 [open] Keep collecting after the week?\n")
	write(t, filepath.Join(proj, ".omut/threads/future.md"), "---\nstatus: parked\nupdated: "+today+"\nby: t/t\nparked_because: doubt\n---\n# Future\n**Next step:** ask about D2\n")
	write(t, filepath.Join(proj, ".omut/threads/old.md"), "---\nstatus: done\nupdated: "+today+"\nby: t/t\n---\n# Old\n")
	write(t, filepath.Join(proj, "src/main.py"), "print(1)\n")

	out, _, err := hookRun(t, filepath.Join(proj, "src"), "")
	if err == nil {
		t.Fatal("hook without an event must fail")
	}
	out, _, err = hookRun(t, filepath.Join(proj, "src"), "", "start")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"# omut: the meta of hn", "An HN parser for a week.", "- D2 [open] Keep collecting", "threads/future.md` (parked: doubt) next step: ask about D2"} {
		if !strings.Contains(out, want) {
			t.Errorf("start context has no %q:\n%s", want, out)
		}
	}
	for _, unwanted := range []string{"D1 [accepted", "old.md"} {
		if strings.Contains(out, unwanted) {
			t.Errorf("start context has %q:\n%s", unwanted, out)
		}
	}

	if out, _, _ := hookRun(t, root, "", "start"); !strings.Contains(out, "you are the lead") {
		t.Errorf("start in the pack root:\n%s", out)
	}
	if out, _, _ := hookRun(t, t.TempDir(), "", "start"); out != "" {
		t.Errorf("start outside any meta must be silent: %q", out)
	}

	// Agents say where the session runs; the hook's own directory doesn't matter.
	elsewhere := t.TempDir()
	out, _, _ = hookRun(t, elsewhere, `{"cwd": "`+proj+`", "hook_event_name": "SessionStart"}`, "start", "--agent", "claude")
	if !strings.Contains(out, "# omut: the meta of hn") {
		t.Errorf("claude: cwd from the payload not used:\n%s", out)
	}
	out, _, _ = hookRun(t, elsewhere, `{"workspace_roots": ["`+proj+`"]}`, "start", "--agent", "cursor")
	var cursor map[string]string
	if err := json.Unmarshal([]byte(out), &cursor); err != nil || !strings.Contains(cursor["additional_context"], "An HN parser") {
		t.Errorf("cursor output: %v\n%s", err, out)
	}
	if _, _, err := hookRun(t, proj, "", "start", "--agent", "gemini"); err == nil {
		t.Error("an unknown agent must be refused")
	}
}

func TestHookCommit(t *testing.T) {
	repo := filepath.Join(t.TempDir(), "hn")
	standalone(t, repo, time.Now().Format("2006-01-02"))
	git(t, repo, "init", "-q")
	git(t, repo, "add", ".")
	git(t, repo, "commit", "-qm", "init")
	if _, msg, err := hookRun(t, repo, "", "commit"); err != nil || msg != "" {
		t.Errorf("nothing staged: %q, %v", msg, err)
	}

	logPath := filepath.Join(repo, ".omut/log.md")
	past := time.Now().Add(-time.Hour)
	os.Chtimes(logPath, past, past)
	write(t, filepath.Join(repo, "main.py"), "print(1)\n")
	git(t, repo, "add", "main.py")
	_, msg, err := hookRun(t, repo, "", "commit")
	if err != nil || !strings.Contains(msg, "main.py changed after the last write to .omut/log.md") {
		t.Errorf("work without a log entry: %q, %v", msg, err)
	}
	var exit exitCode
	if _, _, err := hookRun(t, repo, "", "commit", "--strict"); !errors.As(err, &exit) {
		t.Errorf("--strict did not block: %v", err)
	}
	write(t, logPath, "- 2026-11-02 · t/t · main.py\n")
	if _, msg, err := hookRun(t, repo, "", "commit", "--strict"); err != nil || msg != "" {
		t.Errorf("after the log entry: %q, %v", msg, err)
	}

	// Broken meta blocks only a commit that carries it.
	write(t, filepath.Join(repo, ".omut/decisions.md"), "# Decisions\n- D1 [maybe] what\n")
	if _, msg, err := hookRun(t, repo, "", "commit"); err != nil || !strings.Contains(msg, "shape errors") {
		t.Errorf("broken meta outside the commit: %q, %v", msg, err)
	}
	git(t, repo, "add", ".omut/decisions.md")
	if _, msg, err := hookRun(t, repo, "", "commit"); !errors.As(err, &exit) || !strings.Contains(msg, "fix them first") {
		t.Errorf("broken meta in the commit: %q, %v", msg, err)
	}
	if _, msg, err := hookRun(t, t.TempDir(), "", "commit"); err != nil || msg != "" {
		t.Errorf("outside a repo: %q, %v", msg, err)
	}
}

// The startup context is lean by selection (D26): a parked plan is a pointer,
// claims are not repeated with their lines, Access/Runbook stay behind a
// pointer, the local log caps at three entries.
func TestHookStartLeanSelection(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "hn")
	standalone(t, dir, time.Now().Format("2006-01-02"))
	today := time.Now().Format("2006-01-02")
	write(t, filepath.Join(dir, ".omut/local/instance.md"), "---\nupdated: "+today+"\nby: t/t\nplan: plans/x.md\n---\n## Where it lives\n- checkout here · how: `pwd` · seen 2026-10-01\n\n## Access\n- tokens in the password manager\n\n## Runbook\n- deploy with the script\n")
	write(t, filepath.Join(dir, ".omut/local/plans/x.md"), "---\nstatus: parked\n---\n# Package X\n\nMARKER-FULL-PLAN-BODY\n")
	log := ""
	for i := 1; i <= 5; i++ {
		log += "- 2026-11-10 · t/t · event " + string(rune('0'+i)) + "\n"
	}
	write(t, filepath.Join(dir, ".omut/local/log.md"), log)

	out, _, err := hookRun(t, dir, "", "start")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "MARKER-FULL-PLAN-BODY") || !strings.Contains(out, "(parked)") {
		t.Errorf("parked plan not collapsed:\n%s", out)
	}
	if strings.Contains(out, "## Access") || strings.Contains(out, "## Runbook") || !strings.Contains(out, "stays in that file") {
		t.Errorf("Access/Runbook not behind a pointer:\n%s", out)
	}
	if strings.Count(out, "`pwd`") != 1 || !strings.Contains(out, "- `.omut/local/instance.md:") || !strings.Contains(out, "· seen 2026-10-01") {
		t.Errorf("due claims not reduced to locations:\n%s", out)
	}
	if strings.Contains(out, "event 4") || !strings.Contains(out, "event 3") {
		t.Errorf("local log not capped at three:\n%s", out)
	}

	// Without the status the plan travels in full.
	write(t, filepath.Join(dir, ".omut/local/plans/x.md"), "# Package X\n\nMARKER-FULL-PLAN-BODY\n")
	if out, _, _ := hookRun(t, dir, "", "start"); !strings.Contains(out, "MARKER-FULL-PLAN-BODY") {
		t.Errorf("an active plan lost its body:\n%s", out)
	}
}

func TestHookStartUnrecordedWork(t *testing.T) {
	// Without git: any file newer than the log.
	proj := filepath.Join(t.TempDir(), "hn")
	standalone(t, proj, time.Now().Format("2006-01-02"))
	logPath := filepath.Join(proj, ".omut/log.md")
	code := filepath.Join(proj, "main.py")
	write(t, code, "print(1)\n")
	past := time.Now().Add(-time.Hour)
	os.Chtimes(logPath, past, past)
	if out, _, _ := hookRun(t, proj, "", "start"); !strings.Contains(out, "## Unrecorded work") || !strings.Contains(out, "`main.py` changed at") {
		t.Errorf("unrecorded work without git not reported:\n%s", out)
	}
	write(t, logPath, "- 2026-11-02 · t/t · main.py\n")
	if out, _, _ := hookRun(t, proj, "", "start"); strings.Contains(out, "Unrecorded work") {
		t.Errorf("reported after the log was written:\n%s", out)
	}

	// With git: committed work is not unrecorded, a dirty file is.
	git(t, proj, "init", "-q")
	git(t, proj, "add", ".")
	git(t, proj, "commit", "-qm", "init")
	os.Chtimes(logPath, past, past)
	if out, _, _ := hookRun(t, proj, "", "start"); strings.Contains(out, "Unrecorded work") {
		t.Errorf("committed work reported as unrecorded:\n%s", out)
	}
	write(t, code, "print(2)\n")
	if out, _, _ := hookRun(t, proj, "", "start"); !strings.Contains(out, "`main.py` changed at") {
		t.Errorf("uncommitted work not reported:\n%s", out)
	}
}
