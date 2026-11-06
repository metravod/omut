// handoffbench is a research runner, separate from the omut CLI.
package main

import (
	"archive/tar"
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

const defaultRef = "52c4601ea573c021545491b57155eaaa433d2fd7"
const protocol = "handoff-smoke-v1"

var conditions = []string{"none", "note", "omut"}
var scenarios = []string{"orderly-crlf", "stale-thread", "irrelevant-history"}

const commonRules = `# Project rules
Go, standard library only. Read docs/spec-v0.md for intent.
Run go vet ./... and go test ./... before reporting completion.
Work only inside this workspace. Do not inspect parent directories, other
workspaces, host memory, session logs, or benchmark machinery. Do not use the
network, external services, subagents, or git history. Do not commit.
The harness supplies GOCACHE; use that cache for Go commands.
The supplied TMPDIR is a dedicated writable test scratch directory. Use it as
provided; do not put temporary test fixtures inside this workspace, where their
parent-discovery tests would find this project's Git repository and memory.
Go may create its own temporary fixtures in that supplied scratch directory.
`

const resumePrompt = `Continue the unfinished development work in this repository.
Inspect the current files and available project records to determine what is
implemented, what remains, and which decisions apply. Verify recorded claims
against the current tree. Complete the remaining work, run the project checks,
and update any persistent project records required by its instructions.
If the available evidence does not identify a task, say so instead of inventing
one. Finish autonomously with the available evidence; no human replies will be
provided. Work only inside this workspace and do not commit.
`

type config struct {
	Repo, Out, Ref, Model, Reasoning, Codex string
	Cases                                   []string
	Repeats                                 int
	Seed                                    int64
	Timeout                                 time.Duration
	Execute                                 bool
}

type usage struct {
	Input     int64 `json:"input_tokens"`
	Cached    int64 `json:"cached_input_tokens"`
	Output    int64 `json:"output_tokens"`
	Reasoning int64 `json:"reasoning_output_tokens"`
}

type stage struct {
	ExitCode    int      `json:"exit_code"`
	Seconds     float64  `json:"seconds"`
	Completed   bool     `json:"completed"`
	TimedOut    bool     `json:"timed_out"`
	Usage       usage    `json:"usage"`
	Commands    int      `json:"commands"`
	EventErrors []string `json:"event_errors,omitempty"`
}

type checkResult struct {
	Name   string `json:"name"`
	Passed bool   `json:"passed"`
	Detail string `json:"detail,omitempty"`
}

type cell struct {
	ID, Scenario, Condition string
	Repeat                  int
	Source, Receiver        *stage
	Checks                  []checkResult
	Status                  string
	Success                 bool
	MemoryBytes             int64
	TotalUsage              usage
	AgentSeconds            float64
}

type manifest struct {
	Protocol, Started, SourceCommit, RunnerSHA256, CodexVersion string
	Config                                                      config
	Cells                                                       []cell
	Limitations                                                 []string
	Rescore                                                     *rescoreInfo `json:",omitempty"`
}

type rescoreInfo struct {
	OriginalManifestSHA256, OriginalRun, ScoredAt, Revision string
	InferenceInvoked                                        bool
}

func main() {
	var c config
	var cases string
	var rescore string
	flag.StringVar(&c.Repo, "repo", ".", "omut checkout containing the pinned source commit")
	flag.StringVar(&c.Out, "out", "", "new output directory outside the checkout (required)")
	flag.StringVar(&c.Ref, "ref", defaultRef, "source commit; resolved and recorded")
	flag.StringVar(&c.Model, "model", "gpt-6.1-sol", "fixed model for every session")
	flag.StringVar(&c.Reasoning, "reasoning", "high", "fixed reasoning effort")
	flag.StringVar(&c.Codex, "codex", "codex", "Codex CLI executable")
	flag.StringVar(&cases, "cases", strings.Join(scenarios, ","), "comma-separated scenarios")
	flag.IntVar(&c.Repeats, "repeats", 1, "independent source/receiver pairs per scenario and condition")
	flag.Int64Var(&c.Seed, "seed", 20261105, "cell order randomization seed")
	flag.DurationVar(&c.Timeout, "timeout", 5*time.Minute, "maximum wall time per agent session")
	flag.BoolVar(&c.Execute, "execute", false, "invoke model sessions; otherwise prepare and check fixtures offline")
	flag.StringVar(&rescore, "rescore", "", "rescore a completed run into a new -out directory, without model calls")
	flag.Parse()
	c.Cases = strings.Split(cases, ",")
	var err error
	if rescore != "" {
		err = rescoreRun(rescore, c.Out)
	} else {
		err = run(c)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(c config) error {
	var err error
	c.Repo, err = filepath.Abs(c.Repo)
	if err != nil {
		return err
	}
	if c.Out == "" || c.Repeats < 1 || c.Timeout <= 0 {
		return errors.New("provide -out, positive -repeats and positive -timeout")
	}
	c.Out, err = filepath.Abs(c.Out)
	if err != nil {
		return err
	}
	// A workspace under the checkout inherits the real project's AGENTS.md.
	if rel, e := filepath.Rel(c.Repo, c.Out); e != nil || (rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator))) {
		return errors.New("-out must be outside the checkout to avoid inherited project instructions")
	}
	for _, name := range c.Cases {
		if !slices.Contains(scenarios, name) {
			return fmt.Errorf("unknown scenario %q", name)
		}
	}
	if err := os.Mkdir(c.Out, 0700); err != nil {
		return fmt.Errorf("output directory must be new: %w", err)
	}
	ref, err := command(c.Repo, nil, "git", "rev-parse", c.Ref+"^{commit}")
	if err != nil {
		return err
	}
	c.Ref = strings.TrimSpace(string(ref))
	archive, err := command(c.Repo, nil, "git", "archive", c.Ref)
	if err != nil {
		return err
	}
	files, err := snapshot(archive)
	if err != nil {
		return err
	}
	version, err := command(c.Repo, nil, c.Codex, "--version")
	if err != nil && c.Execute {
		return err
	}
	exe, _ := os.Executable()
	runner, err := os.ReadFile(exe)
	if err != nil {
		return err
	}
	m := manifest{Protocol: protocol, Started: time.Now().UTC().Format(time.RFC3339), SourceCommit: c.Ref,
		RunnerSHA256: digest(runner), CodexVersion: strings.TrimSpace(string(version)), Config: c,
		Limitations: []string{
			"One repository; controlled modifications of real source, not independent real-world tasks.",
			"Source sessions review and verify existing work; they do not implement the original feature.",
			"Fresh ephemeral sessions, native memory disabled in all conditions; none is a lower bound, not a native-memory comparison.",
			"omut uses file discovery through AGENTS.md, not native startup hooks.",
			"workspace-write restricts writes, not all host reads; the workspace-only input boundary is an instruction, not a container.",
			"No human corrections supplied; final reports and boundary compliance still require blinded manual review.",
			"No currency cost estimate; token categories and source/receiver time are reported separately.",
			"Runner revision 2 supplies external per-phase test scratch directories to prevent nested-fixture discovery and side effects.",
		}}
	for _, name := range c.Cases {
		for n := 1; n <= c.Repeats; n++ {
			for _, condition := range conditions {
				m.Cells = append(m.Cells, cell{Scenario: name, Condition: condition, Repeat: n, Status: "prepared"})
			}
		}
	}
	rand.New(rand.NewSource(c.Seed)).Shuffle(len(m.Cells), func(i, j int) { m.Cells[i], m.Cells[j] = m.Cells[j], m.Cells[i] })
	for i := range m.Cells {
		m.Cells[i].ID = fmt.Sprintf("cell-%03d", i+1)
	}
	if err := saveManifest(c.Out, &m); err != nil {
		return err
	}
	if err := jsonFile(filepath.Join(c.Out, "plan.json"), &m); err != nil {
		return err
	}
	// Check the baseline and the scorer before observing any model outcomes.
	for _, name := range c.Cases {
		fmt.Printf("preflight %s\n", name)
		if err := preflight(c, files, name); err != nil {
			return fmt.Errorf("invalid fixture/oracle %s: %w", name, err)
		}
	}
	for i := range m.Cells {
		v := &m.Cells[i]
		workspace := filepath.Join(c.Out, v.ID, "workspace")
		if err := prepare(workspace, files, v.Scenario, v.Condition); err != nil {
			return err
		}
		artifacts := filepath.Join(c.Out, v.ID, "artifacts")
		if err := os.MkdirAll(artifacts, 0700); err != nil {
			return err
		}
		source := sourcePrompt(v.Scenario)
		receiver := receiverPrompt(v.Scenario)
		for name, prompt := range map[string]string{"source-prompt.txt": source, "receiver-prompt.txt": receiver} {
			if err := os.WriteFile(filepath.Join(artifacts, name), []byte(prompt), 0600); err != nil {
				return err
			}
		}
		if !c.Execute {
			continue
		}
		fmt.Printf("%s %s %s repeat %d: source\n", v.ID, v.Scenario, v.Condition, v.Repeat)
		before, err := tree(workspace)
		if err != nil {
			return err
		}
		s, err := agent(c, workspace, artifacts, "source", source)
		if err != nil {
			return err
		}
		v.Source = &s
		v.Checks = append(v.Checks, unchanged(before, workspace, "source scope", ""))
		v.Checks = append(v.Checks, memoryScope(workspace, v.Condition))
		v.MemoryBytes, err = memoryBytes(workspace, v.Condition)
		if err != nil {
			return err
		}
		if !s.Completed || s.ExitCode != 0 || !v.Checks[0].Passed || !v.Checks[1].Passed {
			v.Status = "invalid-source"
			finishCell(v)
			if err := saveManifest(c.Out, &m); err != nil {
				return err
			}
			if !s.Completed || s.ExitCode != 0 {
				return fmt.Errorf("%s source session incomplete; inspect its artifacts before a new run", v.ID)
			}
			continue
		}
		if err := transition(workspace, v.Scenario); err != nil {
			return err
		}
		before, err = tree(workspace)
		if err != nil {
			return err
		}
		v.Status = "source-complete"
		finishCell(v)
		if err := saveManifest(c.Out, &m); err != nil {
			return err
		}
		fmt.Printf("%s: receiver\n", v.ID)
		r, err := agent(c, workspace, artifacts, "receiver", receiver)
		if err != nil {
			return err
		}
		v.Receiver = &r
		v.Checks = append(v.Checks, grade(c, workspace, artifacts, v.Scenario, files, before)...)
		v.Checks = append(v.Checks, memoryScope(workspace, v.Condition))
		v.Status = "scored"
		if !r.Completed || r.ExitCode != 0 {
			v.Status = "incomplete-receiver"
		}
		finishCell(v)
		fmt.Printf("%s: %s, functional pass=%v\n", v.ID, v.Status, v.Success)
		if err := saveManifest(c.Out, &m); err != nil {
			return err
		}
		if v.Status == "incomplete-receiver" {
			return fmt.Errorf("%s receiver session incomplete; inspect its artifacts before a new run", v.ID)
		}
	}
	return saveManifest(c.Out, &m)
}

func digest(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// Only product source, templates and the spec enter a fixture. In particular,
// project memory, worked answers, research, git history and the runner do not.
func snapshot(b []byte) (map[string][]byte, error) {
	files := map[string][]byte{}
	r := tar.NewReader(bytes.NewReader(b))
	for {
		h, err := r.Next()
		if err == io.EOF {
			return files, nil
		}
		if err != nil {
			return nil, err
		}
		name := filepath.ToSlash(filepath.Clean(h.Name))
		if strings.HasPrefix(name, "../") || filepath.IsAbs(name) {
			return nil, fmt.Errorf("unsafe archive path %q", name)
		}
		if h.Typeflag != tar.TypeReg {
			continue
		}
		if name != "go.mod" && name != "docs/spec-v0.md" && !strings.HasPrefix(name, "cmd/") && !strings.HasPrefix(name, "internal/") && !strings.HasPrefix(name, "templates/") {
			continue
		}
		files[name], err = io.ReadAll(r)
		if err != nil {
			return nil, err
		}
	}
}

func prepare(dir string, files map[string][]byte, scenario, condition string) error {
	for name, b := range files {
		if err := write(dir, name, b); err != nil {
			return err
		}
	}
	if scenario == "orderly-crlf" {
		old := `for _, ending := range []struct{ name, text string }{{"LF", "\n"}, {"CRLF", "\r\n"}} {`
		new := `{ // This controlled snapshot exercises LF only.
			ending := struct{ name, text string }{"LF", "\n"}`
		if err := replace(dir, "cmd/omut/size_test.go", old, new); err != nil {
			return err
		}
	}
	rules := commonRules
	switch condition {
	case "none":
		rules += "\nUse the existing project files. No persistent handoff record is configured.\n"
	case "note":
		rules += "\nRead handoff.md at session start if it exists. Before finishing a piece of\nwork, maintain handoff.md with whatever will help the next session continue.\nChoose its structure yourself. Keep its claims checked against the code.\n"
	case "omut":
		block, ok := files["templates/project/block.md"]
		if !ok {
			return errors.New("source snapshot has no omut pointer template")
		}
		rules += "\n<!-- omut:start -->\n" + string(block) + "<!-- omut:end -->\n"
		for name, b := range files {
			if !strings.HasPrefix(name, "templates/project/omut/") {
				continue
			}
			rel := strings.TrimPrefix(name, "templates/project/omut/")
			switch rel {
			case "gitignore":
				rel = ".gitignore"
			case "gitattributes":
				rel = ".gitattributes"
			}
			text := strings.ReplaceAll(string(b), "{{date}}", time.Now().Format("2006-01-02"))
			text = strings.ReplaceAll(text, "{{by}}", "benchmark/template")
			if err := write(dir, ".omut/"+rel, []byte(text)); err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("unknown condition %q", condition)
	}
	if err := write(dir, "AGENTS.md", []byte(rules)); err != nil {
		return err
	}
	for _, args := range [][]string{{"init", "-q"}, {"add", "."}, {"commit", "-qm", "Controlled source snapshot"}} {
		if _, err := git(dir, args...); err != nil {
			return err
		}
	}
	return nil
}

func sourcePrompt(scenario string) string {
	preamble := `Review this repository as the outgoing developer. The project exists to keep
project state in human-readable markdown across agents. The CLI scaffolds and
checks shape; agents own meaning and verification. Adopt only the facts in this
request and the current files. No external history or human answers are needed.
This phase is a review and verification handoff: do not change product source,
tests, specification or AGENTS.md. You may maintain the persistent project
records configured in AGENTS.md. Run the relevant checks and record only their
observed outcomes. Do not perform the next phase in this session.

`
	switch scenario {
	case "orderly-crlf":
		return preamble + `The Unicode volume feature has been implemented: briefs allow 6000 Unicode
code points and 60 lines, threads 8000 code points and 80 lines. Both limits
apply; a final newline is excluded, LF and CRLF must count equally. Byte counts
were rejected because they penalize Cyrillic; tokenizer dependencies were
rejected because the CLI uses only the Go standard library.
Verify the existing ASCII/Cyrillic boundary checks and inspect whether CRLF is
covered. The remaining task for the NEXT session is to add CRLF boundary
regression coverage for both files and both alphabets, at and one code point
above the limits, then run the full gate. The accepted production implementation
is already correct; preserve it. Leave this task pending for that next session.
`
	case "stale-thread":
		return preamble + `Verify the existing volume-budget implementation and tests. Accepted policy:
6000 Unicode code points / 60 lines for briefs and 8000 / 80 for threads;
LF and CRLF count equally and a final newline is excluded. Byte-based limits
were rejected for language fairness. Run the focused size test and full gate.
If they pass, record that this piece of work is complete; no next change is
currently requested. Do not invent a follow-up task.
`
	default:
		return preamble + `Review how existing ADR status formats are read. The project keeps an ADR's
decision in the ADR registry instead of requiring a duplicate decisions.md
entry. Verify the existing ADR tests and full gate. If they pass, record this
review as complete. There is no pending ADR feature or requested code change.
`
	}
}

func receiverPrompt(scenario string) string {
	if scenario == "irrelevant-history" {
		return `A regression makes the thread checker accept an 8001-code-point thread
when it has fewer than 80 lines. Fix this bug according to the current spec,
preserving the existing brief policy, Unicode counting, line limits, newline
handling, and warning/exit behavior. Run the full checks and update any project
records required by AGENTS.md. Work only inside this workspace; do not commit.
`
	}
	return resumePrompt
}

func transition(dir, scenario string) error {
	if scenario == "orderly-crlf" {
		return nil
	}
	return replace(dir, "internal/check/check.go", "checkSize(r, d, ThreadLines, ThreadChars)", `if d.LineCount() > ThreadLines {
		r.add("size", Warn, d.Path, 0, "thread is over the line budget")
	}`)
}

func write(dir, name string, b []byte) error {
	path := filepath.Join(dir, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	return os.WriteFile(path, b, 0600)
}

func replace(dir, name, old, new string) error {
	b, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		return err
	}
	if strings.Count(string(b), old) != 1 {
		return fmt.Errorf("expected exactly one mutation site in %s", name)
	}
	return write(dir, name, []byte(strings.Replace(string(b), old, new, 1)))
}

func command(dir string, env []string, name string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	if err != nil {
		return out, fmt.Errorf("%s %v: %w\n%s", name, args, err, out)
	}
	return out, nil
}

func git(dir string, args ...string) ([]byte, error) {
	return command(dir, nil, "git", append([]string{"-c", "core.hooksPath=/dev/null", "-c", "user.name=handoffbench", "-c", "user.email=benchmark@example.invalid", "-c", "commit.gpgsign=false"}, args...)...)
}

func goEnv(c config) []string {
	env := os.Environ()
	// A shared, prewarmed cache reduces repeated compilation. Wall time remains
	// secondary: cache warmth and server latency are not controlled resources.
	return setEnv(env, "GOCACHE", filepath.Join(c.Out, "go-cache"), "GOTOOLCHAIN", "local")
}

func setEnv(env []string, pairs ...string) []string {
	out := append([]string(nil), env...)
	for i := 0; i < len(pairs); i += 2 {
		key := pairs[i] + "="
		out = slices.DeleteFunc(out, func(s string) bool { return strings.HasPrefix(s, key) })
		out = append(out, key+pairs[i+1])
	}
	return out
}

func agent(c config, dir, artifacts, phase, prompt string) (stage, error) {
	var s stage
	// Auth is referenced, never read into the runner or printed. Each phase
	// otherwise gets an empty Codex home: no user config, memory, hooks or logs.
	home, err := os.MkdirTemp("", "omut-bench-codex-")
	if err != nil {
		return s, err
	}
	authHome := os.Getenv("CODEX_HOME")
	if authHome == "" {
		userHome, err := os.UserHomeDir()
		if err != nil {
			return s, err
		}
		authHome = filepath.Join(userHome, ".codex")
	}
	auth := filepath.Join(authHome, "auth.json")
	if _, err := os.Stat(auth); err == nil {
		if err := os.Symlink(auth, filepath.Join(home, "auth.json")); err != nil {
			return s, err
		}
		defer os.Remove(filepath.Join(home, "auth.json"))
	}
	args := []string{"--no-daemon", "-a", "never", "exec", "--ignore-user-config", "--ignore-rules", "--ephemeral",
		"--sandbox", "workspace-write", "--model", c.Model, "-c", "model_reasoning_effort=" + c.Reasoning,
		"-c", `web_search="disabled"`, "--disable", "memories", "--disable", "multi_agent", "--disable", "hooks",
		"--disable", "plugins", "--disable", "apps", "--enable", "skip_host_skill_discovery",
		"--cd", dir, "--json", "--color", "never", "--output-last-message", filepath.Join(artifacts, phase+"-last.txt"), "-"}
	if err := jsonFile(filepath.Join(artifacts, phase+"-invocation.json"), map[string]any{
		"executable": c.Codex, "args": args, "codex_home": home, "prompt_sha256": digest([]byte(prompt)),
	}); err != nil {
		return s, err
	}
	stdout, err := os.Create(filepath.Join(artifacts, phase+"-events.jsonl"))
	if err != nil {
		return s, err
	}
	defer stdout.Close()
	stderr, err := os.Create(filepath.Join(artifacts, phase+"-stderr.txt"))
	if err != nil {
		return s, err
	}
	defer stderr.Close()
	ctx, cancel := context.WithTimeout(context.Background(), c.Timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, c.Codex, args...)
	cmd.Dir = dir
	scratch, err := os.MkdirTemp(c.Out, "test-scratch-")
	if err != nil {
		return s, err
	}
	cmd.Env = setEnv(goEnv(c), "CODEX_HOME", home, "TMPDIR", scratch)
	cmd.Stdin = strings.NewReader(prompt)
	cmd.Stdout, cmd.Stderr = stdout, stderr
	start := time.Now()
	err = cmd.Run()
	s.Seconds = time.Since(start).Seconds()
	s.ExitCode = 0
	if err != nil {
		s.ExitCode = -1
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			s.ExitCode = exit.ExitCode()
		}
		s.EventErrors = append(s.EventErrors, err.Error())
	}
	s.TimedOut = ctx.Err() != nil
	if err := stdout.Close(); err != nil {
		return s, err
	}
	if err := readEvents(filepath.Join(artifacts, phase+"-events.jsonl"), &s); err != nil {
		s.EventErrors = append(s.EventErrors, err.Error())
		s.Completed = false
	}
	return s, nil
}

func readEvents(path string, s *stage) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 64*1024), 16*1024*1024)
	turns := 0
	for scanner.Scan() {
		var event struct {
			Type  string
			Usage usage
			Item  struct{ Type string }
		}
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			return fmt.Errorf("invalid JSONL event: %w", err)
		}
		switch event.Type {
		case "turn.completed":
			turns++
			s.Usage = addUsage(s.Usage, event.Usage)
		case "turn.failed", "error":
			s.EventErrors = append(s.EventErrors, string(scanner.Bytes()))
		case "item.completed":
			if event.Item.Type == "command_execution" {
				s.Commands++
			}
		}
	}
	s.Completed = turns == 1 && len(s.EventErrors) == 0 && s.ExitCode == 0 && !s.TimedOut
	return scanner.Err()
}

