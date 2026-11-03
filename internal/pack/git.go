package pack

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Private reports whether a project's meta is this copy's alone (D16):
// dir is in a git work tree, git ignores .omut/ and tracks nothing in it.
// That is what `omut start --private` sets up through .git/info/exclude,
// and what a repository that lists .omut/ in its .gitignore gets as well.
// Outside git there is nothing to share, so nothing is private.
func Private(dir string) bool {
	if exec.Command("git", "-C", dir, "rev-parse", "--is-inside-work-tree").Run() != nil {
		return false
	}
	if out, _ := exec.Command("git", "-C", dir, "ls-files", "--", MetaDir).Output(); len(out) > 0 {
		return false
	}
	return exec.Command("git", "-C", dir, "check-ignore", "-q", MetaDir+"/"+Brief).Run() == nil
}

// ErrNoGit: --private hides the meta from git, so it needs git.
var ErrNoGit = errors.New("--private keeps the meta out of git through .git/info/exclude, and this directory is not in a git repository")

const excludeMark = "# omut start --private: this copy's meta, never committed"

// ExcludePrivate adds .omut/ and CLAUDE.local.md of dir to the repository's
// .git/info/exclude, which is not committed: the repository itself stays
// unchanged, byte for byte. It reports whether the file changed.
func ExcludePrivate(dir string) (bool, error) {
	out, err := exec.Command("git", "-C", dir, "rev-parse", "--git-path", "info/exclude", "--show-prefix").Output()
	if err != nil {
		return false, ErrNoGit
	}
	lines := strings.Split(strings.TrimRight(string(out), "\n"), "\n")
	path, prefix := lines[0], ""
	if len(lines) > 1 {
		prefix = lines[1]
	}
	if !filepath.IsAbs(path) {
		path = filepath.Join(dir, path)
	}
	have, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return false, err
	}
	text := string(have)
	var add []string
	for _, p := range []string{"/" + prefix + MetaDir + "/", "/" + prefix + ClaudeLocal} {
		if !hasLine(text, p) {
			add = append(add, p)
		}
	}
	if len(add) == 0 {
		return false, nil
	}
	if text != "" && !strings.HasSuffix(text, "\n") {
		text += "\n"
	}
	if !hasLine(text, excludeMark) {
		text += excludeMark + "\n"
	}
	text += strings.Join(add, "\n") + "\n"
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return false, err
	}
	return true, os.WriteFile(path, []byte(text), 0o644)
}

func hasLine(text, line string) bool {
	for _, l := range strings.Split(text, "\n") {
		if strings.TrimSpace(l) == line {
			return true
		}
	}
	return false
}
