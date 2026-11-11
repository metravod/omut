package main

import (
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/metravod/omut/templates"
)

// entries lists everything under dir, so a test can see a command did nothing.
func entries(t *testing.T, dir string) []string {
	t.Helper()
	var out []string
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && d.Name() == ".git" {
			return filepath.SkipDir
		}
		if path != dir {
			out = append(out, path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func TestHelpSameEveryWayAndTouchesNothing(t *testing.T) {
	dir := t.TempDir()
	for topic := range helpTexts {
		via := must(t, dir, "help", topic)
		if !strings.HasPrefix(via, "omut "+topic) {
			t.Errorf("help %s starts with %q", topic, strings.SplitN(via, "\n", 2)[0])
		}
		for _, flag := range []string{"--help", "-h", "-help"} {
			if got := must(t, dir, topic, flag); got != via {
				t.Errorf("%s %s differs from help %s:\n%s", topic, flag, topic, got)
			}
		}
	}
	if got := entries(t, dir); len(got) > 0 {
		t.Errorf("help left files behind: %v", got)
	}
	overview := must(t, dir)
	for _, args := range [][]string{{"help"}, {"--help"}, {"-h"}, {"help", "help"}} {
		if got := must(t, dir, args...); got != overview {
			t.Errorf("%v differs from the overview", args)
		}
	}
}

func TestHelpFitsATerminal(t *testing.T) {
	dir := t.TempDir()
	calls := [][]string{{"help"}}
	for topic := range helpTexts {
		calls = append(calls, []string{"help", topic})
	}
	for _, args := range calls {
		for line := range strings.Lines(must(t, dir, args...)) {
			line = strings.TrimRight(line, "\n")
			// A JSON recipe is copied as a whole, not read at 80 columns.
			if len(line) > helpWidth && !strings.Contains(line, `"hooks"`) {
				t.Errorf("%v: %d columns: %s", args, len(line), line)
			}
		}
	}
}

func TestHelpWinsOverAnActionOnTheSameLine(t *testing.T) {
	root := t.TempDir()
	must(t, root, "init")
	project := filepath.Join(root, "x")
	must(t, root, "start", "x")
	want := must(t, root, "help", "kill")
	if got := must(t, root, "kill", "x", "--yes", "--force", "--reason", "gone", "--help"); got != want {
		t.Errorf("kill --help printed:\n%s", got)
	}
	if _, err := os.Stat(project); err != nil {
		t.Fatalf("kill --help removed the project: %v", err)
	}

	outside := t.TempDir()
	must(t, outside, "start", "--help")
	must(t, outside, "start", "new", "--private", "-h")
	if got := entries(t, outside); len(got) > 0 {
		t.Errorf("start --help wrote: %v", got)
	}
}

func TestHelpListsEveryCommandAndFlag(t *testing.T) {
	src, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	overview := must(t, dir, "help")
	var dispatched []string
	for _, m := range regexp.MustCompile(`case "([a-z]+)"`).FindAllStringSubmatch(string(src), -1) {
		dispatched = append(dispatched, m[1])
	}
	for _, cmd := range dispatched {
		if cmd == "help" {
			continue
		}
		if _, ok := helpTexts[cmd]; !ok {
			t.Errorf("no help text for %s", cmd)
		}
		if !regexp.MustCompile(`(?m)^  ` + cmd + `\b`).MatchString(overview) {
			t.Errorf("the overview doesn't list %s", cmd)
		}
	}
	for cmd := range helpTexts {
		if !slices.Contains(dispatched, cmd) {
			t.Errorf("help text for %s, which is not a command", cmd)
		}
	}
	for cmd, flags := range map[string][]string{
		"start":   {"--private", "--by agent/model"},
		"check":   {"--fail-on level", "--json"},
		"context": {"--json"},
		"dream":   {"--apply", "--json"},
		"zero":    {"--by agent/model"},
		"init":    {"--by agent/model"},
		"kill":    {"--yes", "--force", "--reason text"},
		"hook":    {"--agent name", "--strict", "--preview"},
	} {
		out := must(t, dir, "help", cmd)
		_, section, ok := strings.Cut(out, "\nFlags:\n")
		if !ok {
			t.Errorf("help %s has no flags", cmd)
		}
		for _, f := range flags {
			if !strings.Contains(section, "  "+f+" ") {
				t.Errorf("help %s doesn't list %s:\n%s", cmd, f, section)
			}
		}
	}
	if out := must(t, dir, "help", "freeze"); strings.Contains(out, "Flags:") {
		t.Error("freeze has no flags to list")
	}
}

// What the help says start and init write is what they write.
func TestHelpNamesEveryFileLaidDown(t *testing.T) {
	gitRepo := func() string {
		dir := t.TempDir()
		if out, err := exec.Command("git", "init", "-q", dir).CombinedOutput(); err != nil {
			t.Fatalf("git init: %s", out)
		}
		return dir
	}
	for _, tc := range []struct {
		name string
		args []string
	}{
		{"start", []string{"start"}},
		{"start", []string{"start", "--private"}},
		{"init", []string{"init"}},
	} {
		dir := gitRepo()
		must(t, dir, tc.args...)
		text := must(t, dir, "help", tc.name)
		for _, path := range entries(t, dir) {
			if !strings.Contains(text, filepath.Base(path)) {
				t.Errorf("%v wrote %s; help %s doesn't say so", tc.args, path, tc.name)
			}
		}
	}
}

// Three lists of the check rules: the code, the help and spec §8.
func TestCheckRulesAgreeWithCodeAndSpec(t *testing.T) {
	inHelp := map[string]string{}
	for _, r := range checkRules {
		inHelp[r.code] = r.level
	}

	inCode := map[string]bool{}
	sources, err := filepath.Glob("../../internal/check/*.go")
	if err != nil || len(sources) == 0 {
		t.Fatalf("no sources of internal/check: %v", err)
	}
	for _, path := range sources {
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range regexp.MustCompile(`r\.add\("([a-z-]+)"`).FindAllSubmatch(b, -1) {
			inCode[string(m[1])] = true
		}
	}
	for rule := range inCode {
		if _, ok := inHelp[rule]; !ok {
			t.Errorf("rule %s is in internal/check, not in the help", rule)
		}
	}

	b, err := os.ReadFile("../../docs/spec-v0.md")
	if err != nil {
		t.Fatal(err)
	}
	_, spec, ok := strings.Cut(string(b), "## 8. check rules")
	if !ok {
		t.Fatal("no section 8 in the spec")
	}
	spec, _, _ = strings.Cut(spec, "\n## ")
	inSpec := map[string]string{}
	for _, m := range regexp.MustCompile("(?m)^\\| `([a-z-]+)` \\| ([a-z]+) \\|").FindAllStringSubmatch(spec, -1) {
		inSpec[m[1]] = m[2]
	}
	if !maps.Equal(inHelp, inSpec) {
		t.Errorf("rules differ:\nhelp %v\nspec %v", inHelp, inSpec)
	}
	for rule := range inHelp {
		if !inCode[rule] {
			t.Errorf("rule %s is in the help, not in internal/check", rule)
		}
	}
}

func TestHelpMetaIsTheRulesAProjectGets(t *testing.T) {
	dir := t.TempDir()
	out := must(t, dir, "help", "meta")
	if got := entries(t, dir); len(got) > 0 {
		t.Errorf("help meta wrote: %v", got)
	}
	b, err := templates.FS.ReadFile("project/omut/README.md")
	if err != nil {
		t.Fatal(err)
	}
	_, body, _ := strings.Cut(string(b), "\n")
	if !strings.HasSuffix(out, body) {
		t.Errorf("help meta is not the template's text:\n%.300s", out)
	}
	if strings.Contains(out, "Tool file:") {
		t.Error("help meta keeps the note for those editing the file")
	}
}

func TestHelpMistakes(t *testing.T) {
	dir := t.TempDir()
	if _, err := omut(t, dir, "help", "nope"); err == nil || !strings.Contains(err.Error(), "omut help") {
		t.Errorf("help nope: %v", err)
	}
	_, err := omut(t, dir, "start", "--nope")
	if err == nil || !strings.Contains(err.Error(), "(omut help start)") {
		t.Errorf("a wrong flag doesn't point at the help: %v", err)
	}
	if _, err := omut(t, dir, "freeze", "--nope"); err == nil || !strings.Contains(err.Error(), "(omut help freeze)") {
		t.Errorf("freeze names the wrong topic: %v", err)
	}
	var h helpRequest
	if errors.As(fmt.Errorf("%w", err), &h) {
		t.Error("a wrong flag is not a request for help")
	}
	if got := entries(t, dir); len(got) > 0 {
		t.Errorf("wrote: %v", got)
	}
}
