package check

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/metravod/omut/internal/meta"
	"github.com/metravod/omut/internal/pack"
)

// The instance layer (D15): .omut/local/ holds what is true only for this
// copy of the project. It must stay out of git, and the project layer must
// not carry its facts.

// checkInstance checks .omut/local/: kept out of git, its log in shape,
// its "Where it lives" checked against reality when it says anything.
func checkInstance(r *report, dir string, opt Options, active bool) {
	local := filepath.Join(dir, pack.MetaDir, pack.Local)
	if _, err := os.Stat(local); err != nil {
		return
	}
	rel := pack.MetaDir + "/" + pack.Local
	if isRepo(dir) {
		if out, _ := exec.Command("git", "-C", dir, "ls-files", "--", rel).Output(); len(out) > 0 {
			r.add("local-in-git", Error, local, 0, "the instance layer is in git, so every clone gets this copy's facts: git rm -r --cached %s", rel)
		} else if exec.Command("git", "-C", dir, "check-ignore", "-q", rel+"/"+pack.Instance).Run() != nil {
			r.add("local-not-ignored", Warn, local, 0, "git does not ignore %s, the next git add -A takes it: omut zero writes .omut/.gitignore", rel)
		}
	}
	if d, err := meta.Read(filepath.Join(local, pack.Log)); err == nil {
		checkLog(r, d)
	}
	if d, err := meta.Read(filepath.Join(local, pack.Instance)); err == nil && active {
		checkClaims(r, d, opt, instanceSections...)
	}
}

// Sections whose "- " lines are claims about reality (D11).
var (
	briefClaimSections = []string{"Where it lives"}
	instanceSections   = []string{"Where it lives", "Access"}
)

// checkClaims looks at what the meta says about reality without looking
// at reality: a claim with a way to check it ("how:") must have been seen
// in the last VerifyDays days; claims without one are counted. Agents run
// the checks; omut only remembers them (D11).
func checkClaims(r *report, d *meta.Doc, opt Options, sections ...string) {
	today := opt.today()
	var unchecked []meta.Claim
	for _, c := range d.Claims(sections...) {
		if c.How == "" {
			unchecked = append(unchecked, c)
			continue
		}
		if c.Seen == "" {
			r.add("unseen", Warn, d.Path, c.Line, "never checked: %s (how: %s)", short(c.Text), c.How)
			continue
		}
		t, ok := meta.ParseDate(c.Seen)
		if !ok {
			r.add("shape", Error, d.Path, c.Line, "seen %q: want a YYYY-MM-DD date", c.Seen)
			continue
		}
		if today.Sub(t) > VerifyDays*24*time.Hour {
			r.add("unseen", Warn, d.Path, c.Line, "last checked %s, more than %d days ago: %s (how: %s)", c.Seen, VerifyDays, short(c.Text), c.How)
		}
	}
	if len(unchecked) > 0 {
		r.add("unchecked", Info, d.Path, unchecked[0].Line, "%d of the claims about where it lives say no way to check them (\" · how: ...\"), first: %s", len(unchecked), short(unchecked[0].Text))
	}
}

// instanceLive says whether this copy declares a footprint.
func instanceLive(dir string) bool {
	d, err := meta.Read(filepath.Join(dir, pack.MetaDir, pack.Local, pack.Instance))
	if err != nil {
		return false
	}
	s, ok := d.Section("Where it lives")
	return ok && d.Filled(s)
}

var (
	homePathRe = regexp.MustCompile(`(?:/Users|/home)/[\w.-]+/`)
	tildeRe    = regexp.MustCompile("(?:^|[\\s`(\"'])~/")
	ipv4Re     = regexp.MustCompile(`(\d{1,3})\.(\d{1,3})\.(\d{1,3})\.(\d{1,3})`)
)

// leakIn returns what in a line of the project layer looks like a fact
// about one copy: a home directory, an IP address. "~/" counts only where
// the brief says where the project lives; elsewhere it is usually a
// general path such as ~/.claude/settings.json.
func leakIn(line string, tilde bool) string {
	if m := homePathRe.FindString(line); m != "" {
		return m
	}
	if ip := findIPv4(line); ip != "" {
		return ip
	}
	if tilde && tildeRe.MatchString(line) {
		return "~/"
	}
	return ""
}

