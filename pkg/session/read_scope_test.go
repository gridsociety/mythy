package session_test

import (
	"context"
	"testing"

	"github.com/gridsociety/mythy/pkg/transport"
)

func TestReadScopeMergesContiguous(t *testing.T) {
	f := transport.NewFake()
	// "Set/Base" contains MB_address (FC03 wire 6145, dim=1) and
	// NomeLinea (FC03 wire 6199, dim=10). Strict gap=0 → 2 batches.
	f.OnReadHolding(6145, 1, []uint16{5})
	f.OnReadHolding(6199, 10, []uint16{0x5341, 0x4D50, 0x4C45, 0, 0, 0, 0, 0, 0, 0}) // "SAMPLE"
	s := mkSession(t, f)

	out, err := s.ReadScope(context.Background(), "Set/Base", false /* includeHidden */)
	if err != nil {
		t.Fatalf("ReadScope: %v", err)
	}
	if out["MB_address"].Number != 5 {
		t.Errorf("MB_address.Number = %d", out["MB_address"].Number)
	}
	if out["NomeLinea"].Str != "SAMPLE" {
		t.Errorf("NomeLinea.Str = %q", out["NomeLinea"].Str)
	}
}

func TestReadScopeUnknownPathErrors(t *testing.T) {
	f := transport.NewFake()
	s := mkSession(t, f)
	_, err := s.ReadScope(context.Background(), "Nonexistent/Path", false)
	if err == nil {
		t.Error("expected error for unknown scope")
	}
}
