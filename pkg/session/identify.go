package session

import (
	"context"
	"errors"

	"github.com/gridsociety/mythy/pkg/catalog"
	"github.com/gridsociety/mythy/pkg/transport"
)

// Identification is the decoded payload of the FC04 discovery read at
// wire 0x143E (XML num 5183). See SPEC § 2.3.
type Identification struct {
	Identification  uint16
	SerialNumber    uint32
	FwRelease       uint16
	ProtocolRelease uint16
}

const (
	addrIdentification uint16 = 0x143E
	addrEnableSecMode  uint16 = 54295 - 1 // wire = num - 1
)

// Identify runs the discovery handshake and resolves the secure-mode
// state. It populates s.ident and s.secMode.
//
// Secure-mode resolution mirrors what ThyVisor does on the wire: if the
// loaded template (s.tpl) doesn't declare ENABLE_SEC_MODE for this
// device variant, we never probe — empirically verified against
// NV10P-EB0-u, where ThyVisor reads nothing in the 0xD4xx area. When
// the template doesn't load at all (unknown variant), we fall back to
// probing 0xD416 and treating illegal-address / illegal-data-value
// exceptions as "secure mode off".
func (s *Session) Identify(ctx context.Context) (*Identification, error) {
	// Audit I1: skip the IDENTIFICATION read if the caller has already
	// seeded the result via SeedIdentification (e.g. connFlags.build
	// reads the discovery once). Secure-mode resolution still runs.
	if s.ident == nil {
		regs, err := s.t.ReadInputRegisters(ctx, addrIdentification, 5)
		if err != nil {
			return nil, err
		}
		if len(regs) != 5 {
			return nil, errInvalid("identification: short response")
		}
		s.ident = &Identification{
			Identification:  regs[0],
			SerialNumber:    (uint32(regs[2]) << 16) | uint32(regs[1]), // low-word-first
			FwRelease:       regs[3],
			ProtocolRelease: regs[4],
		}
	}

	// Template-first: if the catalog template knows this device and
	// doesn't declare ENABLE_SEC_MODE, skip the probe entirely. This
	// matches ThyVisor and avoids a guaranteed-exception round-trip on
	// variants like NV10P-EB0-u.
	if s.tpl != nil {
		key := catalog.ByAddrKey{FC: 4, Addr: int(addrEnableSecMode)}
		if _, ok := s.tpl.ByAddr[key]; !ok {
			s.secMode = false
			return s.ident, nil
		}
	}

	// Secure-mode probe (always — we want fresh state).
	smRegs, err := s.t.ReadInputRegisters(ctx, addrEnableSecMode, 1)
	if err != nil {
		var exc *transport.ModbusException
		if errors.As(err, &exc) && (exc.Code == 0x02 || exc.Code == 0x03) {
			// Register doesn't exist on this firmware; secure mode is off.
			// Some devices (PROX) return 0x02 (illegal data address) for a
			// missing register; others (PRON/NV10P) return 0x03 (illegal
			// data value) for the same condition.
			s.secMode = false
		} else {
			return s.ident, err
		}
	} else if len(smRegs) == 1 && smRegs[0] != 0 {
		s.secMode = true
	}
	return s.ident, nil
}

// SeedIdentification populates s.ident from a value the caller already
// has (e.g. a discovery exchange done by connFlags.build). The next
// Identify call skips the IDENTIFICATION read but still resolves the
// secure-mode state. Audit I1.
func (s *Session) SeedIdentification(id *Identification) {
	s.ident = id
}

// Ident returns the most recent identification, or nil if Identify hasn't run.
func (s *Session) Ident() *Identification { return s.ident }

// errInvalid is a small helper used by various session methods.
type errInvalidStr struct{ s string }

func (e *errInvalidStr) Error() string { return e.s }

func errInvalid(s string) error { return &errInvalidStr{s} }
