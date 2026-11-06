package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
)

const countSite = `n := utf8.RuneCountInString(strings.Join(d.Lines, "\n"))`
const crlfMutant = countSite + `
	// Deliberate verifier mutation: count CRLF as extra content.
	if raw, err := os.ReadFile(d.Path); err == nil {
		n += strings.Count(string(raw), "\r")
	}`

// A real failing Go assertion, rather than a compiler error or timeout, must
// kill the mutant. The unmodified fixture must first survive the same mutant.
func assertionFailed(out []byte) bool {
	for _, line := range bytes.Split(out, []byte("\n")) {
		var e struct{ Action, Test string }
		if json.Unmarshal(line, &e) == nil && e.Action == "fail" && e.Test != "" {
			return true
		}
	}
	return false
}

func gate(c config, dir, artifacts, name string, args ...string) checkResult {
	out, err := command(dir, goEnv(c), "go", args...)
	if writeErr := write(artifacts, name+".txt", out); writeErr != nil {
		return checkResult{name, false, writeErr.Error()}
	}
	return checkResult{name, err == nil, errorText(err)}
}

func errorText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func grade(c config, dir, artifacts, scenario string, files map[string][]byte, before map[string]string) []checkResult {
	allowed := "repair"
	if scenario == "orderly-crlf" {
		allowed = "tests"
	}
	checks := []checkResult{unchanged(before, dir, "receiver scope", allowed)}
	checks = append(checks, gate(c, dir, artifacts, "vet", "vet", "./..."), gate(c, dir, artifacts, "tests", "test", "./...", "-count=1"))
	// Add independently retained policy tests only after the receiving session
	// has ended. Use an isolated name, leaving the agent's own tests intact.
	oracle := strings.Replace(string(files["cmd/omut/size_test.go"]), "TestSizeBudgetsBoundLongParagraphs", "TestHeldOutVolumePolicy", 1)
	path := filepath.Join(dir, "cmd/omut/heldout_volume_test.go")
	if _, err := os.Lstat(path); !os.IsNotExist(err) {
		return append(checks, checkResult{"hidden policy", false, "reserved oracle path already exists"})
	}
	if err := os.WriteFile(path, []byte(oracle), 0600); err != nil {
		return append(checks, checkResult{"hidden policy", false, err.Error()})
	}
	checks = append(checks, gate(c, dir, artifacts, "hidden policy", "test", "./cmd/omut", "-run", "^TestHeldOutVolumePolicy$", "-count=1"))
	// This file was created by the verifier, never by the agent.
	if err := os.Remove(path); err != nil {
		return append(checks, checkResult{"oracle cleanup", false, err.Error()})
	}
	if scenario == "orderly-crlf" {
		checks = append(checks, mutationCheck(c, dir, artifacts))
	}
	return checks
}

func mutationCheck(c config, dir, artifacts string) checkResult {
	path := filepath.Join(dir, "internal/check/check.go")
	original, err := os.ReadFile(path)
	if err != nil {
		return checkResult{"CRLF mutation killed", false, err.Error()}
	}
	if err := replace(dir, "internal/check/check.go", countSite, crlfMutant); err != nil {
		return checkResult{"CRLF mutation killed", false, err.Error()}
	}
	out, testErr := command(dir, goEnv(c), "go", "test", "./cmd/omut", "-json", "-count=1")
	restoreErr := os.WriteFile(path, original, 0600)
	if err := write(artifacts, "mutation.txt", out); err != nil {
		return checkResult{"CRLF mutation killed", false, err.Error()}
	}
	return checkResult{"CRLF mutation killed", restoreErr == nil && testErr != nil && assertionFailed(out), errorText(restoreErr)}
}

func preflight(c config, files map[string][]byte, scenario string) error {
	dir := filepath.Join(c.Out, "preflight", scenario)
	if err := prepare(dir, files, scenario, "none"); err != nil {
		return err
	}
	artifacts := filepath.Join(c.Out, "preflight-artifacts", scenario)
	if err := os.MkdirAll(artifacts, 0700); err != nil {
		return err
	}
	if result := gate(c, dir, artifacts, "baseline", "test", "./...", "-count=1"); !result.Passed {
		return fmt.Errorf("baseline fails: %s", result.Detail)
	}
	if scenario == "orderly-crlf" {
		if mutationCheck(c, dir, artifacts).Passed {
			return fmt.Errorf("LF-only baseline already kills the CRLF mutant: no measurement headroom")
		}
		// Mutation survival must be a successful run, not a compile/infra failure.
		mutantLog, err := os.ReadFile(filepath.Join(artifacts, "mutation.txt"))
		if err != nil || !bytes.Contains(mutantLog, []byte(`"Action":"pass","Package":"github.com/metravod/omut/cmd/omut"`)) {
			return fmt.Errorf("baseline mutant did not execute successfully")
		}
		if err := write(artifacts, "baseline-mutation.txt", mutantLog); err != nil {
			return err
		}
		if err := write(dir, "cmd/omut/size_test.go", files["cmd/omut/size_test.go"]); err != nil {
			return err
		}
		if !mutationCheck(c, dir, artifacts).Passed {
			return fmt.Errorf("retained reference tests do not kill the CRLF mutant")
		}
	} else {
		if err := transition(dir, scenario); err != nil {
			return err
		}
		out, err := command(dir, goEnv(c), "go", "test", "./cmd/omut", "-json", "-run", "^TestSizeBudgetsBoundLongParagraphs$", "-count=1")
		if err == nil || !assertionFailed(out) {
			return fmt.Errorf("injected regression did not fail a test assertion")
		}
		if err := write(dir, "internal/check/check.go", files["internal/check/check.go"]); err != nil {
			return err
		}
	}
	before, err := tree(dir)
	if err != nil {
		return err
	}
	for _, result := range grade(c, dir, artifacts, scenario, files, before) {
		if !result.Passed {
			return fmt.Errorf("reference answer fails %s: %s", result.Name, result.Detail)
		}
	}
	after, err := tree(dir)
	if err != nil || !reflect.DeepEqual(before, after) {
		return fmt.Errorf("verifier changed reference source")
	}
	return nil
}