func tree(dir string) (map[string]string, error) {
	out := map[string]string{}
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(dir, path)
		rel = filepath.ToSlash(rel)
		if d.IsDir() {
			if rel == ".git" || rel == ".omut" {
				return filepath.SkipDir
			}
			return nil
		}
		if rel == "handoff.md" {
			return nil
		}
		if d.Type()&os.ModeSymlink != 0 {
			out[rel] = "symlink"
			return nil
		}
		b, err := os.ReadFile(path)
		if err == nil {
			out[rel] = digest(b)
		}
		return err
	})
	return out, err
}

func unchanged(before map[string]string, dir, name, allowed string) checkResult {
	after, err := tree(dir)
	if err != nil {
		return checkResult{name, false, err.Error()}
	}
	keys := map[string]bool{}
	for k := range before {
		keys[k] = true
	}
	for k := range after {
		keys[k] = true
	}
	var changed []string
	for k := range keys {
		if before[k] == after[k] {
			continue
		}
		if (allowed == "tests" || allowed == "repair") && strings.HasPrefix(k, "cmd/omut/") && strings.HasSuffix(k, "_test.go") &&
			((k == "cmd/omut/size_test.go" && after[k] != "") || before[k] == "") {
			continue
		}
		if allowed == "repair" && k == "internal/check/check.go" {
			continue
		}
		if k == allowed {
			continue
		}
		changed = append(changed, k)
	}
	slices.Sort(changed)
	return checkResult{name, len(changed) == 0, strings.Join(changed, ", ")}
}

