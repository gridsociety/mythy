package catalog

// WalkOptions controls Walk behavior.
//
// IncludeHidden defaults to false; groups with VISIBILITY="3" are skipped
// because they correspond to the Administrator / "Menu Riservato" pane in
// ThyVisor that is not normally relevant for engineers using mythy.
//
// IncludeReadOnly is preserved on the struct for callers that pass it,
// but WalkData does NOT filter on it. Read-only filtering lives in
// pkg/session.ExportFilter — that's the single source of truth.
//
// Module, when non-empty, restricts the walk to DATA entries whose MODULE
// attribute matches OR is empty. Use this when you've identified which
// hardware modules are installed and want to skip data tied to absent ones.
type WalkOptions struct {
	IncludeHidden   bool
	IncludeReadOnly bool // accepted for compat; not consulted by WalkData
	Module          string
}

// WalkData yields every <DATA> leaf reachable from g, depth-first,
// in document order, filtered by opts.IncludeHidden and opts.Module.
//
// IncludeReadOnly is intentionally NOT filtered here. The export
// pipeline (pkg/configio + pkg/session.ExportFilter) is the single
// source of truth for read-only filtering. WalkData yields every
// leaf; the caller decides which to keep.
func (g *Group) WalkData(opts WalkOptions) []*Data {
	if g == nil {
		return nil
	}
	if !opts.IncludeHidden && g.Visibility == "3" {
		return nil
	}
	var out []*Data
	for _, d := range g.Data {
		if opts.Module != "" && d.Module != "" && d.Module != opts.Module {
			continue
		}
		out = append(out, d)
	}
	for _, c := range g.Children {
		out = append(out, c.WalkData(opts)...)
	}
	return out
}

// WalkCommands returns every <COMMAND> leaf reachable from g, depth-first,
// in document order, filtered by opts.IncludeHidden.
func (g *Group) WalkCommands(opts WalkOptions) []*Command {
	if g == nil {
		return nil
	}
	if !opts.IncludeHidden && g.Visibility == "3" {
		return nil
	}
	var out []*Command
	out = append(out, g.Commands...)
	for _, c := range g.Children {
		out = append(out, c.WalkCommands(opts)...)
	}
	return out
}

// WalkGroups yields every <GROUP> reachable from g (including g itself),
// depth-first, in document order, filtered by opts.IncludeHidden.
func (g *Group) WalkGroups(opts WalkOptions) []*Group {
	if g == nil {
		return nil
	}
	if !opts.IncludeHidden && g.Visibility == "3" {
		return nil
	}
	out := []*Group{g}
	for _, c := range g.Children {
		out = append(out, c.WalkGroups(opts)...)
	}
	return out
}
