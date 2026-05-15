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

func TestResolveTypedefsRewritesAliasToBase(t *testing.T) {
	tpl := &Template{
		Typedefs: map[string]*Typedef{
			"ENUM_RELE": {Name: "ENUM_RELE", Tipo: "BIT32"},
			"ENUM_LED":  {Name: "ENUM_LED", Tipo: "BIT32"},
			"ENUM_ING":  {Name: "ENUM_ING", Tipo: "BIT32"},
		},
		Menu: &Group{
			Data: []*Data{
				{Name: "SelRele1", Tipo: "ENUM_RELE"},
				{Name: "SelLed1", Tipo: "ENUM_LED"},
				{Name: "SelIng1", Tipo: "ENUM_ING"},
				{Name: "Threshold", Tipo: "LONG"}, // not a typedef — must stay
			},
		},
	}
	tpl.resolveTypedefs()

	for _, d := range tpl.Menu.Data {
		switch d.Name {
		case "SelRele1", "SelLed1", "SelIng1":
			if d.Tipo != "BIT32" {
				t.Errorf("%s: Tipo = %q, want BIT32", d.Name, d.Tipo)
			}
			wantXML := map[string]string{
				"SelRele1": "ENUM_RELE",
				"SelLed1":  "ENUM_LED",
				"SelIng1":  "ENUM_ING",
			}[d.Name]
			if d.XMLTipo != wantXML {
				t.Errorf("%s: XMLTipo = %q, want %q", d.Name, d.XMLTipo, wantXML)
			}
			if d.DisplayTipo() != wantXML {
				t.Errorf("%s: DisplayTipo() = %q, want %q", d.Name, d.DisplayTipo(), wantXML)
			}
		case "Threshold":
			if d.Tipo != "LONG" {
				t.Errorf("Threshold: Tipo = %q, want LONG (untouched)", d.Tipo)
			}
			if d.XMLTipo != "" {
				t.Errorf("Threshold: XMLTipo = %q, want \"\" (no resolution happened)", d.XMLTipo)
			}
			if d.DisplayTipo() != "LONG" {
				t.Errorf("Threshold: DisplayTipo() = %q, want LONG", d.DisplayTipo())
			}
		}
	}
}

func TestResolveTypedefsIsIdempotent(t *testing.T) {
	// Re-running the pass on an already-resolved template must not
	// double-resolve, lose the XMLTipo, or otherwise mutate state.
	// This matters because cache.go calls it after gob decode and
	// caches may already carry resolved Tipo values.
	tpl := &Template{
		Typedefs: map[string]*Typedef{
			"ENUM_RELE": {Name: "ENUM_RELE", Tipo: "BIT32"},
		},
		Menu: &Group{
			Data: []*Data{{Name: "SelRele1", Tipo: "ENUM_RELE"}},
		},
	}
	tpl.resolveTypedefs()
	tpl.resolveTypedefs() // second pass, should be a no-op

	d := tpl.Menu.Data[0]
	if d.Tipo != "BIT32" {
		t.Errorf("Tipo = %q, want BIT32 after 2 passes", d.Tipo)
	}
	if d.XMLTipo != "ENUM_RELE" {
		t.Errorf("XMLTipo = %q, want ENUM_RELE after 2 passes", d.XMLTipo)
	}
}

func TestResolveTypedefsRecursesIntoNestedGroups(t *testing.T) {
	// Typedef resolution must reach DATA inside nested <GROUP> trees,
	// not just the menu root. Mirrors how real templates organise
	// settings under deep paths like Set/F81/RS1.
	tpl := &Template{
		Typedefs: map[string]*Typedef{
			"ENUM_LED": {Name: "ENUM_LED", Tipo: "BIT32"},
		},
		Menu: &Group{
			Children: []*Group{
				{Children: []*Group{
					{Data: []*Data{{Name: "DeepLed", Tipo: "ENUM_LED"}}},
				}},
			},
		},
	}
	tpl.resolveTypedefs()
	got := tpl.Menu.Children[0].Children[0].Data[0]
	if got.Tipo != "BIT32" || got.XMLTipo != "ENUM_LED" {
		t.Errorf("nested resolution failed: Tipo=%q XMLTipo=%q", got.Tipo, got.XMLTipo)
	}
}
