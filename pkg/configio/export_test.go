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
