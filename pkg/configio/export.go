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
// `# <Path>` comments so diffs are readable.
//
// Module gating: Session.EnabledModules() (Plan 3 Task 3a) probes
// every EnableBoard_* register in the catalog once per Session and
// caches; Export consults the cache to skip DATA tied to disabled
// boards.
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

	// Walk and group by GROUP path so we can emit section comments.
	type entry struct {
		path  string
		name  string
		value any
	}
	var entries []entry

	for _, g := range scope.WalkGroups(catalog.WalkOptions{IncludeHidden: opts.Filter.IncludeHidden, IncludeReadOnly: true}) {
		for _, d := range g.Data {
			if !opts.Filter.KeepData(toDataInfo(g, d), enabledModules) {
				continue
			}
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

	id := s.Ident()
	cf := ConfigFile{
		MythyVersion: 1,
		Device: Device{
			Product:        s.Entry().Product,
			Identification: s.Entry().Identification,
			ExportedAt:     time.Now().Format(time.RFC3339),
		},
		Settings: make(map[string]any, len(entries)),
	}
	if id != nil {
		cf.Device.Identification = int(id.Identification)
		cf.Device.SerialNumber = int64(id.SerialNumber)
		cf.Device.FwRelease = fmt.Sprintf("%04X", id.FwRelease)
		cf.Device.ProtocolRelease = fmt.Sprintf("%04X", id.ProtocolRelease)
	}
	for _, e := range entries {
		cf.Settings[e.name] = e.value
	}

	// Emit YAML manually to control key order + section comments.
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(struct {
		MythyVersion int    `yaml:"mythy_version"`
		Device       Device `yaml:"device"`
	}{1, cf.Device}); err != nil {
		return nil, err
	}
	if err := enc.Close(); err != nil {
		return nil, err
	}
	buf.WriteString("\nsettings:\n")
	currentPath := ""
	for _, e := range entries {
		if e.path != currentPath {
			fmt.Fprintf(&buf, "\n  # %s\n", e.path)
			currentPath = e.path
		}
		val, err := yaml.Marshal(e.value)
		if err != nil {
			return nil, err
		}
		fmt.Fprintf(&buf, "  %s: %s", e.name, string(val))
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
