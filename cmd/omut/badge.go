package main

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

const badgeRecipe = "https://github.com/metravod/omut/blob/main/docs/hooks.md#ci-and-the-badge"

var githubRe = regexp.MustCompile(`^(?:https://github\.com/|git@github\.com:|ssh://git@github\.com/)([\w.-]+)/([\w.-]+?)(?:\.git)?/?$`)

// badge is the README line for the omut badge when dir is the root of a
// git repository whose origin is on GitHub; "" otherwise. The badge goes
// green when CI runs omut check --fail-on warn (badgeRecipe).
func badge(dir string) string {
	git := func(args ...string) string {
		out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).Output()
		if err != nil {
			return ""
		}
		return strings.TrimSpace(string(out))
	}
	top := git("rev-parse", "--show-toplevel")
	if top == "" {
		return ""
	}
	real, err := filepath.EvalSymlinks(dir)
	if err != nil {
		return ""
	}
	if top, err = filepath.EvalSymlinks(top); err != nil || top != real {
		return ""
	}
	m := githubRe.FindStringSubmatch(git("remote", "get-url", "origin"))
	if m == nil {
		return ""
	}
	branch := strings.TrimPrefix(git("symbolic-ref", "--short", "refs/remotes/origin/HEAD"), "origin/")
	if branch == "" {
		branch = "main"
	}
	return fmt.Sprintf("[![omut](https://github.com/%s/%s/actions/workflows/omut.yml/badge.svg?branch=%s)](.omut/)", m[1], m[2], branch)
}
