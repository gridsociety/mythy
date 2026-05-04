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
}

// Export reads the kept catalog leaves under opts.Scope, decodes each,
// and emits YAML bytes following the SPEC § 4 schema. Entries are
// emitted in depth-first menu walk order; group boundaries become
// `# <Path>` head-comments so diffs are readable.
//
// Module gating: Session.EnabledModules() (Plan 3 Task 3a) probes every
// EnableBoard_* register in the catalog once per Session and caches;
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

	type entry struct {
		path  string
		name  string
		value any
	}
	var entries []entry
	seen := make(map[string]struct{})

	for _, g := range scope.WalkGroups(catalog.WalkOptions{IncludeHidden: opts.Filter.IncludeHidden, IncludeReadOnly: true}) {
		for _, d := range g.Data {
			if !opts.Filter.KeepData(toDataInfo(g, d), enabledModules) {
				continue
			}
			if _, dup := seen[d.Name]; dup {
				continue
			}
			seen[d.Name] = struct{}{}
			v, err := s.Read(ctx, d.Name)
			if err != nil {
				return nil, fmt.Errorf("read %s: %w", d.Name, err)
			}
			entries = append(entries, entry{
				path:  g.Path(),
				name:  d.Name,
				value: ValueToYAML(v),
			})
		}
	}

	dev := Device{
		Product:        s.Entry().Product,
		Identification: s.Entry().Identification,
		ExportedAt:     time.Now().Format(time.RFC3339),
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

func toDataInfo(g *catalog.Group, d *catalog.Data) session.DataInfo {
	return session.DataInfo{
		Name:     d.Name,
		ReadOnly: d.ReadOnly,
		Hidden:   g.Visibility == "3",
		Skip:     d.Skip,
		Module:   d.Module,
	}
}
