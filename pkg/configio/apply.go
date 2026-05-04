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
	Skipped    []string // names that Diff flagged as different but Apply refuses to write (READONLY="YES" in the catalog)
}

// Apply computes the diff, then writes only the changed keys inside
// one edit transaction (auto-bundled by SetMany).
//
// READONLY DATA (`READONLY="YES"` in the menu) is never written, even
// if the file disagrees with the live device — those entries land in
// report.Skipped instead. This is a defensive safety: when a user
// exports with --include-readonly and edits anything, we don't want
// to send writes to registers the device wouldn't accept anyway.
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
	var skipped []string
	for _, c := range changes {
		d, err := lookupData(tpl, c.Name)
		if err != nil {
			return nil, err
		}
		if d.ReadOnly {
			skipped = append(skipped, c.Name)
			continue
		}
		typed, err := YAMLToCodec(d.Tipo, d.Enum, c.File)
		if err != nil {
			return nil, fmt.Errorf("encode %s: %w", c.Name, err)
		}
		pairs[c.Name] = typed
		applied = append(applied, c.Name)
	}
	if len(pairs) > 0 {
		if err := s.SetMany(ctx, pairs); err != nil {
			return nil, err
		}
	}
	return &Report{Applied: applied, Skipped: skipped}, nil
}

// ApplyDryRun returns the list of keys that Apply would write, plus
// any READONLY DATA the file disagrees on (which a real Apply would
// silently skip).
func ApplyDryRun(ctx context.Context, s *session.Session, cf *ConfigFile) (*Report, error) {
	changes, err := Diff(ctx, s, cf)
	if err != nil {
		return nil, err
	}
	tpl := s.Template()
	rep := &Report{}
	for _, c := range changes {
		d, err := lookupData(tpl, c.Name)
		if err == nil && d.ReadOnly {
			rep.Skipped = append(rep.Skipped, c.Name)
			continue
		}
		rep.WouldApply = append(rep.WouldApply, c.Name)
	}
	return rep, nil
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
