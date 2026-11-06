package pack

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/metravod/omut/internal/meta"
)

// ReadPlan follows only an explicit plan pointer, never links in the plan.
// Project plans are relative to the project; instance plans to local/ beside
// their owner. Existing Markdown formats need no migration or frontmatter.
func ReadPlan(project string, owner *meta.Doc, local bool) (*meta.Doc, error) {
	ref := owner.Front["plan"]
	if ref == "" {
		return nil, nil
	}
	if !filepath.IsLocal(ref) || filepath.Ext(ref) != ".md" {
		return nil, fmt.Errorf("plan must be a relative .md file within its layer")
	}
	base := project
	if local {
		base = filepath.Dir(owner.Path)
	}
	base, err := filepath.EvalSymlinks(base)
	if err != nil {
		return nil, err
	}
	path, err := filepath.EvalSymlinks(filepath.Join(base, ref))
	if err != nil {
		return nil, err
	}
	if !within(base, path) {
		return nil, fmt.Errorf("plan resolves outside its layer")
	}
	if !local {
		instance := filepath.Join(base, MetaDir, Local)
		if real, err := filepath.EvalSymlinks(instance); err == nil {
			instance = real
		}
		if within(instance, path) || within(filepath.Join(base, MetaDir, Local), filepath.Join(base, ref)) {
			return nil, fmt.Errorf("local plan belongs in local/instance.md, not the project brief")
		}
	}
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("plan must be a regular Markdown file")
	}
	// Preserve the pointer's path for provenance and relative links.
	return meta.Read(filepath.Join(base, ref))
}

func within(base, path string) bool {
	// Paths may spell the same directory differently on a case-insensitive
	// filesystem. Compare directory identities, not a case-sensitive prefix.
	baseInfo, err := os.Stat(base)
	if err != nil {
		return false
	}
	for {
		if info, err := os.Stat(path); err == nil && os.SameFile(baseInfo, info) {
			return true
		}
		parent := filepath.Dir(path)
		if parent == path {
			return false
		}
		path = parent
	}
}
