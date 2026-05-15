package configio

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/gridsociety/mythy/pkg/catalog"
	"github.com/gridsociety/mythy/pkg/session"
	"gopkg.in/yaml.v3"
)

// ExportOptions controls what Export writes.
type ExportOptions struct {
	Scope  string               // empty = whole device
	Filter session.ExportFilter // READONLY/hidden defaults; SKIP always excluded

	// Locale is recorded into device.locale in the YAML so import /
	// diff / validate can detect cross-locale round-trips that would
	// otherwise silently remap enum labels to zero. Empty means "don't
	// record" (legacy behaviour); the CLI always passes a value.
	Locale string

	// Progress, when non-nil, is invoked once before each catalog leaf
	// is read, and once more at the end with done == total and name == "".
	// Use the final call to clear any in-progress UI. The full-device
	// export reads ~7,400 leaves and takes ~4 minutes on LAN; reporting
	// progress matters for CLI UX.
	Progress func(done, total int, name string)

	// Report, when non-nil, is populated with per-category lists of
	// every DATA the filter dropped. The CLI uses it to surface
	// "skipped N module-disabled key(s)" style summary lines so the
	// omission isn't silent (issue #21).
	Report *ExportReport
}

// ExportReport summarises what Export skipped, grouped by the filter
// axis that excluded each DATA. Empty when every change was kept.
type ExportReport struct {
	SkippedSkip            []string // SKIP="YES" (use --include-skip)
	SkippedReadOnly        []string // READONLY="YES" (use --include-readonly)
	SkippedHidden          []string // VISIBILITY="3" cascade (use --include-hidden)
	SkippedDisabledModules []string // MODULE reported off (use --include-disabled-modules)
}

