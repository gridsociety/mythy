package catalog

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCacheRoundTrip(t *testing.T) {
	tmp := t.TempDir()
	src := filepath.Join(tmp, "TEST-VB0-a")
	if err := copyFile(t, filepath.Join("..", "..", "testdata", "us", "TEST-VB0-a"), src); err != nil {
		t.Fatal(err)
	}

	tpl, err := ParseTemplate(src)
	if err != nil {
		t.Fatal(err)
	}

	cachePath := filepath.Join(tmp, "TEST-VB0-a.cache")
	if err := SaveCache(tpl, cachePath); err != nil {
		t.Fatalf("SaveCache: %v", err)
	}

	got, err := LoadCache(cachePath, src)
	if err != nil {
		t.Fatalf("LoadCache: %v", err)
	}
	if got.Identification != tpl.Identification ||
		len(got.Messages) != len(tpl.Messages) ||
		got.Menu.FindGroup("Set/Base").FindData("MB_address") == nil {
		t.Error("round-trip lost data")
	}
}

func TestCacheStaleOnMtime(t *testing.T) {
	tmp := t.TempDir()
	src := filepath.Join(tmp, "TEST-VB0-a")
	if err := copyFile(t, filepath.Join("..", "..", "testdata", "us", "TEST-VB0-a"), src); err != nil {
		t.Fatal(err)
	}
	tpl, _ := ParseTemplate(src)
	cachePath := filepath.Join(tmp, "TEST-VB0-a.cache")
	_ = SaveCache(tpl, cachePath)

	// Bump the source mtime to "now + 1s" so the cache is older than source.
	future := time.Now().Add(1 * time.Second)
	if err := os.Chtimes(src, future, future); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadCache(cachePath, src); err == nil {
		t.Error("expected stale-cache error after mtime bump")
	}
}

func TestLoadCacheResolvesTypedefsForStaleCaches(t *testing.T) {
	// Pre-fix builds saved caches before typedef resolution existed,
	// so Data.Tipo on disk is still "ENUM_RELE" / "ENUM_LED" / etc.
	// LoadCache must normalize them on read so callers see the
	// resolved primitive and never the alias. Realistic scenario:
	// user upgrades mythy without touching their templates dir.
	tmp := t.TempDir()
	src := filepath.Join(tmp, "TEST-typedef")
	// Empty source file is fine — LoadCache only stats it for mtime.
	if err := os.WriteFile(src, []byte("<DEVICE/>"), 0o644); err != nil {
		t.Fatal(err)
	}

	pre := &Template{
		Identification: 42,
		Messages:       map[string]*Message{},
		ByAddr:         map[ByAddrKey]*Message{},
		Enums:          map[string]*Enum{},
		Classes:        map[string]*Class{},
		Typedefs: map[string]*Typedef{
			"ENUM_RELE": {Name: "ENUM_RELE", Tipo: "BIT32"},
		},
		Menu: &Group{
			Data: []*Data{{Name: "SelRele1", Tipo: "ENUM_RELE"}}, // unresolved
		},
	}
	cachePath := filepath.Join(tmp, "TEST-typedef.cache")
	if err := SaveCache(pre, cachePath); err != nil {
		t.Fatalf("SaveCache: %v", err)
	}

	// Make cache strictly newer than the source so it isn't stale-rejected.
	future := time.Now().Add(1 * time.Second)
	if err := os.Chtimes(cachePath, future, future); err != nil {
		t.Fatal(err)
	}

	got, err := LoadCache(cachePath, src)
	if err != nil {
		t.Fatalf("LoadCache: %v", err)
	}
	d := got.Menu.Data[0]
	if d.Tipo != "BIT32" {
		t.Errorf("Tipo = %q, want BIT32 (LoadCache should resolve)", d.Tipo)
	}
	if d.XMLTipo != "ENUM_RELE" {
		t.Errorf("XMLTipo = %q, want ENUM_RELE", d.XMLTipo)
	}
}

func copyFile(t *testing.T, src, dst string) error {
	t.Helper()
	b, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, b, 0o644)
}
