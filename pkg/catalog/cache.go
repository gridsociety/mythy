package catalog

import (
	"encoding/gob"
	"errors"
	"fmt"
	"os"
)

// ErrCacheStale is returned by LoadCache when the source XML is newer than
// the cache file. Callers should fall back to ParseTemplate.
var ErrCacheStale = errors.New("cache is older than source")

// SaveCache writes a parsed Template to disk as gob. The decoder needs a
// concrete type, so gob registrations live here.
func SaveCache(tpl *Template, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create cache %s: %w", path, err)
	}
	defer f.Close()
	enc := gob.NewEncoder(f)
	if err := enc.Encode(tpl); err != nil {
		return fmt.Errorf("gob encode: %w", err)
	}
	return nil
}

// LoadCache reads a cached Template from disk. Returns ErrCacheStale if the
// source XML's mtime is newer than the cache's mtime.
func LoadCache(cachePath, sourcePath string) (*Template, error) {
	cInfo, err := os.Stat(cachePath)
	if err != nil {
		return nil, err
	}
	sInfo, err := os.Stat(sourcePath)
	if err != nil {
		return nil, err
	}
	if sInfo.ModTime().After(cInfo.ModTime()) {
		return nil, ErrCacheStale
	}
	f, err := os.Open(cachePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var tpl Template
	if err := gob.NewDecoder(f).Decode(&tpl); err != nil {
		return nil, fmt.Errorf("gob decode: %w", err)
	}
	if tpl.Menu != nil {
		reparent(tpl.Menu, nil)
	}
	// Re-link DATA/COMMAND.Message — they're pointers into tpl.Messages,
	// gob serializes them by value (deep copies). Re-link by name.
	if tpl.Menu != nil {
		linkMenuToMessages(&tpl)
	}
	// Normalize TIPO aliases. Idempotent: caches written by newer
	// builds already carry resolved Tipo, and the typedef map lookup
	// is a no-op for primitives.
	tpl.resolveTypedefs()
	return &tpl, nil
}

// reparent rebuilds the Group.Parent back-pointers after gob decode.
// Group's custom GobEncode/GobDecode (in menu.go) deliberately omits
// Parent to break the Group↔Parent cycle, so we restore it here.
func reparent(g *Group, parent *Group) {
	g.Parent = parent
	for _, c := range g.Children {
		reparent(c, g)
	}
}