// Export reads the kept catalog leaves under opts.Scope, decodes each,
// and emits YAML bytes following the SPEC § 4 schema. Entries are
// emitted in depth-first menu walk order; group boundaries become
// `# <Path>` head-comments so diffs are readable.
//
// Module gating: Session.EnabledModules() probes every EnableBoard_*
// register in the catalog once per Session and caches the result;
// Export consults the cache to skip DATA tied to disabled boards.
//
// Output format: a yaml.Node AST is built explicitly so map-key order
// follows the menu walk (default yaml.Marshal sorts alphabetically) and
// per-section HeadComment annotations land on the right keys. Duplicate
// DATA NAMEs in the catalog (e.g. the same name reachable through two
// menu paths) are deduped — the first occurrence wins.
func Export(ctx context.Context, s *session.Session, opts ExportOptions) ([]byte, error) {
	tpl := s.Template()
	if tpl == nil || tpl.Menu == nil {
		return nil, fmt.Errorf("Export: no template loaded")
	}
	scope := tpl.Menu
	if opts.Scope != "" {
		scope = scope.FindGroup(opts.Scope)
		if scope == nil {
			return nil, fmt.Errorf("scope %q not found", opts.Scope)
		}
	}

	enabledModules, err := s.EnabledModules(ctx)
	if err != nil {
		return nil, fmt.Errorf("module probe: %w", err)
	}

	// Phase 1: collect every leaf that passes the filter, in menu-walk
	// order. No I/O here — the count drives the progress total.
	type leafRef struct {
		path string
		name string
		data *catalog.Data
	}
	var leaves []leafRef
	seen := make(map[string]struct{})
	for _, g := range scope.WalkGroups(catalog.WalkOptions{IncludeHidden: opts.Filter.IncludeHidden, IncludeReadOnly: true}) {
		for _, d := range g.Data {
			info := toDataInfo(g, d)
			if reason := opts.Filter.SkipReason(info, enabledModules); reason != "" {
				if opts.Report != nil && !alreadySeen(seen, d.Name) {
					// Categorise the skip so the CLI can render
					// "skipped N <category> key(s)" summary lines.
					// Same dedup as the kept-leaf path: a DATA reachable
					// through multiple menu paths counts once.
					seen[d.Name] = struct{}{}
					switch reason {
					case "skip":
						opts.Report.SkippedSkip = append(opts.Report.SkippedSkip, d.Name)
					case "readonly":
						opts.Report.SkippedReadOnly = append(opts.Report.SkippedReadOnly, d.Name)
					case "hidden":
						opts.Report.SkippedHidden = append(opts.Report.SkippedHidden, d.Name)
					case "module-disabled":
						opts.Report.SkippedDisabledModules = append(opts.Report.SkippedDisabledModules, d.Name)
					}
				}
				continue
			}
			if _, dup := seen[d.Name]; dup {
				continue
			}
			seen[d.Name] = struct{}{}
			leaves = append(leaves, leafRef{path: g.Path(), name: d.Name, data: d})
		}
	}

	// Phase 2: read each leaf, reporting progress per item.
	type entry struct {
		path  string
		name  string
		value any
	}
	entries := make([]entry, 0, len(leaves))
	total := len(leaves)
	for i, l := range leaves {
		if opts.Progress != nil {
			opts.Progress(i, total, l.name)
		}
		v, err := s.Read(ctx, l.name)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", l.name, err)
		}
		entries = append(entries, entry{path: l.path, name: l.name, value: ValueToYAML(v)})
	}
	if opts.Progress != nil {
		// Sentinel call — name == "" with done == total means "finished;
		// clear any in-progress UI".
		opts.Progress(total, total, "")
	}

	dev := Device{
		Product:        s.Entry().Product,
		Identification: s.Entry().Identification,
		ExportedAt:     time.Now().Format(time.RFC3339),
		Locale:         opts.Locale,
	}
	if id := s.Ident(); id != nil {
		dev.Identification = int(id.Identification)
		dev.SerialNumber = int64(id.SerialNumber)
		dev.FwRelease = fmt.Sprintf("%04X", id.FwRelease)
		dev.ProtocolRelease = fmt.Sprintf("%04X", id.ProtocolRelease)
	}

	settingsNode := &yaml.Node{Kind: yaml.MappingNode}
	currentPath := ""
	for _, e := range entries {
		key := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: e.name}
		if e.path != currentPath {
			key.HeadComment = e.path
			currentPath = e.path
		}
		valNode := &yaml.Node{}
		if err := valNode.Encode(e.value); err != nil {
			return nil, fmt.Errorf("encode %s: %w", e.name, err)
		}
		settingsNode.Content = append(settingsNode.Content, key, valNode)
	}

	devNode := &yaml.Node{}
	if err := devNode.Encode(dev); err != nil {
		return nil, fmt.Errorf("encode device: %w", err)
	}
	versionNode := &yaml.Node{}
	if err := versionNode.Encode(1); err != nil {
		return nil, err
	}
	root := &yaml.Node{Kind: yaml.DocumentNode, Content: []*yaml.Node{
		{
			Kind: yaml.MappingNode,
			Content: []*yaml.Node{
				{Kind: yaml.ScalarNode, Tag: "!!str", Value: "mythy_version"}, versionNode,
				{Kind: yaml.ScalarNode, Tag: "!!str", Value: "device"}, devNode,
				{Kind: yaml.ScalarNode, Tag: "!!str", Value: "settings"}, settingsNode,
			},
		},
	}}

	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(root); err != nil {
		return nil, err
	}
	if err := enc.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// alreadySeen returns true if name is in the seen set without
// inserting it. Mirror of the seen-map deduplication used for kept
// leaves so skip categorisation only counts a DATA the first time
// it's encountered in the walk.
func alreadySeen(seen map[string]struct{}, name string) bool {
	_, ok := seen[name]
	return ok
}

func toDataInfo(g *catalog.Group, d *catalog.Data) session.DataInfo {
	// "Hidden" covers both the containing group's VISIBILITY="3"
	// (Administrator subtree wholesale) and the DATA's own
	// VISIBILITY="3" (outliers like CodeNum, sitting in an
	// otherwise-visible group). Symmetric with the import-side filter
	// in pkg/configio/apply.go:visibleDataNames.
	return session.DataInfo{
		Name:     d.Name,
		ReadOnly: d.ReadOnly,
		Hidden:   g.Visibility == "3" || d.Visibility == "3",
		Skip:     d.Skip,
		Module:   d.Module,
	}
}
