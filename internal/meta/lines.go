package meta

import (
	"regexp"
	"strconv"
	"strings"
)

// Decision is one "- Dn [status] text" line of decisions.md.
type Decision struct {
	Line       int
	N          int
	Status     string // open | accepted | doubted | superseded
	Supersedes int    // for superseded: the Dn that replaced it
	Text       string
}

var (
	decisionRe   = regexp.MustCompile(`^- D(\d+) \[([^\]]*)\]\s*(.*)$`)
	supersededRe = regexp.MustCompile(`^superseded → D(\d+)$`)
)

// Decisions returns the well-formed decision lines and the 1-based numbers
// of lines that look like a decision but break the format.
func (d *Doc) Decisions() (out []Decision, bad []int) {
	for i := d.BodyStart; i < len(d.Clean); i++ {
		line := d.Clean[i]
		if !strings.HasPrefix(line, "- D") {
			continue
		}
		m := decisionRe.FindStringSubmatch(line)
		if m == nil {
			bad = append(bad, i+1)
			continue
		}
		n, _ := strconv.Atoi(m[1])
		dec := Decision{Line: i + 1, N: n, Text: m[3]}
		inner := strings.TrimSpace(m[2])
		status, _, _ := strings.Cut(inner, " ")
		switch status {
		case "open", "accepted", "doubted":
		case "superseded":
			sm := supersededRe.FindStringSubmatch(inner)
			if sm == nil {
				bad = append(bad, i+1)
				continue
			}
			dec.Supersedes, _ = strconv.Atoi(sm[1])
		default:
			bad = append(bad, i+1)
			continue
		}
		dec.Status = status
		out = append(out, dec)
	}
	return out, bad
}

// LogEntry is one "- date · author · text" line of log.md.
type LogEntry struct {
	Line   int
	Date   string
	Author string
	Text   string
}

var logRe = regexp.MustCompile(`^- (\d{4}-\d{2}-\d{2}) · ([^·]+?) · (.+)$`)

func (d *Doc) LogEntries() (out []LogEntry, bad []int) {
	for i := d.BodyStart; i < len(d.Clean); i++ {
		line := d.Clean[i]
		if !strings.HasPrefix(line, "- ") {
			continue
		}
		m := logRe.FindStringSubmatch(line)
		if m == nil {
			bad = append(bad, i+1)
			continue
		}
		if _, ok := ParseDate(m[1]); !ok {
			bad = append(bad, i+1)
			continue
		}
		out = append(out, LogEntry{Line: i + 1, Date: m[1], Author: strings.TrimSpace(m[2]), Text: m[3]})
	}
	return out, bad
}

// Ref is a path the meta points at: a markdown link target or a path
// after "→".
type Ref struct {
	Line   int
	Target string
}

var (
	mdLinkRe    = regexp.MustCompile(`\]\(([^)\s]+)\)`)
	codeArrowRe = regexp.MustCompile("`[^`]*→[^`]*`")
	pathishRe   = regexp.MustCompile(`^[\p{L}\p{N}_.\-/]+\.(md|txt|toml|ya?ml|json|py|go|sh|sql|csv)$|^[\p{L}\p{N}_.\-]+(/[\p{L}\p{N}_.\-]*)+/$`)
)

func (d *Doc) Refs() []Ref {
	var out []Ref
	for i, line := range d.Clean {
		for _, m := range mdLinkRe.FindAllStringSubmatch(line, -1) {
			if t := refTarget(m[1]); t != "" {
				out = append(out, Ref{Line: i + 1, Target: t})
			}
		}
		// A code span with an arrow inside is an example of the syntax,
		// not a reference.
		line = codeArrowRe.ReplaceAllString(line, "")
		for _, part := range strings.Split(line, "→")[1:] {
			for _, tok := range strings.FieldsFunc(part, func(r rune) bool {
				return r == ' ' || r == ',' || r == ';' || r == '\t'
			}) {
				tok = strings.Trim(tok, "`*()«»\"'")
				tok = strings.TrimRight(tok, ".:")
				if base, _, _ := strings.Cut(tok, "#"); pathishRe.MatchString(base) {
					if t := refTarget(tok); t != "" {
						out = append(out, Ref{Line: i + 1, Target: t})
					}
				}
			}
		}
	}
	return out
}

// refTarget drops anchors and rejects URLs and absolute paths, which the
// meta may mention but check cannot judge.
func refTarget(s string) string {
	if strings.Contains(s, "://") || strings.HasPrefix(s, "mailto:") || strings.HasPrefix(s, "/") || strings.HasPrefix(s, "~") {
		return ""
	}
	s, _, _ = strings.Cut(s, "#")
	return s
}

// Mentions returns the 1-based lines where word occurs as a whole word,
// comments excluded.
func (d *Doc) Mentions(word string) []int {
	re := WordRe(word)
	var out []int
	for i, line := range d.Clean {
		if re.MatchString(line) {
			out = append(out, i+1)
		}
	}
	return out
}

func WordRe(word string) *regexp.Regexp {
	return regexp.MustCompile(`(^|[^\p{L}\p{N}_\-])` + regexp.QuoteMeta(word) + `($|[^\p{L}\p{N}_\-])`)
}
