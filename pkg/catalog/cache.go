package catalog

import (
	"encoding/gob"
	"errors"
	"fmt"
	"os"
)

// ErrCacheStale is returned by LoadCache when the on-disk cache cannot
// be safely loaded into the running build. Three independent triggers:
//
//   - the source XML's mtime is newer than the cache file's mtime
//     (the catalog changed since the cache was written);
//   - the cache's magic header doesn't match cacheFileMagic (a
//     pre-versioning legacy cache file or a corrupted one);
//   - the cache's parser-schema version doesn't equal
//     parserSchemaVersion (the in-memory schema changed since the
//     cache was written, even if the source XML didn't).
//
// In every case the right answer is the same: drop the cache and let
// the caller fall back to ParseTemplate, which writes a fresh cache.
var ErrCacheStale = errors.New("cache is stale or incompatible with this build")

// parserSchemaVersion bumps any time the in-memory shape that gets gob-
// encoded (Template / Group / Data / Class / CompoundFieldOverride /
// related catalog structs) gains, drops, or rewrites a field, OR any
// time a load-time pass — like resolveTypedefs — changes how the same
// XML decodes into those structs. Bumping invalidates every pre-
// existing .cache sibling on disk on the next load, forcing a clean
// reparse so users running a new mythy build don't transparently
// consume an old cache produced by the old parser.
//
// Bumping policy: when a PR touches any file under pkg/catalog/ that
// affects parse output, bump this. Reviewers should treat
// pkg/catalog/ diffs without a version bump as suspicious. Version 1
// is the first build that emits the versioned format; pre-version
// caches are detected via the magic header and rejected as stale.
const parserSchemaVersion = 1

// cacheFileMagic is a sentinel written at the start of the gob stream.
// Pre-versioning caches encoded a bare *Template directly, so any
// decode attempt with the new wrapper fails fast on the magic field
// and routes to ErrCacheStale. Don't change this string — bump
// parserSchemaVersion instead.
const cacheFileMagic = "mythy-catalog-cache"

// cacheFile is the on-disk wrapper around the parsed Template. The
// gob encoding is "the entire cacheFile struct", so adding fields
// here automatically reaches every existing cache via the version
// check.
type cacheFile struct {
	Magic         string
	SchemaVersion int
	Template      *Template
}

// SaveCache writes a parsed Template to disk as gob. The decoder needs a
// concrete type, so gob registrations live here.
func SaveCache(tpl *Template, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create cache %s: %w", path, err)
	}
	defer f.Close()
	enc := gob.NewEncoder(f)
	if err := enc.Encode(cacheFile{
		Magic:         cacheFileMagic,
		SchemaVersion: parserSchemaVersion,
		Template:      tpl,
	}); err != nil {
		return fmt.Errorf("gob encode: %w", err)
	}
	return nil
}

// LoadCache reads a cached Template from disk. Returns ErrCacheStale if
// the source XML is newer, if the cache lacks the versioned header
// (legacy format), or if the version doesn't match the current
// parserSchemaVersion.
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
	var wrapped cacheFile
	if err := gob.NewDecoder(f).Decode(&wrapped); err != nil {
		// Most commonly hit when reading a pre-versioning cache file
		// that gob-decodes into the wrapper as a malformed struct.
		// Treat any decode failure as stale so the caller reparses.
		return nil, ErrCacheStale
	}
	if wrapped.Magic != cacheFileMagic || wrapped.SchemaVersion != parserSchemaVersion {
		return nil, ErrCacheStale
	}
	tpl := wrapped.Template
	if tpl == nil {
		return nil, ErrCacheStale
	}
	if tpl.Menu != nil {
		reparent(tpl.Menu, nil)
	}
	// Re-link DATA/COMMAND.Message — they're pointers into tpl.Messages,
	// gob serializes them by value (deep copies). Re-link by name.
	if tpl.Menu != nil {
		linkMenuToMessages(tpl)
	}
	// Normalize TIPO aliases. Idempotent: caches written by newer
	// builds already carry resolved Tipo, and the typedef map lookup
	// is a no-op for primitives.
	tpl.resolveTypedefs()
	return tpl, nil
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
