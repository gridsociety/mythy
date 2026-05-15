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
	// Fallback path: template == nil (unknown variant), wire returns
	// exception 0x02 (illegal data address) — Identify must still
	// succeed and treat the device as secure-mode-off.
	f := transport.NewFake()
	f.OnReadInput(0x143E, 5, []uint16{0x1234, 0x0001, 0x0000, 0x0100, 0x0000})
	f.OnReadInputException(54295-1, 1, &transport.ModbusException{FC: 4, Code: 0x02})

	s, _ := session.NewWithTransport(f, nil, catalog.DeviceEntry{})
	if _, err := s.Identify(context.Background()); err != nil {
		t.Fatalf("Identify: %v", err)
	}
	if s.SecureMode() {
		t.Error("missing register must default to OFF")
	}
}

func TestIdentifyToleratesIllegalDataValueException(t *testing.T) {
	// Same fallback path as above, but the exception code is 0x03
	// (illegal data value). PRON-family firmware (e.g. NV10P) returns
	// 0x03 instead of 0x02 for a missing register — the NV10P bug
	// that motivated the catalog-first probe gating.
	f := transport.NewFake()
	f.OnReadInput(0x143E, 5, []uint16{0x1234, 0x0001, 0x0000, 0x0100, 0x0000})
	f.OnReadInputException(54295-1, 1, &transport.ModbusException{FC: 4, Code: 0x03})

	s, _ := session.NewWithTransport(f, nil, catalog.DeviceEntry{})
	if _, err := s.Identify(context.Background()); err != nil {
		t.Fatalf("Identify: %v", err)
	}
	if s.SecureMode() {
		t.Error("exception 0x03 must default to OFF")
	}
}

func TestIdentifySkipsProbeWhenTemplateLacksRegister(t *testing.T) {
	// Template loaded but ENABLE_SEC_MODE not declared (mirrors
	// NV10P-EB0-u, where ThyVisor reads nothing in 0xD4xx). The probe
	// must not hit the wire — the Fake transport panics on any
	// unscripted read, so the absence of OnReadInput for 0xD416 is the
	// load-bearing assertion.
	f := transport.NewFake()
	f.OnReadInput(0x143E, 5, []uint16{0x1234, 0x0001, 0x0000, 0x0100, 0x0000})

	tpl, err := catalog.ParseTemplate("../../testdata/us/TEST-VB0-a")
	if err != nil {
		t.Fatalf("ParseTemplate: %v", err)
	}
	delete(tpl.ByAddr, catalog.ByAddrKey{FC: 4, Addr: 54295 - 1})

	s, _ := session.NewWithTransport(f, tpl, catalog.DeviceEntry{})
	if _, err := s.Identify(context.Background()); err != nil {
		t.Fatalf("Identify: %v", err)
	}
	if s.SecureMode() {
		t.Error("template lacking ENABLE_SEC_MODE must default to OFF")
	}
}
