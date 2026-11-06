package context

import (
	"strings"
	"testing"
)

func TestPorcelainV2SubmoduleAndIncompleteRecords(t *testing.T) {
	head := "# branch.oid abc123\x00# branch.head main\x00"
	g := &GitState{Changes: []GitChange{}}
	data := head + "# future.header ignored\x00" + "1 .M S.MU 160000 160000 160000 abc123 abc123 nested submodule\x00"
	if err := g.parse([]byte(data)); err != nil || len(g.Changes) != 1 || g.Changes[0].Submodule != "S.MU" || g.Changes[0].Path != "nested submodule" {
		t.Fatalf("submodule state or future header: %+v, %v", g, err)
	}
	for _, text := range []string{"", strings.TrimSuffix(head, "\x00"), "? path\x00", head + "1 .M bad\x00", head + "2 R. N... 100644 100644 100644 abc abc R100 target\x00", head + "# branch.ab +? -0\x00", head + "x unknown\x00"} {
		if err := (&GitState{}).parse([]byte(text)); err == nil {
			t.Errorf("accepted incomplete or invalid status %q", text)
		}
	}
}
