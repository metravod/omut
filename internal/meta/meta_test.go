package meta

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

const brief = `---
status: active   # active | frozen
updated: 2026-10-31
verified:
by: codex/gpt-5
ignore: [old, scratch]
---
## Why
<!-- template hint -->

## Now
**Verified:**
**Not verified:**

## Next
- one
`

func TestFrontmatter(t *testing.T) {
	d := Parse("brief.md", brief)
	if !d.HasFront {
		t.Fatal("frontmatter not found")
	}
	if got := d.Front["status"]; got != "active" {
		t.Errorf("status = %q, trailing comment must be dropped", got)
	}
	if v, ok := d.Front["verified"]; !ok || v != "" {
		t.Errorf("verified = %q, %v; want present and empty", v, ok)
	}
	if !slices.Equal(d.Lists["ignore"], []string{"old", "scratch"}) {
		t.Errorf("ignore = %v", d.Lists["ignore"])
	}
	if d.FrontLine["by"] != 5 {
		t.Errorf("by line = %d", d.FrontLine["by"])
	}
}

func TestFilledIgnoresCommentsAndBareLabels(t *testing.T) {
	d := Parse("brief.md", brief)
	for title, want := range map[string]bool{"Why": false, "Now": false, "Next": true} {
		s, ok := d.Section(title)
		if !ok {
			t.Fatalf("no section %s", title)
		}
		if got := d.Filled(s); got != want {
			t.Errorf("Filled(%s) = %v, want %v", title, got, want)
		}
	}
}

func TestDecisions(t *testing.T) {
	d := Parse("decisions.md", `# Decisions

- D1 [accepted 2026-06-02] SQLite
- D2 [accepted ≤2026-10-22] carried over
- D3 [open] Keep collecting?
- D4 [doubted 2026-10-31] Keep raw HTML
- D5 [superseded → D2] Deploy with cron
- D6 [maybe] not a status
- D7 [open 2026-01-01] open may say since when
- Dx broken
<!-- - D9 [nonsense] does not count in a comment -->
`)
	decs, bad := d.Decisions()
	if len(decs) != 6 {
		t.Fatalf("parsed %d decisions: %+v", len(decs), decs)
	}
	if decs[4].Status != "superseded" || decs[4].Supersedes != 2 {
		t.Errorf("superseded parsed as %+v", decs[4])
	}
	if !slices.Equal(bad, []int{8, 10}) {
		t.Errorf("bad lines = %v", bad)
	}
}

func TestLogEntries(t *testing.T) {
	d := Parse("log.md", "<!-- hint -->\n- 2026-10-31 · claude-code/opus-5.5 · did it → threads/x.md\n- 2026-10-30 broken\n")
	entries, bad := d.LogEntries()
	if len(entries) != 1 || entries[0].Author != "claude-code/opus-5.5" {
		t.Errorf("entries = %+v", entries)
	}
	if !slices.Equal(bad, []int{3}) {
		t.Errorf("bad = %v", bad)
	}
}

func TestRefs(t *testing.T) {
	d := Parse("x.md", "- D1 [accepted] a → docs/adr/0001.md, threads/x.md#step\n"+
		"- D2 [open] «question → checked plan» → AGENTS.md «Main invariant»\n"+
		"see [spec](docs/spec.md) and [site](https://example.com/a.md)\n"+
		"example: `→ ../other/.omut/decisions.md#Dn` is not a link\n"+
		"<!-- → hidden.md -->\n")
	var got []string
	for _, r := range d.Refs() {
		got = append(got, r.Target)
	}
	want := []string{"docs/adr/0001.md", "threads/x.md", "AGENTS.md", "docs/spec.md"}
	if !slices.Equal(got, want) {
		t.Errorf("refs = %v, want %v", got, want)
	}
}

func TestMentionsWholeWord(t *testing.T) {
	d := Parse("x.md", "site is dead\nwebsite is not\nsite-builder neither\nsee (site)\n")
	if got := d.Mentions("site"); !slices.Equal(got, []int{1, 4}) {
		t.Errorf("mentions = %v", got)
	}
}

func TestSetFrontKeepsComment(t *testing.T) {
	path := filepath.Join(t.TempDir(), "brief.md")
	if err := os.WriteFile(path, []byte(brief), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := SetFront(path, "status", "frozen"); err != nil {
		t.Fatal(err)
	}
	d, _ := Read(path)
	if d.Front["status"] != "frozen" || d.Lines[1] != "status: frozen  # active | frozen" {
		t.Errorf("line = %q", d.Lines[1])
	}
	if err := SetFront(path, "missing", "x"); err == nil {
		t.Error("SetFront on a missing key must fail")
	}
}

func TestClaims(t *testing.T) {
	d := Parse("brief.md", "## Where it lives\n- PyPI: zymi-core 0.9.3 · how: `pip index versions zymi-core` · seen 2026-11-03\n- a VPS · seen 2026-10-01\n  prose under it\n- GitHub · public · how: gh repo view\n## Now\n- not a claim · how: x\n")
	got := d.Claims("Where it lives")
	want := []Claim{
		{2, "PyPI: zymi-core 0.9.3", "pip index versions zymi-core", "2026-11-03"},
		{3, "a VPS", "", "2026-10-01"},
		{5, "GitHub · public", "gh repo view", ""},
	}
	if len(got) != len(want) {
		t.Fatalf("claims: %+v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("claim %d: %+v, want %+v", i, got[i], want[i])
		}
	}
}
