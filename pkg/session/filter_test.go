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
		t.Error("SKIP=YES always excluded")
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

func TestFilterIncludeReadOnlyOverride(t *testing.T) {
	f := session.ExportFilter{IncludeReadOnly: true}
	if !f.KeepData(session.DataInfo{Name: "X", ReadOnly: true}, nil) {
		t.Error("--include-readonly must override")
	}
}

func TestFilterSkipNeverOverridable(t *testing.T) {
	f := session.ExportFilter{IncludeReadOnly: true, IncludeHidden: true}
	if f.KeepData(session.DataInfo{Name: "X", Skip: true}, nil) {
		t.Error("SKIP=YES has no override")
	}
}
