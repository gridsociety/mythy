package configio

import (
	"context"
	"fmt"

	"github.com/gridsociety/mythy/pkg/catalog"
	"github.com/gridsociety/mythy/pkg/session"
)

// Report describes the outcome of an Apply or ApplyDryRun call.
type Report struct {
	Applied    []string // names actually written by Apply
	WouldApply []string // names that ApplyDryRun would have written
}

// Apply computes the diff, then writes only the changed keys inside
// one edit transaction (or via SetMany's auto-bundling).
//
// The product-mismatch check lives in Validate; the caller is expected
// to have run it already (mythy import does).
func Apply(ctx context.Context, s *session.Session, cf *ConfigFile) (*Report, error) {
	changes, err := Diff(ctx, s, cf)
	if err != nil {
		return nil, err
	}
	if len(changes) == 0 {
		return &Report{}, nil
	}

	tpl := s.Template()
	pairs := make(map[string]any, len(changes))
	applied := make([]string, 0, len(changes))
	for _, c := range changes {
		// Look up TIPO/ENUM to drive the codec selector.
		d, err := lookupData(tpl, c.Name)
		if err != nil {
			return nil, err
		}
		typed, err := YAMLToCodec(d.Tipo, d.Enum, c.File)
		if err != nil {
			return nil, fmt.Errorf("encode %s: %w", c.Name, err)
		}
		pairs[c.Name] = typed
		applied = append(applied, c.Name)
	}
	if err := s.SetMany(ctx, pairs); err != nil {
		return nil, err
	}
	return &Report{Applied: applied}, nil
}

// ApplyDryRun returns the list of keys that Apply would write, without
// performing any writes.
func ApplyDryRun(ctx context.Context, s *session.Session, cf *ConfigFile) (*Report, error) {
	changes, err := Diff(ctx, s, cf)
	if err != nil {
		return nil, err
	}
	out := make([]string, len(changes))
	for i, c := range changes {
		out[i] = c.Name
	}
	return &Report{WouldApply: out}, nil
}

func lookupData(tpl *catalog.Template, name string) (*catalog.Data, error) {
	for _, g := range tpl.Menu.WalkGroups(catalog.WalkOptions{IncludeHidden: true, IncludeReadOnly: true}) {
		for _, d := range g.Data {
			if d.Name == name {
				return d, nil
			}
		}
	}
	return nil, fmt.Errorf("DATA %q not found in catalog", name)
}
