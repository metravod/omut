package meta

import (
	"strings"
)

// Claim is one item of a footprint: a "- " line of "Where it lives" (or of
// "Access" in the instance layer), with how an agent checks it and when it
// last did (D11). omut never runs How; it remembers it and how old Seen is.
//
//   - skill adr: ~/.claude/skills/adr → this checkout · how: readlink ~/.claude/skills/adr · seen 2026-11-03
type Claim struct {
	Line int    // 1-based
	Text string // the claim itself, without how and seen
	How  string // "" when the line doesn't say how to check it
	Seen string // the raw date after "seen", "" when never
}

// Claims returns the claims of the named sections, in file order.
func (d *Doc) Claims(sections ...string) []Claim {
	var out []Claim
	for _, title := range sections {
		s, ok := d.Section(title)
		if !ok {
			continue
		}
		for i := s.Start; i < s.End; i++ {
			line := strings.TrimSpace(d.Clean[i])
			item, ok := strings.CutPrefix(line, "- ")
			if !ok {
				continue
			}
			c := Claim{Line: i + 1}
			var text []string
			for _, part := range strings.Split(item, " · ") {
				p := strings.TrimSpace(part)
				switch {
				case strings.HasPrefix(p, "how:"):
					c.How = strings.Trim(strings.TrimSpace(p[len("how:"):]), "`")
				case strings.HasPrefix(p, "seen "):
					c.Seen = strings.TrimSpace(p[len("seen "):])
				default:
					text = append(text, part)
				}
			}
			c.Text = strings.TrimSpace(strings.Join(text, " · "))
			out = append(out, c)
		}
	}
	return out
}
