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

func TestSetCompoundHonoursCompoundOverrides(t *testing.T) {
	// Issue #6: encoding a compound DATA must honour the per-instance
	// CompoundOverrides parsed from nested <DATA> children. TEST_SOGLIA
	// in the fixture declares SOGLIA_T class with State as ENUM (1 reg
	// in the class) but overrides it to ENUM_LONG (2 regs) in the
	// instance, so the encoded payload must be 3 regs.
	f := transport.NewFake()
	f.OnReadInput(0x1402, 1, []uint16{1}) // START_CHANGE_DB
	f.OnWriteMultiOK(6148)                // TEST_SOGLIA wire = num-1 = 6148
	f.OnReadInput(0x1403, 1, []uint16{1}) // END_CHANGE_DB
	s := mkSession(t, f)

	if err := s.Set(context.Background(), "TEST_SOGLIA", map[string]any{
		"State":  int64(1),
		"Pickup": uint64(42),
	}); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if len(f.Writes) != 1 {
		t.Fatalf("Writes = %d, want 1", len(f.Writes))
	}
	w := f.Writes[0]
	if w.FC != 16 {
		t.Errorf("FC = %d, want 16 (compound goes through WriteMultipleRegisters)", w.FC)
	}
	if w.Addr != 6148 {
		t.Errorf("Addr = %d, want 6148", w.Addr)
	}
	// State ENUM_LONG=1 → regs[0]=0x0001, regs[1]=0x0000;
	// Pickup UWORD=42 → regs[2]=0x002A.
	want := []uint16{0x0001, 0x0000, 0x002A}
	if len(w.Values) != 3 || w.Values[0] != want[0] || w.Values[1] != want[1] || w.Values[2] != want[2] {
		t.Errorf("Values = %v, want %v (pre-fix would emit only 2 regs)", w.Values, want)
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
