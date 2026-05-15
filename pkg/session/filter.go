package session

// ExportFilter controls which DATA leaves are included in an export
// (or any other filtered walk). The defaults match SPEC § 4: hidden,
// read-only, SKIP="YES", and DATA whose MODULE attribute is reported
// disabled — all excluded.
//
// SKIP="YES" marks DATA that is operationally hazardous to read or
// write over the live connection — identification fields, comm
// parameters, IP config. By default it stays excluded; the
// IncludeSkip override exists for the legitimate first-time-deploy
// scenario where the operator is intentionally pushing network
// config and accepts that the connection may need to be re-established.
//
// Module gating: pass an enabledModules map; a DATA whose Module is
// non-empty and reported false (or missing) in the map is excluded.
// The IncludeDisabledModules override lifts that — useful when the
// operator wants the full catalog state captured for snapshots /
// migration sources / cross-device diff even if some boards aren't
// installed. Issue #21.
type ExportFilter struct {
	IncludeHidden          bool // include VISIBILITY="3" groups
	IncludeReadOnly        bool // include READONLY="YES" data
	IncludeSkip            bool // include SKIP="YES" data
	IncludeDisabledModules bool // include DATA whose MODULE is reported disabled
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

// SkipReason returns the first filter axis that excludes d, or empty
// when d passes. Useful for diagnostics that want to surface why a
// DATA was dropped from a walk. Categories (in evaluation order):
// "skip", "readonly", "hidden", "module-disabled".
//
// enabledModules is keyed by board name (e.g. "SchedaPT100"). A nil
// or empty map means "no module info available — assume all enabled".
func (f ExportFilter) SkipReason(d DataInfo, enabledModules map[string]bool) string {
	if d.Skip && !f.IncludeSkip {
		return "skip"
	}
	if d.ReadOnly && !f.IncludeReadOnly {
		return "readonly"
	}
	if d.Hidden && !f.IncludeHidden {
		return "hidden"
	}
	if d.Module != "" && enabledModules != nil && !enabledModules[d.Module] && !f.IncludeDisabledModules {
		return "module-disabled"
	}
	return ""
}

// KeepData reports whether the entry passes the filter. Equivalent
// to SkipReason returning the empty string; kept as a thin wrapper
// because most callers don't need the reason category.
func (f ExportFilter) KeepData(d DataInfo, enabledModules map[string]bool) bool {
	return f.SkipReason(d, enabledModules) == ""
}
