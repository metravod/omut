package context

import (
	"bytes"
	"cmp"
	stdcontext "context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"
)

// GitState is an observation, separate from the project's recorded meta.
// Clean and divergence counts are null when unknown. Change paths are relative
// to Root; Directory is the project/worktree directory the status is scoped to.
type GitState struct {
	State     string      `json:"state"` // available | not-repository | unavailable
	Directory string      `json:"directory"`
	Root      string      `json:"root,omitempty"`
	Head      string      `json:"head,omitempty"`
	Branch    string      `json:"branch,omitempty"`
	Detached  bool        `json:"detached"`
	Unborn    bool        `json:"unborn"`
	Upstream  string      `json:"upstream,omitempty"`
	Ahead     *int        `json:"ahead"`
	Behind    *int        `json:"behind"`
	Clean     *bool       `json:"clean"`
	Conflicts int         `json:"conflicts"`
	Changes   []GitChange `json:"changes"`
	Error     string      `json:"error,omitempty"`
}

type GitChange struct {
	Status       string `json:"status"` // two columns: index/worktree, or ??
	Path         string `json:"path"`
	OriginalPath string `json:"original_path,omitempty"`
	Submodule    string `json:"submodule,omitempty"` // raw S<c><m><u> state
	Conflict     bool   `json:"conflict,omitempty"`
}

func (r *Report) git(dir string) *GitState {
	g := &GitState{State: "not-repository", Directory: resolved(dir), Changes: []GitChange{}}
	// Git worktrees, submodules and separate git dirs all carry a .git file
	// or directory. Walk upward for projects inside a larger repository.
	var err error
	for parent := g.Directory; ; parent = filepath.Dir(parent) {
		_, err = os.Lstat(filepath.Join(parent, ".git"))
		if err == nil || !os.IsNotExist(err) || parent == filepath.Dir(parent) {
			break
		}
	}
	if os.IsNotExist(err) {
		return g
	}
	if err == nil {
		var top []byte
		top, err = runGit(g.Directory, "rev-parse", "--show-toplevel")
		if err == nil {
			g.Root = resolved(strings.TrimSuffix(string(top), "\n"))
			var status []byte
			status, err = runGit(g.Directory, "status", "--porcelain=v2", "--branch", "--ahead-behind", "--renames", "--ignore-submodules=none", "--untracked-files=all", "-z", "--", ".")
			if err == nil {
				err = g.parse(status)
			}
		}
	}
	if err != nil {
		g.State, g.Error = "unavailable", err.Error()
		g.Clean = nil
		r.problem(dir, fmt.Errorf("Git observation unavailable: %w", err))
		return g
	}
	g.State = "available"
	clean := len(g.Changes) == 0
	g.Clean = &clean
	return g
}

func runGit(dir string, args ...string) ([]byte, error) {
	ctx, cancel := stdcontext.WithTimeout(stdcontext.Background(), 30*time.Second)
	defer cancel()
	// Optional index refresh and fsmonitor hooks are unnecessary for a read.
	cmd := exec.CommandContext(ctx, "git", append([]string{"--no-optional-locks", "-c", "core.fsmonitor=false", "-c", "status.relativePaths=false", "-C", dir}, args...)...)
	cmd.WaitDelay = time.Second
	// Repository selection belongs to dir, including when omut is invoked
	// from a Git hook with GIT_DIR or GIT_INDEX_FILE set.
	for _, env := range os.Environ() {
		key, _, _ := strings.Cut(env, "=")
		if !slices.Contains([]string{"GIT_DIR", "GIT_WORK_TREE", "GIT_INDEX_FILE", "GIT_COMMON_DIR", "GIT_PREFIX", "GIT_OBJECT_DIRECTORY", "GIT_ALTERNATE_OBJECT_DIRECTORIES", "GIT_NAMESPACE"}, key) {
			cmd.Env = append(cmd.Env, env)
		}
	}
	cmd.Env = append(cmd.Env, "LC_ALL=C")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if ctx.Err() != nil {
		return nil, fmt.Errorf("git %s: %w", args[0], ctx.Err())
	}
	if err != nil {
		return nil, fmt.Errorf("git %s: %w: %s", args[0], err, strings.TrimSpace(stderr.String()))
	}
	return out, nil
}

