package main

import (
	"path/filepath"
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

// A paragraph can be arbitrarily long without crossing a physical line budget.
// Exercise the command, including its warning exit policy and Unicode counting.
func TestSizeBudgetsBoundLongParagraphs(t *testing.T) {
	for _, tc := range []struct {
		name, file, text string
		characters       int
	}{
		{"brief", ".omut/brief.md", fill(time.Now().Format("2006-01-02"), time.Now().Format("2006-01-02"), ""), 6000},
		{"thread", ".omut/threads/resume.md", "---\nstatus: active\nupdated: " + time.Now().Format("2006-01-02") + "\nby: test/test\n---\n# Resume\n**Next step:** inspect the failing check.\n", 8000},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			standalone(t, dir, time.Now().Format("2006-01-02"))
			path := filepath.Join(dir, tc.file)
			base := strings.TrimSuffix(tc.text, "\n") + "\n"
			// Same budget for Cyrillic and ASCII: UTF-8 bytes are not characters.
			for _, ending := range []struct{ name, text string }{{"LF", "\n"}, {"CRLF", "\r\n"}} {
				t.Run(ending.name, func(t *testing.T) {
					for _, letter := range []string{"x", "я"} {
						text := base + strings.Repeat(letter, tc.characters-utf8.RuneCountInString(base))
						write(t, path, strings.ReplaceAll(text+"\n", "\n", ending.text))
						for _, f := range findings(t, dir) {
							if f.Rule == "size" {
								t.Fatalf("at the %d-character budget (%s): %+v", tc.characters, letter, f)
							}
						}
						write(t, path, strings.ReplaceAll(text+letter+"\n", "\n", ending.text))
						found := false
						for _, f := range findings(t, dir) {
							if f.Rule == "size" {
								found = true
								if f.Level != "warn" {
									t.Errorf("size should be a warning: %+v", f)
								}
								if !strings.Contains(f.Message, "characters") {
									t.Errorf("size finding does not explain the volume budget: %+v", f)
								}
							}
						}
						if !found {
							t.Fatalf("a short-line file over the character budget (%s) passed", letter)
						}
						if _, err := omut(t, dir, "check", "--fail-on", "warn"); err == nil {
							t.Error("--fail-on warn accepted oversized meta")
						}
					}
				})
			}
			// Many short lines must still hit the original line budget.
			write(t, path, tc.text+strings.Repeat("x\n", 81))
			if !strings.Contains(must(t, dir, "check"), "size") {
				t.Error("line budget stopped warning")
			}
		})
	}
}
