package catalog

import (
	"path/filepath"
	"testing"
)

func TestParseTemplateRoot(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "us", "TEST-VB0-a")
	tpl, err := ParseTemplate(path)
	if err != nil {
		t.Fatalf("ParseTemplate: %v", err)
	}
	if tpl.Name != "TEST_V00" {
		t.Errorf("Name = %q", tpl.Name)
	}
	if tpl.Identification != 99999 {
		t.Errorf("Identification = %d", tpl.Identification)
	}
	if tpl.Family != "TEST" {
		t.Errorf("Family = %q", tpl.Family)
	}
	if tpl.ProtocolRelease != "0100" || tpl.XMLRelease != "0101" {
		t.Errorf("Protocol/XML release = %q/%q", tpl.ProtocolRelease, tpl.XMLRelease)
	}
}