// Parse only the documented fields; NUL framing preserves all filename bytes,
// including tabs/newlines and both sides of a rename. Unknown headers are valid.
// https://git-scm.com/docs/git-status#_porcelain_format_version_2
func (g *GitState) parse(data []byte) error {
	if len(data) == 0 || data[len(data)-1] != 0 {
		return fmt.Errorf("incomplete porcelain-v2 output")
	}
	records := strings.Split(string(data[:len(data)-1]), "\x00")
	headSeen, branchSeen := false, false
	for i := 0; i < len(records); i++ {
		record := records[i]
		switch {
		case strings.HasPrefix(record, "# branch.oid "):
			g.Head, headSeen = strings.TrimPrefix(record, "# branch.oid "), true
			if g.Head == "(initial)" {
				g.Head, g.Unborn = "", true
			}
		case strings.HasPrefix(record, "# branch.head "):
			g.Branch, branchSeen = strings.TrimPrefix(record, "# branch.head "), true
			if g.Branch == "(detached)" {
				g.Branch, g.Detached = "", true
			}
		case strings.HasPrefix(record, "# branch.upstream "):
			g.Upstream = strings.TrimPrefix(record, "# branch.upstream ")
		case strings.HasPrefix(record, "# branch.ab "):
			var ahead, behind int
			n, err := fmt.Sscanf(record, "# branch.ab +%d -%d", &ahead, &behind)
			if err != nil || n != 2 || ahead < 0 || behind < 0 {
				return fmt.Errorf("invalid porcelain-v2 divergence header")
			}
			g.Ahead, g.Behind = &ahead, &behind
		case strings.HasPrefix(record, "# "):
		case strings.HasPrefix(record, "? "):
			g.Changes = append(g.Changes, GitChange{Status: "??", Path: record[2:]})
		default:
			if len(record) < 2 {
				return fmt.Errorf("invalid porcelain-v2 record")
			}
			fields := 9
			switch record[0] {
			case '1':
			case '2':
				fields = 10
			case 'u':
				fields = 11
			default:
				return fmt.Errorf("unknown porcelain-v2 record type %q", record[0])
			}
			f := strings.SplitN(record, " ", fields)
			if len(f) != fields || len(f[1]) != 2 || len(f[2]) != 4 || f[fields-1] == "" {
				return fmt.Errorf("invalid porcelain-v2 changed-file record")
			}
			change := GitChange{Status: strings.ReplaceAll(f[1], ".", " "), Path: f[fields-1], Conflict: record[0] == 'u'}
			if f[2] != "N..." {
				change.Submodule = f[2]
			}
			if record[0] == '2' {
				i++
				if i >= len(records) || records[i] == "" {
					return fmt.Errorf("missing porcelain-v2 rename source")
				}
				change.OriginalPath = records[i]
			}
			if change.Conflict {
				g.Conflicts++
			}
			g.Changes = append(g.Changes, change)
		}
	}
	if !headSeen || !branchSeen {
		return fmt.Errorf("missing porcelain-v2 branch headers")
	}
	slices.SortFunc(g.Changes, func(a, b GitChange) int {
		return cmp.Or(cmp.Compare(a.Path, b.Path), cmp.Compare(a.OriginalPath, b.OriginalPath))
	})
	return nil
}

func (g *GitState) markdown() string {
	var b strings.Builder
	b.WriteString("\n### Git observation\n\n")
	switch g.State {
	case "not-repository":
		b.WriteString("No Git working tree.\n")
	case "unavailable":
		fmt.Fprintf(&b, "Unavailable: %s\n", g.Error)
	default:
		fmt.Fprintf(&b, "Repository: %s\nScope: %s (changes below this project only; paths are relative to the repository).\n", strconv.Quote(g.Root), strconv.Quote(g.Directory))
		switch {
		case g.Detached:
			fmt.Fprintf(&b, "Detached HEAD: %s\n", g.Head)
		case g.Unborn:
			fmt.Fprintf(&b, "Branch: %s (no commits yet)\n", strconv.Quote(g.Branch))
		default:
			fmt.Fprintf(&b, "Branch: %s\n", strconv.Quote(g.Branch))
		}
		if g.Upstream == "" {
			b.WriteString("Upstream: none reported.\n")
		} else if g.Ahead == nil || g.Behind == nil {
			fmt.Fprintf(&b, "Upstream: %s; ahead/behind unavailable.\n", strconv.Quote(g.Upstream))
		} else {
			fmt.Fprintf(&b, "Upstream: %s; ahead %d, behind %d (local refs; no fetch).\n", strconv.Quote(g.Upstream), *g.Ahead, *g.Behind)
		}
		fmt.Fprintf(&b, "Clean in scope: %t · conflicts: %d\n", *g.Clean, g.Conflicts)
		if len(g.Changes) > 0 {
			b.WriteString("Changes (XY = index/worktree; ?? = untracked):\n\n")
		}
		for _, change := range g.Changes {
			fmt.Fprintf(&b, "- `%s` %s", change.Status, strconv.Quote(change.Path))
			if change.OriginalPath != "" {
				fmt.Fprintf(&b, " (from %s)", strconv.Quote(change.OriginalPath))
			}
			if change.Submodule != "" {
				fmt.Fprintf(&b, " [submodule %s]", change.Submodule)
			}
			b.WriteByte('\n')
		}
	}
	return b.String()
}
