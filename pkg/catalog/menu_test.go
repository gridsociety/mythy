package catalog

import (
	"path/filepath"
	"testing"
)

func TestParseMenu(t *testing.T) {
	tpl, err := ParseTemplate(filepath.Join("..", "..", "testdata", "us", "TEST-VB0-a"))
	if err != nil {
		t.Fatalf("ParseTemplate: %v", err)
	}
	root := tpl.Menu
	if root == nil {
		t.Fatal("Menu is nil")
	}

	wantTopLevel := []string{"Read", "Set", "Communication", "Administrator", "Commands"}
	got := make([]string, len(root.Children))
	for i, c := range root.Children {
		got[i] = c.Name
	}
	if !equalSlices(got, wantTopLevel) {
		t.Fatalf("top-level groups = %v, want %v", got, wantTopLevel)
	}

	// Drill down to Set/Base/MB_address
	base := root.FindGroup("Set/Base")
	if base == nil {
		t.Fatal("Set/Base not found")
	}
	if len(base.Data) != 2 {
		t.Errorf("Set/Base has %d data entries, want 2", len(base.Data))
	}
	mb := base.FindData("MB_address")
	if mb == nil {
		t.Fatal("MB_address not found in Set/Base")
	}
	if mb.Tipo != "UBYTE" || mb.Valore != "1" || mb.Default != "1" {
		t.Errorf("MB_address attrs = %+v", mb)
	}

	// Hidden group
	admin := root.FindGroup("Administrator")
	if admin == nil || admin.Visibility != "3" {
		t.Errorf("Administrator group: visibility = %q", admin.Visibility)
	}

	// Command
	cmds := root.FindGroup("Commands")
	if cmds == nil || len(cmds.Commands) != 2 ||
		cmds.Commands[0].Name != "MSG_CMD_RESET_DA_PC" ||
		cmds.Commands[1].Name != "SET_RTC" {
		t.Errorf("Commands group: %+v", cmds)
	}
}

func TestParseCompoundOverridesFromNestedDATA(t *testing.T) {
	// Issue #6: <DATA TIPO="<class>"> can carry nested <DATA NAME="..."
	// TIPO="..."> children that override the CLASS VAR's TIPO for that
	// instance. Without parsing those, the wire layout falls back to the
	// (under-filled) CLASS-level widths and writes corrupt the device.
	tpl, err := ParseTemplate(filepath.Join("..", "..", "testdata", "us", "TEST-VB0-a"))
	if err != nil {
		t.Fatalf("ParseTemplate: %v", err)
	}
	soglie := tpl.Menu.FindGroup("Set/Soglie")
	if soglie == nil {
		t.Fatal("Set/Soglie group missing from fixture")
	}
	d := soglie.FindData("TEST_SOGLIA")
	if d == nil {
		t.Fatal("TEST_SOGLIA DATA missing")
	}
	if d.CompoundOverrides == nil {
		t.Fatal("CompoundOverrides nil — nested DATA children were not captured")
	}
	state, ok := d.CompoundOverrides["State"]
	if !ok || state.Tipo != "ENUM_LONG" {
		t.Errorf("State override = %+v, want Tipo=ENUM_LONG", state)
	}
	pickup, ok := d.CompoundOverrides["Pickup"]
	if !ok || pickup.Tipo != "UWORD" {
		t.Errorf("Pickup override = %+v, want Tipo=UWORD", pickup)
	}
	// Other DATA in the same fixture must NOT have overrides (we want to
	// confirm we didn't accidentally start collecting children for
	// non-compound DATA).
	mb := tpl.Menu.FindGroup("Set/Base").FindData("MB_address")
	if mb.CompoundOverrides != nil {
		t.Errorf("MB_address (non-compound) accidentally has CompoundOverrides: %+v", mb.CompoundOverrides)
	}
}

func equalSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