// checkLeak warns about facts of this copy in the project layer, which
// goes into git and to everyone who clones it.
func checkLeak(r *report, lead string) {
	var files []string
	for _, f := range []string{pack.Brief, pack.Decisions, pack.Log} {
		files = append(files, filepath.Join(lead, f))
	}
	threads, _ := filepath.Glob(filepath.Join(lead, pack.Threads, "*.md"))
	files = append(files, threads...)
	for _, path := range files {
		d, err := meta.Read(path)
		if err != nil {
			continue
		}
		var lives *meta.Section
		if filepath.Base(path) == pack.Brief {
			if s, ok := d.Section("Where it lives"); ok {
				lives = &s
			}
		}
		first, firstWhat, n := 0, "", 0
		for i, line := range d.Clean {
			inLives := lives != nil && i >= lives.Start && i < lives.End
			if what := leakIn(line, inLives); what != "" {
				if n == 0 {
					first, firstWhat = i+1, what
				}
				n++
			}
		}
		if n > 0 {
			more := ""
			if n > 1 {
				more = fmt.Sprintf(" (%d lines like it)", n)
			}
			r.add("leak", Warn, path, first, "%q looks like a fact about this copy, not the project%s: move it to .omut/local/instance.md, which stays out of git", strings.TrimSpace(firstWhat), more)
		}
	}
}

// findIPv4 finds an IPv4 address that is not part of something longer: a
// version (v1.2.3.4, 1.2.3.4-beta), a longer dotted number. Loopback and
// 0.0.0.0 say nothing about a copy.
func findIPv4(line string) string {
	for _, m := range ipv4Re.FindAllStringSubmatchIndex(line, -1) {
		start, end := m[0], m[1]
		if start > 0 {
			if c := line[start-1]; c == '.' || c == 'v' || c == 'V' || isWord(c) {
				continue
			}
		}
		if end < len(line) {
			c := line[end]
			if isWord(c) || c == '-' || c == '+' || c == '.' && end+1 < len(line) && isWord(line[end+1]) {
				continue
			}
		}
		ok := true
		for i := 2; i < 10; i += 2 {
			if n, _ := strconv.Atoi(line[m[i]:m[i+1]]); n > 255 {
				ok = false
			}
		}
		if ip := line[start:end]; ok && ip != "127.0.0.1" && ip != "0.0.0.0" {
			return ip
		}
	}
	return ""
}

func isWord(c byte) bool {
	return c == '_' || c >= '0' && c <= '9' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z'
}

// DueDays: a claim seen longer ago than this is offered to the agent at
// session start, well before it turns into an unseen warning.
const DueDays = 7

// DueClaim is a claim to check now, with the file it is in, relative to
// the project.
type DueClaim struct {
	File string
	meta.Claim
}

// Due picks up to n claims an agent should check at the start of a
// session: those with a way to check them, never seen first, then the
// oldest, none seen in the last DueDays days. It also counts the claims
// with no way to check them. Nothing for a frozen project. This is how
// the audit happens a little at a time instead of in a ritual nobody runs.
func Due(dir string, n int, opt Options) (due []DueClaim, unchecked int) {
	lead := filepath.Join(dir, pack.MetaDir)
	brief, err := meta.Read(filepath.Join(lead, pack.Brief))
	if err != nil || brief.Front["status"] == "frozen" {
		return nil, 0
	}
	var all []DueClaim
	collect := func(d *meta.Doc, sections ...string) {
		rel, _ := filepath.Rel(dir, d.Path)
		for _, c := range d.Claims(sections...) {
			if c.How == "" {
				unchecked++
				continue
			}
			all = append(all, DueClaim{File: filepath.ToSlash(rel), Claim: c})
		}
	}
	collect(brief, briefClaimSections...)
	if d, err := meta.Read(filepath.Join(lead, pack.Local, pack.Instance)); err == nil {
		collect(d, instanceSections...)
	}
	cutoff := opt.today().Add(-DueDays * 24 * time.Hour).Format(meta.DateLayout)
	all = slices.DeleteFunc(all, func(c DueClaim) bool { return c.Seen != "" && c.Seen >= cutoff })
	// "" sorts before any date, so never-seen claims come first.
	slices.SortStableFunc(all, func(a, b DueClaim) int { return strings.Compare(a.Seen, b.Seen) })
	if len(all) > n {
		all = all[:n]
	}
	return all, unchecked
}
