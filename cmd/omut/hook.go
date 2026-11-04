package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"slices"

	"github.com/metravod/omut/internal/hook"
)

// agentInput is the part of an agent's hook payload omut needs: where the
// session runs. Claude Code and Codex send cwd, Cursor workspace_roots.
type agentInput struct {
	Cwd            string   `json:"cwd"`
	WorkspaceRoots []string `json:"workspace_roots"`
}

var agents = []string{"claude", "codex", "cursor"}

func cmdHook(args []string, out io.Writer, stdin *os.File, stderr io.Writer) error {
	fs := flag.NewFlagSet("hook", flag.ContinueOnError)
	agent := fs.String("agent", "", "`name` the agent calling: claude, codex or cursor; reads its JSON payload and answers in its format")
	strict := fs.Bool("strict", false, "commit: block work committed without a log entry")
	pos, err := parse(fs, args)
	if err != nil {
		return err
	}
	if len(pos) != 1 {
		return errors.New("omut hook start|commit [--agent claude|codex|cursor] [--strict]")
	}
	if *agent != "" && !slices.Contains(agents, *agent) {
		return fmt.Errorf("--agent %q: known are claude, codex, cursor", *agent)
	}
	dir := "."
	if *agent != "" {
		if in := readInput(stdin); in.Cwd != "" {
			dir = in.Cwd
		} else if len(in.WorkspaceRoots) > 0 {
			dir = in.WorkspaceRoots[0]
		}
	}
	switch pos[0] {
	case "start":
		text := hook.Start(dir)
		if text == "" {
			return nil
		}
		if *agent == "cursor" {
			return json.NewEncoder(out).Encode(map[string]string{"additional_context": text})
		}
		fmt.Fprint(out, text)
		return nil
	case "commit":
		res := hook.Commit(dir, *strict)
		fmt.Fprint(stderr, res.Message)
		if res.Block {
			return exitCode(1)
		}
		return nil
	}
	return fmt.Errorf("no hook %q: start or commit", pos[0])
}

// readInput reads the agent's JSON payload when stdin is not a terminal.
// A payload omut can't read is not an error: the hook falls back to the
// current directory.
func readInput(stdin *os.File) agentInput {
	var in agentInput
	if stdin == nil {
		return in
	}
	if info, err := stdin.Stat(); err != nil || info.Mode()&os.ModeCharDevice != 0 {
		return in
	}
	b, err := io.ReadAll(io.LimitReader(stdin, 1<<20))
	if err == nil {
		json.Unmarshal(b, &in)
	}
	return in
}
