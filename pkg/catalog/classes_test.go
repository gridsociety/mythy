package catalog

import (
	"path/filepath"
	"testing"
)

func TestParseClasses(t *testing.T) {
	tpl, err := ParseTemplate(filepath.Join("..", "..", "testdata", "us", "TEST-VB0-a"))
	if err != nil {
		t.Fatalf("ParseTemplate: %v", err)
	}
	c, ok := tpl.Classes["CONTATORE"]
	if !ok {
		t.Fatal("CONTATORE class missing")
	}
	if c.Dim != 3 {
		t.Errorf("CONTATORE Dim = %d, want 3", c.Dim)
	}
	if len(c.Vars) < 2 {
		t.Fatalf("CONTATORE Vars = %d, want >= 2", len(c.Vars))
	}
	if c.Vars[0].Name != "Tipo" || c.Vars[0].Tipo != "ENUM_LONG" {
		t.Errorf("CONTATORE Vars[0] = %+v", c.Vars[0])
	}

	timer, ok := tpl.Classes["TIMER"]
	if !ok || timer.Dim != 4 {
		t.Errorf("TIMER class missing or wrong DIM: %+v", timer)
	}
}