func memoryBytes(dir, condition string) (int64, error) {
	if condition == "none" {
		return 0, nil
	}
	path := filepath.Join(dir, "handoff.md")
	if condition == "omut" {
		path = filepath.Join(dir, ".omut")
	}
	var size int64
	err := filepath.WalkDir(path, func(p string, d os.DirEntry, err error) error {
		if os.IsNotExist(err) {
			return nil
		}
		if err != nil {
			return err
		}
		if d.Type().IsRegular() {
			info, err := d.Info()
			if err != nil {
				return err
			}
			size += info.Size()
		}
		return nil
	})
	return size, err
}

func memoryScope(dir, condition string) checkResult {
	var forbidden []string
	if condition != "omut" {
		forbidden = append(forbidden, ".omut")
	}
	if condition != "note" {
		forbidden = append(forbidden, "handoff.md")
	}
	for _, name := range forbidden {
		if _, err := os.Lstat(filepath.Join(dir, name)); !os.IsNotExist(err) {
			return checkResult{"memory scope", false, "unexpected " + name}
		}
	}
	return checkResult{"memory scope", true, ""}
}

func addUsage(a, b usage) usage {
	return usage{a.Input + b.Input, a.Cached + b.Cached, a.Output + b.Output, a.Reasoning + b.Reasoning}
}

