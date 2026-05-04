package session_test

import (
	"context"
	"testing"

	"github.com/gridsociety/mythy/pkg/transport"
)

func TestG61850GetIedName(t *testing.T) {
	f := transport.NewFake()
	// CMD wire 54586, PAR1 wire 51257, PAR2 wire 51258, TRIGGER wire 5147 (= 0x141B), REPLY wire 51256
	f.OnWriteSingleOK(54586) // CMD = 7
	f.OnReadInput(0x141B, 1, []uint16{1})
	// REPLY = "TEMPLATE" packed: 0x5445 0x4D50 0x4C41 0x5445 then NULs
	reply := make([]uint16, 50)
	reply[0] = 0x5445 // 'T''E'
	reply[1] = 0x4D50 // 'M''P'
	reply[2] = 0x4C41 // 'L''A'
	reply[3] = 0x5445 // 'T''E'
	f.OnReadInput(51256, 50, reply)
	s := mkSession(t, f)

	got, err := s.G61850(context.Background(), "GetIedName", "", "")
	if err != nil {
		t.Fatalf("G61850: %v", err)
	}
	if got != "TEMPLATE" {
		t.Errorf("reply = %q, want %q", got, "TEMPLATE")
	}
}

func TestG61850UnknownFunction(t *testing.T) {
	f := transport.NewFake()
	s := mkSession(t, f)
	if _, err := s.G61850(context.Background(), "NotARealFunction", "", ""); err == nil {
		t.Error("expected error for unknown function")
	}
}

func TestG61850SetSkipsReplyRead(t *testing.T) {
	// Set* doesn't update REPLY, so the impl shouldn't FC04-read it.
	f := transport.NewFake()
	f.OnWriteMultiOK(51257)  // PAR1
	f.OnWriteSingleOK(54586) // CMD = 8 (SetIedName)
	f.OnReadInput(0x141B, 1, []uint16{1})
	s := mkSession(t, f)

	if _, err := s.G61850(context.Background(), "SetIedName", "SAMPLE", ""); err != nil {
		// PAR2 is empty so we shouldn't write it
		t.Fatalf("G61850 Set: %v", err)
	}
	// Verify PAR1 was written but PAR2 was not.
	wroteP1 := false
	wroteP2 := false
	for _, w := range f.Writes {
		switch w.Addr {
		case 51257:
			wroteP1 = true
		case 51258:
			wroteP2 = true
		}
	}
	if !wroteP1 {
		t.Error("expected PAR1 write")
	}
	if wroteP2 {
		t.Error("PAR2 was empty; should not have been written")
	}
}
