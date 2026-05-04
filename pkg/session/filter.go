package session

// ExportFilter controls which DATA leaves are included in an export
// (or any other filtered walk). The defaults match SPEC § 4: hidden
// and read-only excluded; SKIP="YES" excluded with no override.
//
// Module gating: pass an enabledModules map; a DATA whose Module is
// non-empty and not present-or-true in the map is excluded.
type ExportFilter struct {
	IncludeHidden   bool // include VISIBILITY="3" groups
	IncludeReadOnly bool // include READONLY="YES" data
}

// DataInfo is a small projection of catalog.Data, kept here so the
// filter logic doesn't depend on the catalog package and can be unit
// tested without a fixture.
type DataInfo struct {
	Name     string
	ReadOnly bool
	Hidden   bool
	Skip     bool
	Module   string
}

// KeepData reports whether the entry passes the filter.
//
// enabledModules is keyed by board name (e.g. "SchedaPT100"). A nil
// or empty map means "no module info available — assume all enabled".
func (f ExportFilter) KeepData(d DataInfo, enabledModules map[string]bool) bool {
	if d.Skip {
		return false
	}
	if d.ReadOnly && !f.IncludeReadOnly {
		return false
	}
	if d.Hidden && !f.IncludeHidden {
		return false
	}
	if d.Module != "" && enabledModules != nil {
		if !enabledModules[d.Module] {
			return false
		}
	}
	return true
}
