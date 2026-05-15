package configio

import (
	"context"
	"fmt"

	"github.com/gridsociety/mythy/pkg/catalog"
	"github.com/gridsociety/mythy/pkg/session"
)

// Report describes the outcome of Apply or ApplyDryRun.
//
// Applied / WouldApply and Skipped are decisions (written, would-be-
// written, or refused). InSkipCategory and InHiddenCategory are
// CLASSIFICATIONS — they list every changed key in those catalog
// categories regardless of whether opt-ins promoted them to
// Applied/WouldApply this run. The CLI uses the intersection of
// In*Category and Applied/Skipped to render the banner: "writing N
// connection-disruptive keys" vs "skipping N hidden keys (would need
// --include-hidden)".
type Report struct {
	Applied    []string // written by Apply
	WouldApply []string // what ApplyDryRun would have written
	Skipped    []string // READONLY="YES" — never written, no opt-in lifts this

	InSkipCategory   []string // every changed key with SKIP="YES"
	InHiddenCategory []string // every changed key reachable only through VISIBILITY="3" groups
}

// ApplyOptions configures Apply / ApplyDryRun.
//
// IncludeSkip and IncludeHidden gate writes to two locale-independent
// catalog-marker categories that are dangerous by default. See the
// issue-#8 design note for the rationale.
type ApplyOptions struct {
	// Progress is forwarded to the inner Diff so callers can render
	// one progress UI for the whole import.
	Progress func(done, total int, name string)

	// IncludeSkip allows writes to DATA whose catalog declares
	// SKIP="YES" — identification, comm parameters, IP config. Writing
	// any of these over the live Modbus connection typically drops the
	// connection mid-transaction.
	IncludeSkip bool

	// IncludeHidden allows writes to DATA reachable only through
	// VISIBILITY="3" groups (the Administrator / Menu Riservato subtree).
	// These include device identification fields (CodeNum) and module
	// enable flags (EnableBoard_*) whose mis-configuration can leave the
	// device in a state requiring a factory reset.
	IncludeHidden bool
}

// Apply computes the diff, then writes only the changed keys inside
// one edit transaction (auto-bundled by SetMany).
//
// READONLY="YES" keys are always refused (mythy-side defensive — the
// device empirically does not enforce read-only). SKIP="YES" and
// effectively-hidden keys are refused unless their corresponding
// opt-in is set. In all cases the Report fully classifies every
// changed key so callers can render accurate diagnostics.
//
// The product-mismatch check lives in Validate; the caller is expected
// to have run it already (mythy import does).
func Apply(ctx context.Context, s *session.Session, cf *ConfigFile, opts ApplyOptions) (*Report, error) {
	changes, err := Diff(ctx, s, cf, DiffOptions{Progress: opts.Progress, IncludeAll: true})
	if err != nil {
		return nil, err
	}
	if len(changes) == 0 {
		return &Report{}, nil
	}

	tpl := s.Template()
	visible := visibleDataNames(tpl)
	pairs := make(map[string]any, len(changes))
	rep := &Report{}
	for _, c := range changes {
		d, err := lookupData(tpl, c.Name)
		if err != nil {
			return nil, err
		}
		category := classify(d, c.Name, visible)
		recordCategory(rep, c.Name, category)
		if !shouldWrite(category, opts) {
			continue
		}
		typed, err := YAMLToCodec(d.Tipo, d.Enum, c.File)
		if err != nil {
			return nil, fmt.Errorf("encode %s: %w", c.Name, err)
		}
		pairs[c.Name] = typed
		rep.Applied = append(rep.Applied, c.Name)
	}
	if len(pairs) > 0 {
		if err := s.SetMany(ctx, pairs); err != nil {
			return nil, err
		}
	}
	return rep, nil
}

// ApplyDryRun returns the same classification Apply would produce, but
// without sending any writes. WouldApply is populated where Apply
// would have populated Applied.
func ApplyDryRun(ctx context.Context, s *session.Session, cf *ConfigFile, opts ApplyOptions) (*Report, error) {
	changes, err := Diff(ctx, s, cf, DiffOptions{Progress: opts.Progress, IncludeAll: true})
	if err != nil {
		return nil, err
	}
	tpl := s.Template()
	visible := visibleDataNames(tpl)
	rep := &Report{}
	for _, c := range changes {
		d, err := lookupData(tpl, c.Name)
		if err != nil {
			return nil, err
		}
		category := classify(d, c.Name, visible)
		recordCategory(rep, c.Name, category)
		if shouldWrite(category, opts) {
			rep.WouldApply = append(rep.WouldApply, c.Name)
		}
	}
	return rep, nil
}

// changeCategory is internal — it labels a single changed key with the
// catalog marker that determines whether opt-ins are needed.
type changeCategory int

const (
	catWritable changeCategory = iota // no danger marker → write by default
	catReadOnly                       // never written (mythy-side defensive)
	catSkip                           // SKIP="YES" — needs IncludeSkip
	catHidden                         // VISIBILITY="3" cascade — needs IncludeHidden
)

func classify(d *catalog.Data, name string, visible map[string]bool) changeCategory {
	switch {
	case d.ReadOnly:
		return catReadOnly
	case d.Skip:
		return catSkip
	case !visible[name]:
		return catHidden
	default:
		return catWritable
	}
}

func recordCategory(rep *Report, name string, c changeCategory) {
	switch c {
	case catReadOnly:
		rep.Skipped = append(rep.Skipped, name)
	case catSkip:
		rep.InSkipCategory = append(rep.InSkipCategory, name)
	case catHidden:
		rep.InHiddenCategory = append(rep.InHiddenCategory, name)
	}
}

func shouldWrite(c changeCategory, opts ApplyOptions) bool {
	switch c {
	case catWritable:
		return true
	case catReadOnly:
		return false
	case catSkip:
		return opts.IncludeSkip
	case catHidden:
		return opts.IncludeHidden
	}
	return false
}

// visibleDataNames returns the set of DATA names reachable through at
// least one non-hidden menu path and not marked hidden at the DATA
// level. VISIBILITY="3" can appear on the containing <GROUP> (catches
// the Administrator subtree wholesale) or directly on the <DATA>
// (catches outliers like CodeNum that sit in an otherwise-visible
// group). Both are treated identically.
//
// A DATA appearing in multiple groups is "visible" if any of those
// menu occurrences passes both checks — the most permissive
// interpretation. The complement is the set we treat as effectively
// hidden for #8 scope filtering.
func visibleDataNames(tpl *catalog.Template) map[string]bool {
	out := make(map[string]bool)
	if tpl == nil || tpl.Menu == nil {
		return out
	}
	for _, g := range tpl.Menu.WalkGroups(catalog.WalkOptions{IncludeHidden: false, IncludeReadOnly: true}) {
		for _, d := range g.Data {
			if d.Visibility == "3" {
				continue
			}
			out[d.Name] = true
		}
	}
	return out
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
