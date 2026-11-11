package dream

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/metravod/omut/internal/check"
	"github.com/metravod/omut/internal/pack"
)

// writeProject lays down a scope with the given brief frontmatter and dated
// records, like a project or a pack root that has been worked on.
func writeProject(t *testing.T, front string, log, archive, decisions, graveyard string, threads map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	lead := filepath.Join(dir, pack.MetaDir)
	if err := os.MkdirAll(lead, 0o755); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(lead, pack.Brief), "---\n"+front+"---\n## Why\nworks\n")
	if log != "" {
		write(t, filepath.Join(lead, pack.Log), log)
	}
	if archive != "" {
		if err := os.MkdirAll(filepath.Join(lead, "log-archive"), 0o755); err != nil {
			t.Fatal(err)
		}
		write(t, filepath.Join(lead, "log-archive", "2026.md"), archive)
	}
	if decisions != "" {
		write(t, filepath.Join(lead, pack.Decisions), decisions)
	}
	if graveyard != "" {
		write(t, filepath.Join(lead, pack.Graveyard), graveyard)
	}
	for name, text := range threads {
		if err := os.MkdirAll(filepath.Join(lead, pack.Threads), 0o755); err != nil {
			t.Fatal(err)
		}
		write(t, filepath.Join(lead, pack.Threads, name), text)
	}
	return dir
}

