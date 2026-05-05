package configio_test

import (
	"context"
	"testing"

	"github.com/gridsociety/mythy/pkg/configio"
	"github.com/gridsociety/mythy/pkg/session"
	"github.com/gridsociety/mythy/pkg/transport"
)

// TestExportImportNoOp guarantees that a fresh `export → parse → validate
// → apply` cycle produces zero writes. This catches encoding asymmetries
// where a value decodes one way and re-encodes differently.
//
// The fake's OnReadHolding(addr, qty, regs) sets a map entry that returns
// the same regs on every call to ReadHoldingRegisters(addr, qty), so
// scripting once is enough — Export reads, Apply's internal Diff reads
// again, both hit the same fake response.
func TestExportImportNoOp(t *testing.T) {
	f := transport.NewFake()
	f.OnReadHolding(6145, 1, []uint16{5})  // MB_address = 5
	f.OnReadHolding(15362, 1, []uint16{3}) // MB_baudrate = "9600 baud" (OVERRIDE=3)
	s := mkSession(t, f)

	bytes1, err := configio.Export(context.Background(), s, configio.ExportOptions{
		Scope:  "Set/Base",
		Filter: session.ExportFilter{},
	})
	if err != nil {
		t.Fatalf("Export: %v", err)
	}

	parsed, err := configio.Parse(bytes1)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if err := configio.Validate(parsed, s.Template(), s.Entry().Product); err != nil {
		t.Fatalf("Validate: %v", err)
	}
	report, err := configio.Apply(context.Background(), s, parsed, configio.ApplyOptions{})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if len(report.Applied) != 0 {
		t.Errorf("re-import of fresh export must be a no-op; wrote: %v", report.Applied)
	}
	if len(f.Writes) != 0 {
		t.Errorf("no-op import must not generate any writes; got %+v", f.Writes)
	}
}

func TestDiffRoundTripFromIncludeReadonlySnapshot(t *testing.T) {
	// Acceptance criterion #4 from issue #1: a YAML produced by
	// `mythy export --include-readonly` and re-fed to `mythy diff`
	// against the same live device must report zero differences in
	// the default (filtered) view, even if runtime state has drifted
	// between export-time and diff-time.
	//
	// We drive this by hand-building a ConfigFile that pretends to be
	// such a snapshot: writable Set/ values match the device, but the
	// runtime/state values disagree (simulating drift). With the
	// default filter the diff must be empty; the fake transport
	// panics on unscripted reads, so any read attempt on a filtered
	// key would fail the test.
	f := transport.NewFake()
	f.OnReadHolding(6145, 1, []uint16{5})  // MB_address current=5 (matches file)
	f.OnReadHolding(15362, 1, []uint16{3}) // MB_baudrate current="9600 baud" (matches)
	s := mkSession(t, f)

	cf := &configio.ConfigFile{
		MythyVersion: 1,
		Device:       configio.Device{Product: "TEST-VX0-a"},
		Settings: map[string]any{
			// Config — matches device.
			"MB_address":  int64(5),
			"MB_baudrate": "9600 baud",
			// Runtime/state — file shows stale snapshot values; device
			// has moved on. Default filter must skip these.
			"UL1":           int64(230000), // file says 230 V long-ago; device has drifted
			"F25_CONT_P_Sc": map[string]any{"Tipo": "Manual", "MaxValore": 9999, "Valore": 7},
			"NomeLinea":     "OLD",
		},
	}

	changes, err := configio.Diff(context.Background(), s, cf, configio.DiffOptions{})
	if err != nil {
		t.Fatalf("default Diff: %v", err)
	}
	if len(changes) != 0 {
		t.Errorf("default-filter diff must be empty, got %d: %+v", len(changes), changes)
	}
}
