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
