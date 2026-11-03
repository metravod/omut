package meta

import "testing"

func TestParseADR(t *testing.T) {
	for _, c := range []struct {
		name, text, status, waiting string
	}{
		{"nygard line", "# Use Rust\n\nDate: 2026-04-02\n\nStatus: Proposed.\n", "proposed", "open"},
		{"nygard section", "# Use Rust\n\n## Status\n\nAccepted\n\n## Context\n", "accepted", ""},
		{"madr frontmatter", "---\nstatus: draft\n---\n# Use Rust\n", "draft", "open"},
		{"madr bullet", "# Use Rust\n\n* Status: accepted\n* Deciders: x\n", "accepted", ""},
		{"bold", "# Use Rust\n\n**Status:** Proposed\n", "proposed", "open"},
		{"stare-decisis tentative", "# Use Rust\n\nDate: 2026-04-02\nFirmness: tentative\nRevisit if: it is slow\n", "", "doubted"},
		{"tentative but superseded", "# Use Rust\n\nFirmness: tentative\nStatus: Superseded by 0007\n", "superseded", ""},
		{"no status", "# Use Rust\n\nDate: 2026-04-02\n\n## Context\nThe status of the code is bad.\n", "", ""},
		{"long status", "# Use Rust\n\nStatus: Accepted. Amended by 0044 (venv)\n", "accepted", ""},
	} {
		a := ParseADR(Parse("0001.md", c.text))
		if a.Title != "Use Rust" || a.Status != c.status || a.Waiting() != c.waiting {
			t.Errorf("%s: %+v, waiting %q", c.name, a, a.Waiting())
		}
	}
}
