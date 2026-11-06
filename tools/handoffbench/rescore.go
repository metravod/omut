package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Rescoring is explicit, offline, applied to every available cell, and written
// separately. It never replaces the original run's scores or transcripts.
func rescoreRun(original, output string) error {
	if original == "" || output == "" {
		return fmt.Errorf("-rescore and a new -out directory are required")
	}
	original, err := filepath.Abs(original)
	if err != nil {
		return err
	}
	b, err := os.ReadFile(filepath.Join(original, "manifest.json"))
	if err != nil {
		return err
	}
	var m manifest
	if err := json.Unmarshal(b, &m); err != nil {
		return err
	}
	for _, v := range m.Cells {
		if v.Status == "prepared" || v.Status == "source-complete" {
			return fmt.Errorf("run is unfinished: %s is %s", v.ID, v.Status)
		}
	}
	if err := os.Mkdir(output, 0700); err != nil {
		return err
	}
	archive, err := command(m.Config.Repo, nil, "git", "archive", m.SourceCommit)
	if err != nil {
		return err
	}
	files, err := snapshot(archive)
	if err != nil {
		return err
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	runner, err := os.ReadFile(exe)
	if err != nil {
		return err
	}
	m.Protocol = protocol + "-scorer2"
	m.RunnerSHA256 = digest(runner)
	m.Rescore = &rescoreInfo{OriginalManifestSHA256: digest(b), OriginalRun: original,
		ScoredAt: time.Now().UTC().Format(time.RFC3339),
		Revision: "Allow task-related regression tests during production repair; initial scorer incorrectly rejected them.",
	}
	for i := range m.Cells {
		v := &m.Cells[i]
		if v.Receiver == nil {
			continue // preserve incomplete/invalid source outcomes, never invent a receiver
		}
		// Source scope checks established that only memory changed. Recreate the
		// identical non-memory receiving baseline from the pinned source.
		baseline := filepath.Join(output, "baselines", v.ID)
		workspace := filepath.Join(original, v.ID, "workspace")
		if err := prepare(baseline, files, v.Scenario, v.Condition); err != nil {
			return err
		}
		// Recover the actual initial instructions: their template can change
		// between runner revisions, and the receiver's copy is not the baseline.
		agents, err := initialAgents(workspace)
		if err != nil {
			return err
		}
		if err := write(baseline, "AGENTS.md", agents); err != nil {
			return err
		}
		if err := transition(baseline, v.Scenario); err != nil {
			return err
		}
		before, err := tree(baseline)
		if err != nil {
			return err
		}
		if len(v.Checks) < 2 {
			return fmt.Errorf("missing source scope checks for %s", v.ID)
		}
		v.Checks = append([]checkResult(nil), v.Checks[:2]...)
		artifacts := filepath.Join(output, v.ID, "artifacts")
		fmt.Printf("rescore %s %s %s\n", v.ID, v.Scenario, v.Condition)
		v.Checks = append(v.Checks, grade(m.Config, workspace, artifacts, v.Scenario, files, before)...)
		v.Checks = append(v.Checks, memoryScope(workspace, v.Condition))
		finishCell(v)
		if err := saveManifest(output, &m); err != nil {
			return err
		}
	}
	return saveManifest(output, &m)
}

func initialAgents(workspace string) ([]byte, error) {
	root, err := git(workspace, "rev-list", "--max-parents=0", "HEAD")
	if err != nil {
		return nil, err
	}
	commits := strings.Fields(string(root))
	if len(commits) != 1 {
		return nil, fmt.Errorf("expected one controlled fixture root commit")
	}
	return git(workspace, "show", commits[0]+":AGENTS.md")
}
