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

// Diff reads each settings key from the live device and compares it
// to the file value. Only keys present in the file are checked.
// Order is by menu path (depth-first, alphabetical fallback).
func Diff(ctx context.Context, s *session.Session, cf *ConfigFile) ([]Change, error) {
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
	for _, it := range items {
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
	return changes, nil
}

// valuesEqual is reflect.DeepEqual with int64/int/uint64 normalisation
// so that a YAML-loaded `5` (int) compares equal to a session.ValueToYAML
// result (`int64(5)`).
func valuesEqual(a, b any) bool {
	a = normaliseInt(a)
	b = normaliseInt(b)
	return reflect.DeepEqual(a, b)
}

func normaliseInt(v any) any {
	switch x := v.(type) {
	case int:
		return int64(x)
	case uint64:
		return int64(x)
	}
	return v
}

// walkAll returns WalkOptions that include hidden + read-only entries.
// Used wherever Diff/Apply needs the unfiltered tree.
func walkAll() catalog.WalkOptions {
	return catalog.WalkOptions{IncludeHidden: true, IncludeReadOnly: true}
}
