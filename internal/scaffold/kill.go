package scaffold

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/metravod/omut/internal/meta"
	"github.com/metravod/omut/internal/pack"
)

// Mention is a line of live meta that names a project.
type Mention struct {
	File string // relative to the pack root
	Line int
	Text string
}

// Unsaved lists what would be lost: uncommitted changes, gitignored files
// no build recreates, and commits that are on no remote. A project without git has no copy anywhere, so all of
// it would be lost.
func Unsaved(dir string) []string {
	if _, err := os.Stat(filepath.Join(dir, ".git")); err != nil {
		return []string{"no git: the files exist only here"}
	}
	git := func(args ...string) (string, error) {
		out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).Output()
		return strings.TrimSpace(string(out)), err
	}
	var out []string
	if s, err := git("status", "--porcelain"); err != nil {
		out = append(out, "git status failed: "+err.Error())
	} else if s != "" {
		out = append(out, fmt.Sprintf("uncommitted changes: %d files", len(strings.Split(s, "\n"))))
	}
	if s, err := git("ls-files", "--others", "--ignored", "--exclude-standard", "--directory", "-z"); err == nil {
		if lost := ignoredLost(s); len(lost) > 0 {
			out = append(out, "gitignored, exists only here: "+lost)
		}
	}
	if _, err := git("rev-parse", "--verify", "HEAD"); err != nil {
		return out // no commits yet
	}
	if s, err := git("rev-list", "--count", "HEAD", "--not", "--remotes"); err != nil {
		out = append(out, "could not compare with remotes: "+err.Error())
	} else if s != "0" {
		out = append(out, "commits on no remote: "+s)
	}
	return out
}

// regenerable are gitignored names a build or a tool recreates; losing them
// loses nothing.
var regenerable = map[string]bool{
	".DS_Store": true, "node_modules": true, "target": true, "dist": true, "build": true,
	".venv": true, "venv": true, "__pycache__": true, ".pytest_cache": true, ".mypy_cache": true,
	".ruff_cache": true, ".tox": true, ".next": true, ".gradle": true, ".idea": true, ".vscode": true,
	"coverage": true, ".coverage": true, ".cache": true,
}

// ignoredLost names the gitignored paths (git ls-files -z output, ignored
// directories collapsed) that no build recreates: .env, local data, meta
// kept out of git. "" when there are none.
func ignoredLost(z string) string {
	var lost []string
	for _, p := range strings.Split(z, "\x00") {
		name := filepath.Base(strings.TrimSuffix(p, "/"))
		if p == "" || regenerable[name] || strings.HasSuffix(name, ".pyc") || strings.HasSuffix(name, ".egg-info") {
			continue
		}
		lost = append(lost, p)
	}
	const shown = 5
	if len(lost) > shown {
		return fmt.Sprintf("%s and %d more", strings.Join(lost[:shown], ", "), len(lost)-shown)
	}
	return strings.Join(lost, ", ")
}

// Kill removes a project and writes its tombstone. The guards (--yes,
// Unsaved) are the caller's job; Kill only does the work. A symlinked
// project loses only the link, the files it points to stay.
func Kill(p *pack.Pack, name, reason string, opt Options) (tombstone string, mentions []Mention, err error) {
	if err := pack.ValidName(name); err != nil {
		return "", nil, err
	}
	dir := p.Dir(name)
	if _, err := os.Lstat(dir); err != nil {
		return "", nil, fmt.Errorf("no project %s", name)
	}
	what := "what it was is not recorded"
	if d, err := meta.Read(filepath.Join(dir, pack.MetaDir, pack.Brief)); err == nil {
		if s, ok := d.Section("Why"); ok {
			if t := d.FirstText(s); t != "" {
				what = clip(t, 140)
			}
		}
	}
	if err := os.RemoveAll(dir); err != nil {
		return "", nil, err
	}
	tombstone = fmt.Sprintf("- %s — removed %s · %s · %s", name, opt.date(), reason, what)
	if err := appendLine(p.Lead(pack.Graveyard), tombstone); err != nil {
		return tombstone, nil, fmt.Errorf("the project is removed, but the tombstone is not written: %w", err)
	}
	return tombstone, findMentions(p, name), nil
}

func appendLine(path, line string) error {
	b, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	text := string(b)
	if text != "" && !strings.HasSuffix(text, "\n") {
		text += "\n"
	}
	return os.WriteFile(path, []byte(text+line+"\n"), 0o644)
}

// findMentions scans the meta of the pack and of the remaining projects,
// logs excluded: history may name the dead.
func findMentions(p *pack.Pack, name string) []Mention {
	var roots []string
	roots = append(roots, filepath.Join(p.Root, pack.MetaDir))
	names, _ := p.Projects()
	for _, n := range names {
		roots = append(roots, filepath.Join(p.Dir(n), pack.MetaDir))
	}
	var out []Mention
	for _, root := range roots {
		filepath.WalkDir(root, func(path string, e fs.DirEntry, err error) error {
			if err != nil || e.IsDir() || !strings.HasSuffix(path, ".md") {
				return nil
			}
			if e.Name() == pack.Log || e.Name() == pack.Graveyard || path == filepath.Join(root, pack.Readme) || strings.Contains(path, "log-archive") {
				return nil
			}
			d, err := meta.Read(path)
			if err != nil {
				return nil
			}
			for _, line := range d.Mentions(name) {
				rel, _ := filepath.Rel(p.Root, path)
				out = append(out, Mention{File: rel, Line: line, Text: strings.TrimSpace(d.Lines[line-1])})
			}
			return nil
		})
	}
	return out
}

func clip(s string, n int) string {
	if r := []rune(s); len(r) > n {
		return string(r[:n-1]) + "…"
	}
	return s
}
