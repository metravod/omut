// Package meta reads the markdown files a pack is made of: frontmatter,
// second-level sections, and the line formats of decisions and the log.
// It knows the shape of the files, not what they mean.
package meta

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"
)

const DateLayout = "2006-01-02"

// Doc is one markdown file split into frontmatter and body.
type Doc struct {
	Path  string
	Lines []string
	// Clean is Lines with HTML comments blanked out, index for index, so
	// line numbers found in it point at the real file.
	Clean     []string
	HasFront  bool
	BodyStart int // index of the first body line
	Front     map[string]string
	FrontLine map[string]int // 1-based
	Lists     map[string][]string
}

func Read(path string) (*Doc, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return Parse(path, string(b)), nil
}

func Parse(path, text string) *Doc {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	lines := strings.Split(strings.TrimSuffix(text, "\n"), "\n")
	d := &Doc{
		Path:      path,
		Lines:     lines,
		Clean:     stripComments(lines),
		Front:     map[string]string{},
		FrontLine: map[string]int{},
		Lists:     map[string][]string{},
	}
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return d
	}
	end := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			end = i
			break
		}
	}
	if end < 0 {
		return d
	}
	d.HasFront = true
	d.BodyStart = end + 1
	last := ""
	for i := 1; i < end; i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "- ") && last != "" {
			d.Lists[last] = append(d.Lists[last], strings.TrimSpace(trimmed[2:]))
			continue
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = stripTrailingComment(strings.TrimSpace(value))
		d.Front[key] = value
		d.FrontLine[key] = i + 1
		last = key
		if strings.HasPrefix(value, "[") && strings.HasSuffix(value, "]") {
			for _, item := range strings.Split(value[1:len(value)-1], ",") {
				if item = strings.TrimSpace(item); item != "" {
					d.Lists[key] = append(d.Lists[key], item)
				}
			}
		}
	}
	return d
}

// stripTrailingComment drops a YAML-style "# ..." that follows whitespace.
func stripTrailingComment(v string) string {
	if strings.HasPrefix(v, "#") {
		return ""
	}
	if i := strings.Index(v, " #"); i >= 0 {
		v = v[:i]
	}
	return strings.TrimSpace(v)
}

func stripComments(lines []string) []string {
	out := make([]string, len(lines))
	in := false
	for i, line := range lines {
		var b strings.Builder
		rest := line
		for rest != "" {
			if in {
				j := strings.Index(rest, "-->")
				if j < 0 {
					rest = ""
					break
				}
				rest = rest[j+3:]
				in = false
				continue
			}
			j := strings.Index(rest, "<!--")
			if j < 0 {
				b.WriteString(rest)
				break
			}
			b.WriteString(rest[:j])
			rest = rest[j+4:]
			in = true
		}
		out[i] = b.String()
	}
	return out
}

// Section is a "## Title" block of the body.
type Section struct {
	Title string
	Line  int // 1-based line of the heading
	Start int // index of the first line after the heading
	End   int // index past the last line
}

func (d *Doc) Sections() []Section {
	var out []Section
	for i := d.BodyStart; i < len(d.Clean); i++ {
		if t, ok := strings.CutPrefix(d.Clean[i], "## "); ok {
			if n := len(out); n > 0 {
				out[n-1].End = i
			}
			out = append(out, Section{Title: strings.TrimSpace(t), Line: i + 1, Start: i + 1, End: len(d.Clean)})
		}
	}
	return out
}

func (d *Doc) Section(title string) (Section, bool) {
	for _, s := range d.Sections() {
		if s.Title == title {
			return s, true
		}
	}
	return Section{}, false
}

var emptyLabel = regexp.MustCompile(`^\*\*[^*]+:\*\*\s*(/\s*\*\*[^*]+:\*\*\s*)?$`)

// Filled reports whether a section has content beyond comments and bare
// "**Label:**" lines left from the template.
func (d *Doc) Filled(s Section) bool {
	for _, line := range d.Clean[s.Start:s.End] {
		t := strings.TrimSpace(line)
		if t != "" && !emptyLabel.MatchString(t) {
			return true
		}
	}
	return false
}

// FirstText returns the first non-empty line of a section, for one-line
// summaries such as a tombstone.
func (d *Doc) FirstText(s Section) string {
	for _, line := range d.Clean[s.Start:s.End] {
		if t := strings.TrimSpace(line); t != "" {
			return t
		}
	}
	return ""
}

func (d *Doc) LineCount() int { return len(d.Lines) }

func ParseDate(s string) (time.Time, bool) {
	t, err := time.ParseInLocation(DateLayout, s, time.Local)
	return t, err == nil
}

// SetFront rewrites the value of one frontmatter key in place, keeping
// any trailing comment on the line.
func SetFront(path, key, value string) error {
	d, err := Read(path)
	if err != nil {
		return err
	}
	n, ok := d.FrontLine[key]
	if !d.HasFront || !ok {
		return fmt.Errorf("%s: no %q in frontmatter", path, key)
	}
	line := d.Lines[n-1]
	_, rest, _ := strings.Cut(line, ":")
	comment := ""
	if i := strings.Index(rest, " #"); i >= 0 {
		comment = "  " + strings.TrimSpace(rest[i:])
	}
	d.Lines[n-1] = key + ": " + value + comment
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	return os.WriteFile(path, []byte(strings.Join(d.Lines, "\n")+"\n"), info.Mode().Perm())
}
