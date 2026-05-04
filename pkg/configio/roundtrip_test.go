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
