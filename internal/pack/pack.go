// Package pack finds a pack root and lists its projects. The list is read
// from the directory every time; nothing is registered anywhere.
package pack

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/metravod/omut/internal/meta"
)

const (
	MetaDir     = ".omut"
	LeadFile    = "LEAD.md"
	Readme      = "README.md"
	Brief       = "brief.md"
	Decisions   = "decisions.md"
	Graveyard   = "graveyard.md"
	Log         = "log.md"
	Threads     = "threads"
	Agents      = "AGENTS.md"
	Claude      = "CLAUDE.md"
	ClaudeLocal = "CLAUDE.local.md"

	BlockStart = "<!-- omut:start -->"
	BlockEnd   = "<!-- omut:end -->"
)

var ErrNoPack = errors.New("not inside a pack: no .omut/LEAD.md here or above (omut init creates one)")

type Pack struct {
	Root string
}

// IsRoot reports whether dir holds a pack: .omut/LEAD.md is the mark.
func IsRoot(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, MetaDir, LeadFile))
	return err == nil
}

// Find walks up from dir to the nearest pack root.
func Find(dir string) (*Pack, error) {
	dir, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	for {
		if IsRoot(dir) {
			return &Pack{Root: dir}, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return nil, ErrNoPack
		}
		dir = parent
	}
}

func (p *Pack) Lead(file string) string { return filepath.Join(p.Root, MetaDir, file) }

func (p *Pack) Dir(name string) string { return filepath.Join(p.Root, name) }

// Ignore is the "ignore:" list of the pack brief: root directories that
// are not projects.
func (p *Pack) Ignore() []string {
	d, err := meta.Read(p.Lead(Brief))
	if err != nil {
		return nil
	}
	return d.Lists["ignore"]
}

// Projects lists the root's subdirectories that count as projects:
// everything except hidden ones (.omut/ among them) and the ignore list. Symlinked
// directories count, so a pack can be assembled from links.
func (p *Pack) Projects() ([]string, error) {
	entries, err := os.ReadDir(p.Root)
	if err != nil {
		return nil, err
	}
	ignore := p.Ignore()
	var out []string
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, ".") || slices.Contains(ignore, name) {
			continue
		}
		info, err := os.Stat(filepath.Join(p.Root, name))
		if err != nil || !info.IsDir() {
			continue
		}
		out = append(out, name)
	}
	return out, nil
}

// Graveyard returns the names of killed projects.
func (p *Pack) Graveyard() []string {
	d, err := meta.Read(p.Lead(Graveyard))
	if err != nil {
		return nil
	}
	var out []string
	for i := d.BodyStart; i < len(d.Clean); i++ {
		line, ok := strings.CutPrefix(d.Clean[i], "- ")
		if !ok {
			continue
		}
		if name, _, ok := strings.Cut(line, " — "); ok {
			out = append(out, strings.TrimSpace(name))
		}
	}
	return out
}

// ValidName rejects names that cannot be a project directory of the root.
func ValidName(name string) error {
	switch {
	case name == "" || name == "." || name == "..":
		return fmt.Errorf("empty project name")
	case strings.ContainsAny(name, `/\`):
		return fmt.Errorf("%q: a project is a directory right under the pack root, no /", name)
	case strings.HasPrefix(name, "."):
		return fmt.Errorf("%q: hidden directories are not projects", name)
	}
	return nil
}

// Resolve turns a command argument into a project name: either a bare
// name or a path (".", "../x") to a directory directly under the root.
func (p *Pack) Resolve(arg string) (string, error) {
	if !strings.ContainsAny(arg, `/\`) && arg != "." && arg != ".." {
		return arg, ValidName(arg)
	}
	abs, err := filepath.Abs(arg)
	if err != nil {
		return "", err
	}
	if filepath.Dir(abs) != p.Root {
		return "", fmt.Errorf("%s is not a project of the pack %s: a project is a directory right under the root", arg, p.Root)
	}
	name := filepath.Base(abs)
	return name, ValidName(name)
}