func finishCell(v *cell) {
	if v.Source != nil {
		v.TotalUsage = v.Source.Usage
		v.AgentSeconds = v.Source.Seconds
	}
	if v.Receiver != nil {
		v.TotalUsage = addUsage(v.TotalUsage, v.Receiver.Usage)
		v.AgentSeconds += v.Receiver.Seconds
	}
	v.Success = v.Status == "scored" && len(v.Checks) > 0 &&
		v.Source != nil && v.Source.Completed && v.Receiver != nil && v.Receiver.Completed
	for _, check := range v.Checks {
		v.Success = v.Success && check.Passed
	}
}

func jsonFile(path string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(b, '\n'), 0600)
}

func saveManifest(out string, m *manifest) error {
	if err := jsonFile(filepath.Join(out, "manifest.json"), m); err != nil {
		return err
	}
	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\nSource: `%s`. This is a smoke test, not evidence of general superiority.\n\n", m.Protocol, m.SourceCommit)
	if m.Rescore != nil {
		fmt.Fprintf(&b, "Scorer amendment: %s. Original model transcripts and initial scores are retained; no inference was invoked for this rescore.\n\n", m.Rescore.Revision)
	}
	b.WriteString("| Cell | Scenario | Condition | Status | Functional pass | Source s | Receiver s | Input tokens | Cached subset | Output tokens |\n|---|---|---|---|---|---:|---:|---:|---:|---:|\n")
	for _, v := range m.Cells {
		var sourceTime, receiverTime float64
		if v.Source != nil {
			sourceTime = v.Source.Seconds
		}
		if v.Receiver != nil {
			receiverTime = v.Receiver.Seconds
		}
		pass := "pending"
		if v.Status != "prepared" {
			pass = fmt.Sprint(v.Success)
		}
		fmt.Fprintf(&b, "| %s | %s | %s | %s | %s | %.1f | %.1f | %d | %d | %d |\n", v.ID, v.Scenario, v.Condition, v.Status, pass, sourceTime, receiverTime, v.TotalUsage.Input, v.TotalUsage.Cached, v.TotalUsage.Output)
	}
	b.WriteString("\nCached input is a subset of input, not an additional token count. Source costs include verification and memory writing. No currency estimate is made.\n\n")
	for _, limitation := range m.Limitations {
		fmt.Fprintf(&b, "- %s\n", limitation)
	}
	return os.WriteFile(filepath.Join(out, "report.md"), []byte(b.String()), 0600)
}
