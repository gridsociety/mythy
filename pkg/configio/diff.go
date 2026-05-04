package configio

import (
	"context"
	"reflect"
	"sort"

	"github.com/gridsociety/mythy/pkg/catalog"
	"github.com/gridsociety/mythy/pkg/session"
)

// Change is one row of the diff: a key present in the file whose
// value differs from the live device.
type Change struct {
	Name    string
	Path    string // menu path, for sorting + display
	Current any    // value read from the device
	File    any    // value from the YAML
}

// DiffOptions controls Diff behaviour. The zero value is fine; the
// only knob today is Progress, which lets the caller render a
// progress indicator across the per-key Modbus reads (Diff issues
// one read per file entry).
type DiffOptions struct {
	// Progress, when non-nil, is invoked once before each Modbus
	// read and once more at the end with done == total and name == ""
	// to signal completion. Mirrors ExportOptions.Progress.
	Progress func(done, total int, name string)
}

// Diff reads each settings key from the live device and compares it
// to the file value. Only keys present in the file are checked.
// Order is by menu path (depth-first, alphabetical fallback).
//
// Diff is the read-bound phase of import: a full-device YAML
// (~7,400 keys on PROX-VX0-e) takes the same ~4 minutes a full
// export does, since both issue one Modbus read per entry. The
// follow-up write phase in Apply only touches the keys that
// actually differ.
func Diff(ctx context.Context, s *session.Session, cf *ConfigFile, opts DiffOptions) ([]Change, error) {
	tpl := s.Template()
	pathByName := make(map[string]string)
	for _, g := range tpl.Menu.WalkGroups(walkAll()) {
		for _, d := range g.Data {
			pathByName[d.Name] = g.Path()
		}
	}

	type item struct {
		name, path string
		fileVal    any
	}
	items := make([]item, 0, len(cf.Settings))
	for k, v := range cf.Settings {
		items = append(items, item{name: k, path: pathByName[k], fileVal: v})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].path != items[j].path {
			return items[i].path < items[j].path
		}
		return items[i].name < items[j].name
	})

	var changes []Change
	total := len(items)
	for i, it := range items {
		if opts.Progress != nil {
			opts.Progress(i, total, it.name)
		}
		v, err := s.Read(ctx, it.name)
		if err != nil {
			return nil, err
		}
		current := ValueToYAML(v)
		if !valuesEqual(current, it.fileVal) {
			changes = append(changes, Change{
				Name:    it.name,
				Path:    it.path,
				Current: current,
				File:    it.fileVal,
			})
		}
	}
	if opts.Progress != nil {
		opts.Progress(total, total, "")
	}
	return changes, nil
}

// valuesEqual is reflect.DeepEqual with int64/int/uint64 normalisation
// so that a YAML-loaded `5` (int) compares equal to a session.ValueToYAML
// result (`int64(5)`). Recurses into nested maps so compound values
// (TIMER, CONTATORE, …) round-trip cleanly.
func valuesEqual(a, b any) bool {
	return reflect.DeepEqual(normaliseValue(a), normaliseValue(b))
}

func normaliseValue(v any) any {
	switch x := v.(type) {
	case int:
		return int64(x)
	case uint64:
		return int64(x)
	case map[string]any:
		out := make(map[string]any, len(x))
		for k, sub := range x {
			out[k] = normaliseValue(sub)
		}
		return out
	}
	return v
}

// walkAll returns WalkOptions that include hidden + read-only entries.
// Used wherever Diff/Apply needs the unfiltered tree.
func walkAll() catalog.WalkOptions {
	return catalog.WalkOptions{IncludeHidden: true, IncludeReadOnly: true}
}
