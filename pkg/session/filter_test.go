package session_test

import (
	"testing"

	"github.com/gridsociety/mythy/pkg/session"
)

func TestFilterDefaultsExcludeReadOnlyHiddenAndSkip(t *testing.T) {
	f := session.ExportFilter{}
	keep := func(name string, ro, hidden, skip bool, module string) bool {
		return f.KeepData(session.DataInfo{Name: name, ReadOnly: ro, Hidden: hidden, Skip: skip, Module: module}, nil)
	}
	if keep("X", true, false, false, "") {
		t.Error("READONLY excluded by default")
	}
	if keep("X", false, true, false, "") {
		t.Error("hidden excluded by default")
	}
	if keep("X", false, false, true, "") {
		t.Error("SKIP=YES excluded by default")
	}
	if !keep("X", false, false, false, "") {
		t.Error("plain DATA must be kept")
	}
}

func TestFilterModuleGating(t *testing.T) {
	f := session.ExportFilter{}
	enabled := map[string]bool{"SchedaPT100": false, "SchedaMadre": true}
	keep := func(module string) bool {
		return f.KeepData(session.DataInfo{Name: "X", Module: module}, enabled)
	}
	if keep("SchedaPT100") {
		t.Error("module disabled → exclude")
	}
	if !keep("SchedaMadre") {
		t.Error("module enabled → include")
	}
	if !keep("") {
		t.Error("no module gate → include")
	}
}

func TestFilterIncludeDisabledModulesOverride(t *testing.T) {
	// Issue #21: --include-disabled-modules / --all needs to bypass
	// the module gate so MODULE-attribute DATA (RELE_K*, ING_I*, …)
	// land in the export when the operator wants full state capture.
	enabled := map[string]bool{"SchedaPT100": false}
	off := session.ExportFilter{}
	if off.KeepData(session.DataInfo{Name: "X", Module: "SchedaPT100"}, enabled) {
		t.Error("default: module-disabled DATA must be excluded")
	}
	on := session.ExportFilter{IncludeDisabledModules: true}
	if !on.KeepData(session.DataInfo{Name: "X", Module: "SchedaPT100"}, enabled) {
		t.Error("--include-disabled-modules must include DATA whose MODULE is disabled")
	}
}

func TestFilterSkipReasonNamesTheAxis(t *testing.T) {
	// SkipReason underpins the per-category skip-summary lines the
	// export CLI now emits. Verify each axis lights up the right tag,
	// and that "" comes back when nothing trips.
	enabled := map[string]bool{"M": false}
	cases := []struct {
		name string
		d    session.DataInfo
		want string
	}{
		{"skip", session.DataInfo{Name: "X", Skip: true}, "skip"},
		{"readonly", session.DataInfo{Name: "X", ReadOnly: true}, "readonly"},
		{"hidden", session.DataInfo{Name: "X", Hidden: true}, "hidden"},
		{"module-disabled", session.DataInfo{Name: "X", Module: "M"}, "module-disabled"},
		{"kept", session.DataInfo{Name: "X"}, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := session.ExportFilter{}.SkipReason(c.d, enabled)
			if got != c.want {
				t.Errorf("SkipReason = %q, want %q", got, c.want)
			}
		})
	}
}

func TestFilterIncludeReadOnlyOverride(t *testing.T) {
	f := session.ExportFilter{IncludeReadOnly: true}
	if !f.KeepData(session.DataInfo{Name: "X", ReadOnly: true}, nil) {
		t.Error("--include-readonly must override")
	}
}

func TestFilterIncludeSkipOverride(t *testing.T) {
	// Issue #8 added an explicit override for SKIP=YES. With it off,
	// the marker still excludes; with it on, SKIP=YES is kept.
	off := session.ExportFilter{IncludeReadOnly: true, IncludeHidden: true}
	if off.KeepData(session.DataInfo{Name: "X", Skip: true}, nil) {
		t.Error("SKIP=YES must be excluded when IncludeSkip is false (regardless of other includes)")
	}
	on := session.ExportFilter{IncludeSkip: true}
	if !on.KeepData(session.DataInfo{Name: "X", Skip: true}, nil) {
		t.Error("--include-skip must keep SKIP=YES entries")
	}
}
