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
