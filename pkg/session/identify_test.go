package session_test

import (
	"context"
	"testing"

	"github.com/gridsociety/mythy/pkg/catalog"
	"github.com/gridsociety/mythy/pkg/session"
	"github.com/gridsociety/mythy/pkg/transport"
)

func TestIdentifyDecodesDiscovery(t *testing.T) {
	f := transport.NewFake()
	// Real bytes from SPEC § 2.3 example: 4660 / 100000 / 0x0100 / 0x0000.
	f.OnReadInput(0x143E, 5, []uint16{0x1234, 0x86A0, 0x0001, 0x0100, 0x0000})
	// ENABLE_SEC_MODE = 0 (off)
	f.OnReadInput(54295-1, 1, []uint16{0})

	tpl, _ := catalog.ParseTemplate("../../testdata/us/TEST-VB0-a")
	s, _ := session.NewWithTransport(f, tpl, catalog.DeviceEntry{Product: "PROX-VX0-e"})
	id, err := s.Identify(context.Background())
	if err != nil {
		t.Fatalf("Identify: %v", err)
	}
	if id.Identification != 4660 {
		t.Errorf("Identification = %d", id.Identification)
	}
	if id.SerialNumber != 100000 {
		t.Errorf("SerialNumber = %d", id.SerialNumber)
	}
	if id.FwRelease != 0x0100 {
		t.Errorf("FwRelease = 0x%04X", id.FwRelease)
	}
	if s.SecureMode() {
		t.Error("SecureMode must be false when ENABLE_SEC_MODE=0")
	}
}

func TestIdentifyDetectsSecureMode(t *testing.T) {
	f := transport.NewFake()
	f.OnReadInput(0x143E, 5, []uint16{0x1234, 0x0001, 0x0000, 0x0100, 0x0000})
	f.OnReadInput(54295-1, 1, []uint16{1}) // secure mode ON

	tpl, _ := catalog.ParseTemplate("../../testdata/us/TEST-VB0-a")
	s, _ := session.NewWithTransport(f, tpl, catalog.DeviceEntry{})
	if _, err := s.Identify(context.Background()); err != nil {
		t.Fatalf("Identify: %v", err)
	}
	if !s.SecureMode() {
		t.Error("SecureMode must be true when ENABLE_SEC_MODE=1")
	}
}

func TestIdentifyHandlesMissingSecureModeRegister(t *testing.T) {
	// On firmware that doesn't have ENABLE_SEC_MODE the read returns
	// exception 0x02 (illegal data address). Identify must still succeed
	// and treat the device as secure-mode-off.
	f := transport.NewFake()
	f.OnReadInput(0x143E, 5, []uint16{0x1234, 0x0001, 0x0000, 0x0100, 0x0000})
	f.OnReadInputException(54295-1, 1, &transport.ModbusException{FC: 4, Code: 0x02})

	tpl, _ := catalog.ParseTemplate("../../testdata/us/TEST-VB0-a")
	s, _ := session.NewWithTransport(f, tpl, catalog.DeviceEntry{})
	if _, err := s.Identify(context.Background()); err != nil {
		t.Fatalf("Identify: %v", err)
	}
	if s.SecureMode() {
		t.Error("missing register must default to OFF")
	}
}
