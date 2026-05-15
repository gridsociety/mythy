package catalog

import (
	"encoding/gob"
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

func TestLoadCacheRejectsLegacyUnwrappedFormat(t *testing.T) {
	// Before issue #5 was fixed, SaveCache wrote a bare gob-encoded
	// *Template (no magic, no version). After the fix, LoadCache must
	// detect this and route to ErrCacheStale so the caller reparses
	// the XML — otherwise a user upgrading mythy without wiping their
	// Templates/ would silently keep running on the old parser's output.
	tmp := t.TempDir()
	src := filepath.Join(tmp, "TEST")
	if err := os.WriteFile(src, []byte("<DEVICE/>"), 0o644); err != nil {
		t.Fatal(err)
	}
	cachePath := filepath.Join(tmp, "TEST.cache")
	f, err := os.Create(cachePath)
	if err != nil {
		t.Fatal(err)
	}
	// Write a legacy-format cache: bare *Template, no wrapper.
	if err := gob.NewEncoder(f).Encode(&Template{Identification: 7}); err != nil {
		t.Fatal(err)
	}
	f.Close()

	// Ensure the cache mtime is newer than the source, so the only
	// remaining stale-trigger is the missing header.
	future := time.Now().Add(1 * time.Second)
	if err := os.Chtimes(cachePath, future, future); err != nil {
		t.Fatal(err)
	}

	if _, err := LoadCache(cachePath, src); err != ErrCacheStale {
		t.Errorf("LoadCache on legacy format: err = %v, want ErrCacheStale", err)
	}
}

func TestLoadCacheRejectsWrongSchemaVersion(t *testing.T) {
	// When the parser changes the on-disk shape, parserSchemaVersion
	// is bumped. A cache written by an older build must be rejected
	// so the caller reparses with the current parser.
	tmp := t.TempDir()
	src := filepath.Join(tmp, "TEST")
	if err := os.WriteFile(src, []byte("<DEVICE/>"), 0o644); err != nil {
		t.Fatal(err)
	}
	cachePath := filepath.Join(tmp, "TEST.cache")
	f, err := os.Create(cachePath)
	if err != nil {
		t.Fatal(err)
	}
	// Write a cacheFile with a deliberately-wrong schema version.
	if err := gob.NewEncoder(f).Encode(cacheFile{
		Magic:         cacheFileMagic,
		SchemaVersion: parserSchemaVersion + 1, // pretend we're an older build
		Template:      &Template{Identification: 7},
	}); err != nil {
		t.Fatal(err)
	}
	f.Close()

	future := time.Now().Add(1 * time.Second)
	if err := os.Chtimes(cachePath, future, future); err != nil {
		t.Fatal(err)
	}

	if _, err := LoadCache(cachePath, src); err != ErrCacheStale {
		t.Errorf("LoadCache on wrong-version cache: err = %v, want ErrCacheStale", err)
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
