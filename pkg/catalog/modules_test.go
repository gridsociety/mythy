package catalog

import (
	"path/filepath"
	"testing"
)

func TestParseModules(t *testing.T) {
	tpl, err := ParseTemplate(filepath.Join("..", "..", "testdata", "us", "TEST-VB0-a"))
	if err != nil {
		t.Fatalf("ParseTemplate: %v", err)
	}
	if len(tpl.Modules) < 2 {
		t.Fatalf("Modules = %d, want >= 2", len(tpl.Modules))
	}
	want := map[string]string{"SchedaMadre": "EnableBoard_Madre", "SchedaMMI": "EnableBoard_MMI"}
	for _, m := range tpl.Modules {
		if v, ok := want[m.Name]; ok && m.Variabile != v {
			t.Errorf("module %s variabile = %q, want %q", m.Name, m.Variabile, v)
		}
	}
}
