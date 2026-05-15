package configio_test

import (
	"context"
	"strings"
	"testing"

	"github.com/gridsociety/mythy/pkg/catalog"
	"github.com/gridsociety/mythy/pkg/configio"
	"github.com/gridsociety/mythy/pkg/session"
	"github.com/gridsociety/mythy/pkg/transport"
)

func mkSession(t *testing.T, f *transport.Fake) *session.Session {
	t.Helper()
	tpl, err := catalog.ParseTemplate("../../testdata/us/TEST-VB0-a")
	if err != nil {
		t.Fatal(err)
	}
	s, err := session.NewWithTransport(f, tpl, catalog.DeviceEntry{Product: "TEST-VX0-a", Identification: 99999})
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestExportScopedYAML(t *testing.T) {
	f := transport.NewFake()
	// Set/Base contents — only MB_address passes the default filter
	// (NomeLinea is READONLY=YES so it's excluded).
	f.OnReadHolding(6145, 1, []uint16{5}) // MB_address
	s := mkSession(t, f)

	bytes, err := configio.Export(context.Background(), s, configio.ExportOptions{
		Scope:  "Set/Base",
		Filter: session.ExportFilter{},
	})
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	got := string(bytes)
	for _, want := range []string{
		"mythy_version: 1",
		"product: TEST-VX0-a",
		"identification: 99999",
		"MB_address: 5",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "NomeLinea") {
		t.Error("READONLY entry NomeLinea must be excluded by default")
	}
}

func TestExportRecordsLocaleInHeader(t *testing.T) {
	// Issue #4: the export captures the locale it ran under into
	// device.locale, so a later import can detect cross-locale round-
	// trips that would silently remap enum labels to zero.
	f := transport.NewFake()
	f.OnReadHolding(6145, 1, []uint16{5})
	s := mkSession(t, f)

	bytes, err := configio.Export(context.Background(), s, configio.ExportOptions{
		Scope:  "Set/Base",
		Locale: "it",
		Filter: session.ExportFilter{},
	})
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	got := string(bytes)
	if !strings.Contains(got, "locale: it") {
		t.Errorf("output must record device.locale: it\n%s", got)
	}

	// Empty Locale: omitempty must skip the field (back-compat with
	// pre-fix exports). Run the same export with Locale unset.
	bytes2, err := configio.Export(context.Background(), s, configio.ExportOptions{
		Scope:  "Set/Base",
		Filter: session.ExportFilter{},
	})
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	if strings.Contains(string(bytes2), "locale:") {
		t.Errorf("empty Locale must be omitted (omitempty) from YAML, got:\n%s", string(bytes2))
	}
}

func TestExportRespectsAndReportsModuleDisabledSkip(t *testing.T) {
	// Issue #21: DATA whose MODULE attribute is reported disabled
	// (the test fixture's TEST_PT100_Setting carries MODULE=
	// "SchedaPT100", which isn't declared in <MODULES> so
	// Session.EnabledModules has no key for it → treated as disabled)
	// must be skipped by default but included when
	// --include-disabled-modules is set. The skip must surface in
	// ExportReport.SkippedDisabledModules so the CLI can render the
	// "skipped N module-disabled" line.
	f := transport.NewFake()
	s := mkSession(t, f)

	// Default filter: module-disabled DATA must be skipped + reported.
	// Scope to Set/PT100 so the export doesn't need to read every Set
	// child — TEST_PT100_Setting is the only leaf in that subgroup
	// and gets filtered out before any wire I/O.
	report := &configio.ExportReport{}
	bytes, err := configio.Export(context.Background(), s, configio.ExportOptions{
		Scope:  "Set/PT100",
		Filter: session.ExportFilter{},
		Report: report,
	})
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	if strings.Contains(string(bytes), "TEST_PT100_Setting") {
		t.Error("module-disabled DATA must NOT be in output by default")
	}
	found := false
	for _, n := range report.SkippedDisabledModules {
		if n == "TEST_PT100_Setting" {
			found = true
		}
	}
	if !found {
		t.Errorf("ExportReport.SkippedDisabledModules must include TEST_PT100_Setting, got %v", report.SkippedDisabledModules)
	}

	// IncludeDisabledModules=true: same fixture, expect the DATA in
	// the output and NOT in the skip report.
	f2 := transport.NewFake()
	f2.OnReadHolding(6149, 1, []uint16{0}) // TEST_PT100_Setting at num=6150 (wire=6149)
	s2 := mkSession(t, f2)
	report2 := &configio.ExportReport{}
	bytes2, err := configio.Export(context.Background(), s2, configio.ExportOptions{
		Scope:  "Set/PT100",
		Filter: session.ExportFilter{IncludeDisabledModules: true},
		Report: report2,
	})
	if err != nil {
		t.Fatalf("Export (IncludeDisabledModules): %v", err)
	}
	if !strings.Contains(string(bytes2), "TEST_PT100_Setting") {
		t.Errorf("IncludeDisabledModules must surface module-disabled DATA in the YAML:\n%s", string(bytes2))
	}
	if len(report2.SkippedDisabledModules) > 0 {
		t.Errorf("IncludeDisabledModules: SkippedDisabledModules must be empty, got %v", report2.SkippedDisabledModules)
	}
}

func TestExportFiresProgressPerLeaf(t *testing.T) {
	// Set/Base under default filter has exactly one kept leaf
	// (MB_address; NomeLinea is READONLY=YES so it's excluded).
	// Progress should fire once before the read with done=0/total=1,
	// and once at the end with done=total=1 and name="".
	f := transport.NewFake()
	f.OnReadHolding(6145, 1, []uint16{5})
	s := mkSession(t, f)

	type call struct {
		done, total int
		name        string
	}
	var calls []call

	if _, err := configio.Export(context.Background(), s, configio.ExportOptions{
		Scope:  "Set/Base",
		Filter: session.ExportFilter{},
		Progress: func(done, total int, name string) {
			calls = append(calls, call{done, total, name})
		},
	}); err != nil {
		t.Fatalf("Export: %v", err)
	}

	if len(calls) != 2 {
		t.Fatalf("got %d progress calls, want 2: %+v", len(calls), calls)
	}
	if calls[0] != (call{done: 0, total: 1, name: "MB_address"}) {
		t.Errorf("first call = %+v, want {0 1 MB_address}", calls[0])
	}
	if calls[1] != (call{done: 1, total: 1, name: ""}) {
		t.Errorf("final call = %+v, want {1 1 \"\"}", calls[1])
	}
}

func TestExportIncludeReadOnly(t *testing.T) {
	f := transport.NewFake()
	f.OnReadHolding(6145, 1, []uint16{5})
	f.OnReadHolding(6199, 10, append([]uint16{0x5341, 0x4D50, 0x4C45}, make([]uint16, 7)...)) // "SAMPLE"
	s := mkSession(t, f)

	bytes, err := configio.Export(context.Background(), s, configio.ExportOptions{
		Scope:  "Set/Base",
		Filter: session.ExportFilter{IncludeReadOnly: true},
	})
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	if !strings.Contains(string(bytes), "NomeLinea") {
		t.Error("--include-readonly must include NomeLinea")
	}
}
