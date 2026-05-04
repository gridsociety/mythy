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
	if cmds == nil || len(cmds.Commands) != 1 || cmds.Commands[0].Name != "MSG_CMD_RESET_DA_PC" {
		t.Errorf("Commands group: %+v", cmds)
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
