package catalog

import (
	"path/filepath"
	"testing"
)

func TestLoadByIdentification(t *testing.T) {
	root := filepath.Join("..", "..", "testdata")
	tpl, entry, err := Load(LoadOptions{
		Root:           root,
		Locale:         "en",
		Identification: 99999,
	})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if entry.Product != "TEST-VX0-a" {
		t.Errorf("entry.Product = %q", entry.Product)
	}
	if tpl.Identification != 99999 {
		t.Errorf("tpl.Identification = %d", tpl.Identification)
	}
}

func TestLoadByProduct(t *testing.T) {
	root := filepath.Join("..", "..", "testdata")
	tpl, entry, err := Load(LoadOptions{
		Root:    root,
		Locale:  "en",
		Product: "TEST-VX0-a",
	})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if entry.Identification != 99999 || tpl.Family != "TEST" {
		t.Errorf("got %+v / %s", entry, tpl.Family)
	}
}

func TestLoadRequiresOneOf(t *testing.T) {
	root := filepath.Join("..", "..", "testdata")
	if _, _, err := Load(LoadOptions{Root: root, Locale: "en"}); err == nil {
		t.Error("expected error when neither Identification nor Product is set")
	}
}
