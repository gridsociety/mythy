package session_test

import (
	"context"
	"testing"

	"github.com/gridsociety/mythy/pkg/catalog"
	"github.com/gridsociety/mythy/pkg/session"
	"github.com/gridsociety/mythy/pkg/transport"
)

func mkSession(t *testing.T, f *transport.Fake) *session.Session {
	t.Helper()
	tpl, err := catalog.ParseTemplate("../../testdata/us/TEST-VB0-a")
	if err != nil {
		t.Fatal(err)
	}
	s, err := session.NewWithTransport(f, tpl, catalog.DeviceEntry{Product: "TEST-VX0-a"})
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestReadULONG(t *testing.T) {
	f := transport.NewFake()
	// UL1 wire 39999, dim=2, low-word-first ULONG
	f.OnReadInput(39999, 2, []uint16{0x9876, 0x0001}) // 0x00019876 = 104566
	s := mkSession(t, f)

	v, err := s.Read(context.Background(), "UL1")
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if v.Number != 104566 {
		t.Errorf("Number = %d, want 104566", v.Number)
	}
	if v.Tipo != "ULONG" {
		t.Errorf("Tipo = %q", v.Tipo)
	}
}

func TestReadCompoundContatore(t *testing.T) {
	f := transport.NewFake()
	// F25_CONT_P_Sc wire 40009, dim=3, CONTATORE
	f.OnReadInput(40009, 3, []uint16{0x0001, 0x0000, 0x270F})
	s := mkSession(t, f)

	v, err := s.Read(context.Background(), "F25_CONT_P_Sc")
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if v.Tipo != "CONTATORE" {
		t.Errorf("Tipo = %q", v.Tipo)
	}
	if v.Compound["MaxValore"].Number != 9999 {
		t.Errorf("MaxValore = %d", v.Compound["MaxValore"].Number)
	}
}

func TestReadUnknownNameErrors(t *testing.T) {
	f := transport.NewFake()
	s := mkSession(t, f)
	_, err := s.Read(context.Background(), "NotARealName")
	if err == nil {
		t.Error("expected error for unknown name")
	}
}

func TestReadHoldingRegisterForWREG(t *testing.T) {
	f := transport.NewFake()
	// MB_address wire 6145, dim=1, WREG → FC03
	f.OnReadHolding(6145, 1, []uint16{5})
	s := mkSession(t, f)

	v, err := s.Read(context.Background(), "MB_address")
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if v.Number != 5 {
		t.Errorf("Number = %d, want 5", v.Number)
	}
}
