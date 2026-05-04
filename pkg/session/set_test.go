package session_test

import (
	"context"
	"testing"

	"github.com/gridsociety/mythy/pkg/transport"
)

func TestSetUByteParam(t *testing.T) {
	f := transport.NewFake()
	f.OnReadInput(0x1402, 1, []uint16{1})
	f.OnWriteSingleOK(6145) // MB_address wire
	f.OnReadInput(0x1403, 1, []uint16{1})
	s := mkSession(t, f)

	if err := s.Set(context.Background(), "MB_address", uint64(5)); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if got := f.Writes; len(got) != 1 || got[0].Addr != 6145 || got[0].Values[0] != 5 {
		t.Errorf("Writes = %+v", got)
	}
}

func TestSetRangeRejected(t *testing.T) {
	// MB_address has TIPO=UBYTE → max 255. Pass 999 and expect a refusal.
	f := transport.NewFake()
	s := mkSession(t, f)
	err := s.Set(context.Background(), "MB_address", uint64(999))
	if err == nil {
		t.Error("expected validation error for value > 255")
	}
}

func TestSetEnumByLabel(t *testing.T) {
	f := transport.NewFake()
	f.OnReadInput(0x1402, 1, []uint16{1})
	f.OnWriteSingleOK(15362) // MB_baudrate wire (num 15363 - 1)
	f.OnReadInput(0x1403, 1, []uint16{1})
	s := mkSession(t, f)
	if err := s.Set(context.Background(), "MB_baudrate", "19200 baud"); err != nil {
		t.Fatalf("Set enum: %v", err)
	}
	if got := f.Writes[0].Values[0]; got != 4 {
		t.Errorf("wire value = %d, want 4 (= 19200 baud OVERRIDE)", got)
	}
}

func TestSetManyOneTransaction(t *testing.T) {
	f := transport.NewFake()
	f.OnReadInput(0x1402, 1, []uint16{1})
	f.OnWriteSingleOK(6145)
	f.OnWriteSingleOK(15362)
	f.OnReadInput(0x1403, 1, []uint16{1})
	s := mkSession(t, f)

	if err := s.SetMany(context.Background(), map[string]any{
		"MB_address":  uint64(5),
		"MB_baudrate": "19200 baud",
	}); err != nil {
		t.Fatalf("SetMany: %v", err)
	}
	// Order is deterministic (alphabetical) so tests can assert the trail.
	if len(f.Writes) != 2 {
		t.Errorf("Writes len = %d, want 2", len(f.Writes))
	}
}
