package catalog

import (
	"path/filepath"
	"sort"
	"testing"
)

// TestWalk: opt into hidden groups so we can verify all 5 leaves
// (including "Identification" under the hidden Administrator group)
// are reachable via WalkData. The default-hidden behavior is covered
// by TestWalkSkipHidden below; this test exists to confirm that
// IncludeHidden=true exposes the hidden subtree.
//
// Note: Plan 1 Task 11 originally specified WalkOptions{} (zero value)
// here, but that contradicts TestWalkSkipHidden's expectation that
// hidden groups are skipped by default. Resolved by opting into
// IncludeHidden explicitly so both tests can be internally consistent.
func TestWalk(t *testing.T) {
	tpl, _ := ParseTemplate(filepath.Join("..", "..", "testdata", "us", "TEST-VB0-a"))

	all := tpl.Menu.WalkData(WalkOptions{IncludeHidden: true})
	got := make([]string, len(all))
	for i, d := range all {
		got[i] = d.Name
	}
	sort.Strings(got)
	want := []string{
		"ENABLE_SEC_MODE",
		"EnF81_TSc",
		"F25_CONT_P_Sc",
		"GST61850_CMD",
		"GST61850_CMD_PAR1",
		"GST61850_CMD_PAR2",
		"GST61850_CMD_REPLY",
		"Identification",
		"MB_address",
		"MB_baudrate",
		"NomeLinea",
		"TEST_EnableBoard",
		"TEST_IP_Address",
		"TEST_SOGLIA",
		"UL1",
	}
	if !equalSlices(got, want) {
		t.Errorf("walk(IncludeHidden) = %v, want %v", got, want)
	}
}

func TestWalkSkipHidden(t *testing.T) {
	tpl, _ := ParseTemplate(filepath.Join("..", "..", "testdata", "us", "TEST-VB0-a"))

	all := tpl.Menu.WalkData(WalkOptions{IncludeHidden: false})
	for _, d := range all {
		if d.Name == "Identification" {
			t.Errorf("Identification should be hidden by default (Administrator group has VISIBILITY=3)")
		}
	}
}

func TestWalkScope(t *testing.T) {
	tpl, _ := ParseTemplate(filepath.Join("..", "..", "testdata", "us", "TEST-VB0-a"))
	scope := tpl.Menu.FindGroup("Set/Base")
	all := scope.WalkData(WalkOptions{IncludeHidden: true})
	if len(all) != 2 {
		t.Errorf("scope walk = %d entries, want 2", len(all))
	}
}