func write(t *testing.T, path, text string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestCountCountsOnlyRecordsAfterDreamed(t *testing.T) {
	dir := writeProject(t,
		"status: active\ndreamed: 2026-11-08\n",
		"- 2026-11-10 · t/t · newer work\n- 2026-11-07 · t/t · older work\n",
		"- 2026-11-09 · t/t · archived work\n- 2026-11-01 · t/t · old archived\n",
		"# Decisions\n\n- D1 [accepted 2026-11-09] newer decision\n- D2 [accepted 2026-11-02] older decision\n",
		"- gone — removed 2026-11-10 · reason · what it was\n- earlier — removed 2026-11-05 · reason · what\n",
		map[string]string{
			"new.md": "---\nstatus: active\nupdated: 2026-11-09\nby: t/t\n---\n# New thread\n",
			"old.md": "---\nstatus: parked\nupdated: 2026-11-05\nby: t/t\nparked_because: doubt\n---\n# Old thread\n",
		})
	c, problems := CountDir(dir)
	if len(problems) != 0 {
		t.Fatalf("problems: %+v", problems)
	}
	if c.Dreamed != "2026-11-08" {
		t.Errorf("dreamed %q", c.Dreamed)
	}
	if c.Total() != 5 {
		t.Fatalf("counted %d records, want 5: %+v", c.Total(), c.Records)
	}
	want := map[string]bool{
		"newer work":                                       true, // log: the entry's text
		"- 2026-11-09 · t/t · archived work":               true, // archive: the line as it is
		"- D1 [accepted 2026-11-09] newer decision":        true,
		"New thread":                                       true,
		"- gone — removed 2026-11-10 · reason · what it was": true,
	}
	for _, rec := range c.Records {
		if !want[rec.Text] {
			t.Errorf("unexpected record: %+v", rec)
		}
		delete(want, rec.Text)
	}
	for text := range want {
		t.Errorf("missing record %q", text)
	}
	// Newest first, regardless of the source file.
	if c.Records[0].Date != "2026-11-10" {
		t.Errorf("first record: %+v", c.Records[0])
	}
}

func TestCountThresholdAndDue(t *testing.T) {
	log := ""
	for i := 0; i < 9; i++ {
		log += "- 2026-11-10 · t/t · work\n"
	}
	// Nine records, default threshold 10: below.
	dir := writeProject(t, "status: active\n", log, "", "", "", nil)
	c, _ := CountDir(dir)
	if c.Total() != 9 || c.Threshold != 10 || c.Due() {
		t.Errorf("default threshold: %d/%d due=%t", c.Total(), c.Threshold, c.Due())
	}
	// Tenth record: due.
	write(t, filepath.Join(dir, pack.MetaDir, pack.Log), log+"- 2026-11-10 · t/t · more\n")
	c, _ = CountDir(dir)
	if !c.Due() {
		t.Errorf("10 records: not due (%d/%d)", c.Total(), c.Threshold)
	}
	// The project's own threshold, and an invalid one falls back.
	dir = writeProject(t, "status: active\ndream: 3\n", log, "", "", "", nil)
	if c, _ = CountDir(dir); !c.Due() || c.Threshold != 3 {
		t.Errorf("dream: 3: %d/%d due=%t", c.Total(), c.Threshold, c.Due())
	}
	dir = writeProject(t, "status: active\ndream: many\n", log, "", "", "", nil)
	if c, _ = CountDir(dir); c.Threshold != DefaultThreshold {
		t.Errorf("invalid dream: threshold %d", c.Threshold)
	}
	// A never-dreamed small memory stays silent.
	dir = writeProject(t, "status: active\n", "- 2026-11-10 · t/t · one\n", "", "", "", nil)
	if c, _ = CountDir(dir); c.Due() || c.Dreamed != "" {
		t.Errorf("small undreamed memory: %+v", c)
	}
}

func TestForProjectReportAndContract(t *testing.T) {
	brief := "status: active\nupdated: 2026-11-01\nverified: 2026-11-01\nby: t/t\n---\n" +
		"## Why\nworks\n## What it is\nworks\n## Where it lives\n\n## Now\n**Verified:** a run\n## Next\n- one\n"
	dir := writeProject(t, brief,
		"- 2026-11-10 · t/t · work\n", "", "# Decisions\n\n", "",
		map[string]string{"w.md": "---\nstatus: active\nupdated: 2026-11-09\nby: t/t\n---\n# W\n**Next step:** x\n"})
	pl := pack.Place{Dir: dir, Instance: dir}
	r := ForProject(pl, check.Options{}, false)
	if r.Scope != "project" || r.Mode != "report" || r.Total() != 2 {
		t.Fatalf("report: %+v", r)
	}
	if r.Incomplete {
		t.Errorf("a healthy project is complete: %+v", r.Problems)
	}
	md := r.Markdown()
	for _, want := range []string{
		"# omut dream:", "no dream is recorded yet", "## Records since the last dream",
		"## What check found", "## The dream contract", "Change\nnothing until they approve",
	} {
		if !strings.Contains(md, want) {
			t.Errorf("markdown has no %q:\n%s", want, md)
		}
	}
	apply := ForProject(pl, check.Options{}, true)
	if apply.Mode != "apply" || !strings.Contains(apply.Markdown(), "the human has approved") {
		t.Errorf("apply mode: %s", apply.Markdown())
	}
	if strings.Contains(apply.Markdown(), "Change\nnothing until they approve") {
		t.Error("apply contract still says wait for approval")
	}
	if len(apply.Records) != len(r.Records) {
		t.Error("apply and report count differently")
	}
}

func TestForPackCountsGraveyardAndRoutes(t *testing.T) {
	root := writeProject(t, "status: active\n", "", "",
		"# Decisions\n\n- D1 [open 2026-11-10] waiting\n", "- gone — removed 2026-11-09 · reason · what\n", nil)
	// A pack root is one that holds LEAD.md.
	write(t, filepath.Join(root, pack.MetaDir, pack.LeadFile), "# lead\n")
	r := ForPack(&pack.Pack{Root: root}, check.Options{}, false)
	if r.Scope != "pack" {
		t.Errorf("scope %q", r.Scope)
	}
	kinds := map[string]int{}
	for _, rec := range r.Records {
		kinds[rec.Kind]++
	}
	if kinds["decision"] != 1 || kinds["graveyard"] != 1 {
		t.Errorf("pack records: %+v", r.Records)
	}
	if !strings.Contains(r.Contract, "dream <project>") {
		t.Errorf("pack contract doesn't route to projects: %s", r.Contract)
	}
}

func TestCountReportsReadFailures(t *testing.T) {
	dir := writeProject(t, "status: active\n", "", "", "", "", nil)
	// A thread entry that cannot be read: a dangling symlink.
	threads := filepath.Join(dir, pack.MetaDir, pack.Threads)
	if err := os.MkdirAll(threads, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(dir, "gone"), filepath.Join(threads, "d.md")); err != nil {
		t.Fatal(err)
	}
	_, problems := CountDir(dir)
	if len(problems) == 0 {
		t.Fatal("an unreadable thread is silent")
	}
	r := ForProject(pack.Place{Dir: dir, Instance: dir}, check.Options{}, false)
	if !r.Incomplete {
		t.Errorf("incomplete not set: %+v", r.Problems)
	}
	if !strings.Contains(r.Markdown(), "## Incomplete") {
		t.Error("markdown says nothing about incompleteness")
	}
}
