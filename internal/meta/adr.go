package meta

import (
	"regexp"
	"strings"
)

// ADR is what check needs from an architecture decision record: its title
// and where it stands. ADR formats differ, so the status is looked for in
// frontmatter, a "Status:" line (also "* Status:", "**Status:**") or the
// first line of a "## Status" section.
type ADR struct {
	Path      string
	Title     string
	Status    string // the status's first word, lower case; "" when none is written
	Line      int    // 1-based line of the status, or of the title
	Tentative bool   // "Firmness: tentative" (stare-decisis)
}

var (
	statusLineRe = regexp.MustCompile(`(?i)^(?:[*-]\s+)?(?:\*\*)?status(?:\*\*)?\s*:\s*(?:\*\*)?\s*(.*)$`)
	firmnessRe   = regexp.MustCompile(`(?i)^(?:[*-]\s+)?(?:\*\*)?firmness(?:\*\*)?\s*:\s*(?:\*\*)?\s*tentative\b`)
)

// ParseADR reads an ADR's title and status.
func ParseADR(d *Doc) ADR {
	a := ADR{Path: d.Path}
	if s, ok := d.Front["status"]; ok {
		a.Status, a.Line = firstWord(s), d.FrontLine["status"]
	}
	if strings.EqualFold(strings.TrimSpace(d.Front["firmness"]), "tentative") {
		a.Tentative = true
	}
	inStatus := false
	for i := d.BodyStart; i < len(d.Clean); i++ {
		line := strings.TrimSpace(d.Clean[i])
		switch {
		case a.Title == "" && strings.HasPrefix(line, "# "):
			a.Title = strings.TrimSpace(line[2:])
			if a.Line == 0 {
				a.Line = i + 1
			}
		case strings.HasPrefix(line, "#"):
			inStatus = strings.EqualFold(strings.TrimSpace(strings.TrimLeft(line, "#")), "status")
		case firmnessRe.MatchString(line):
			a.Tentative = true
		case a.Status != "":
		case inStatus && line != "":
			a.Status, a.Line, inStatus = firstWord(line), i+1, false
		default:
			if m := statusLineRe.FindStringSubmatch(line); m != nil && m[1] != "" {
				a.Status, a.Line = firstWord(m[1]), i+1
			}
		}
	}
	return a
}

// Waiting says whether the ADR waits for the human, in decisions.md terms:
// "open" for a proposal or draft, "doubted" for a tentative decision still
// in force, "" otherwise. An ADR with no status written counts as decided.
func (a ADR) Waiting() string {
	switch a.Status {
	case "proposed", "draft", "open", "pending":
		return "open"
	case "superseded", "deprecated", "rejected", "withdrawn":
		return ""
	}
	if a.Tentative {
		return "doubted"
	}
	return ""
}

func firstWord(s string) string {
	s = strings.TrimLeft(strings.TrimSpace(s), "*_`[")
	if i := strings.IndexFunc(s, func(r rune) bool { return !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r == '-') }); i >= 0 {
		s = s[:i]
	}
	return strings.ToLower(s)
}
