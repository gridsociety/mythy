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

func copyFile(t *testing.T, src, dst string) error {
	t.Helper()
	b, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, b, 0o644)
}
