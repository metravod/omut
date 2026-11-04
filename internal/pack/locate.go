package pack

import (
	"bufio"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Place says where the meta of a directory is. A clone keeps one memory, in
// its main checkout (D17): a linked worktree reads the project layer from its
// own branch when that has one, and this copy's layer from the main checkout.
type Place struct {
	Dir      string // holds .omut/; "" when there is none
	Instance string // holds this copy's .omut/local/
	Worktree string // the linked worktree dir is in, when Dir is not in it
	Foreign  bool   // Dir is above the git work tree dir is in: not its project
}

// Locate walks up from dir to the nearest .omut/. In a linked worktree that
// has none, it goes on from the same path in the main checkout.
func Locate(dir string) Place {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return Place{}
	}
	top := gitTop(abs)
	main := ""
	if top != "" {
		main = MainCheckout(top)
	}
	rel := relIn(top, abs)
	for d, above := abs, false; ; {
		if hasMeta(d) {
			pl := Place{Dir: d, Instance: d, Foreign: above}
			if main != "" && !above {
				pl.Instance = filepath.Join(main, relIn(top, d))
			}
			return pl
		}
		if main != "" && !above && same(d, top) {
			for m := filepath.Join(main, rel); ; m = filepath.Dir(m) {
				if hasMeta(m) {
					return Place{Dir: m, Instance: m, Worktree: top}
				}
				if same(m, main) {
					break
				}
			}
		}
		if top != "" && same(d, top) {
			above = true
		}
		parent := filepath.Dir(d)
		if parent == d {
			return Place{}
		}
		d = parent
	}
}

func hasMeta(dir string) bool {
	info, err := os.Stat(filepath.Join(dir, MetaDir))
	return err == nil && info.IsDir()
}

// same compares paths as the file system sees them: git prints resolved ones,
// the working directory may go through a symlink.
func same(a, b string) bool {
	return real(a) == real(b)
}

func real(path string) string {
	if r, err := filepath.EvalSymlinks(path); err == nil {
		return r
	}
	return path
}

// relIn is path relative to top, "." when it can't be told.
func relIn(top, path string) string {
	if top == "" {
		return "."
	}
	rel, err := filepath.Rel(real(top), real(path))
	if err != nil {
		return "."
	}
	return rel
}

func gitTop(dir string) string {
	out, err := exec.Command("git", "-C", dir, "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// LinkedWorktree reports whether dir is the root of a linked git worktree,
// whose .git is a file pointing into the main repository's worktrees/.
func LinkedWorktree(dir string) bool {
	b, err := os.ReadFile(filepath.Join(dir, ".git"))
	if err != nil {
		return false
	}
	text := filepath.ToSlash(strings.TrimSpace(string(b)))
	return strings.HasPrefix(text, "gitdir:") && strings.Contains(text, "/worktrees/")
}

// MainCheckout is the main working tree when dir is in a linked worktree, ""
// otherwise, and in a bare repository's worktrees, which have no main one.
func MainCheckout(dir string) string {
	top := gitTop(dir)
	if top == "" || !LinkedWorktree(top) {
		return ""
	}
	out, err := exec.Command("git", "-C", top, "worktree", "list", "--porcelain").Output()
	if err != nil {
		return ""
	}
	main := ""
	for sc := bufio.NewScanner(strings.NewReader(string(out))); sc.Scan(); {
		line := sc.Text()
		if line == "" {
			break
		}
		if line == "bare" {
			return ""
		}
		if path, ok := strings.CutPrefix(line, "worktree "); ok {
			main = path
		}
	}
	return main
}

// MainPath is where dir would be in the main checkout, "" outside a linked
// worktree.
func MainPath(dir string) string {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return ""
	}
	main := MainCheckout(abs)
	if main == "" {
		return ""
	}
	return filepath.Join(main, relIn(gitTop(abs), abs))
}
