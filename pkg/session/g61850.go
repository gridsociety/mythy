package session

import (
	"context"
	"fmt"
	"strings"

	"github.com/gridsociety/mythy/pkg/codec"
)

const (
	addrG61850Cmd     uint16 = 54587 - 1 // 0xD53A
	addrG61850Par1    uint16 = 51258 - 1 // 0xC839
	addrG61850Par2    uint16 = 51259 - 1 // 0xC83A
	addrG61850Reply   uint16 = 51257 - 1 // 0xC838
	addrG61850Trigger uint16 = 0x141B
	g61850ParRegs            = 50 // 100 chars / 2 chars per reg
)

// G61850 invokes a parser function.
//
// For Get* functions it returns the ASCII reply read from
// GST61850_CMD_REPLY. For Set* functions it returns "" — the device
// does not write to REPLY for Set* (verified, § 2.8.2).
//
// Parameter validation: numeric Set* functions (SetReportScanRate,
// SetBrcbBufSize) are checked client-side; the device accepts garbage
// strings otherwise.
//
// Destructive function safety: WriteCid / ResetCid / ResetAll are NOT
// refused at the session layer (audit C5). The CLI is the right place
// to gate — see cmd/mythy/g61850.go which requires --force and (when
// stdin is a TTY) an interactive confirm. Calling Session.G61850
// directly from Go bypasses those gates by design.
func (s *Session) G61850(ctx context.Context, function, par1, par2 string) (string, error) {
	enum, ok := s.tpl.Enums["Gst61850_Msg"]
	if !ok {
		return "", fmt.Errorf("device %q has no G61850 parser (no Gst61850_Msg enum in template)", s.entry.Product)
	}
	// Audit D4 / I10: also gate on the trigger command's presence.
	if _, ok := s.tpl.Messages["MSG_GST61850"]; !ok {
		return "", fmt.Errorf("device %q has no MSG_GST61850 trigger; G61850 parser unreachable", s.entry.Product)
	}
	num, err := enum.ValueFor(function)
	if err != nil {
		return "", fmt.Errorf("g61850: %w", err)
	}

	if err := validateNumericSet(function, par1); err != nil {
		return "", err
	}

	if par1 != "" {
		regs := encodeStringRegs(par1, g61850ParRegs)
		if err := s.t.WriteMultipleRegisters(ctx, addrG61850Par1, regs); err != nil {
			return "", s.mapErr(err)
		}
	}
	if par2 != "" {
		regs := encodeStringRegs(par2, g61850ParRegs)
		if err := s.t.WriteMultipleRegisters(ctx, addrG61850Par2, regs); err != nil {
			return "", s.mapErr(err)
		}
	}
	if err := s.t.WriteSingleRegister(ctx, addrG61850Cmd, uint16(num)); err != nil {
		return "", s.mapErr(err)
	}
	tr, err := s.t.ReadInputRegisters(ctx, addrG61850Trigger, 1)
	if err != nil {
		return "", s.mapErr(err)
	}
	if len(tr) != 1 || tr[0] != 1 {
		return "", fmt.Errorf("g61850 trigger returned %v, want [1]", tr)
	}
	if isGet(function) {
		regs, err := s.t.ReadInputRegisters(ctx, addrG61850Reply, g61850ParRegs)
		if err != nil {
			return "", s.mapErr(err)
		}
		return codec.DecodeSTRING(regs)
	}
	return "", nil
}

func isGet(name string) bool { return strings.HasPrefix(name, "Get") }

// IsDestructive reports whether the named G61850 function is one of
// the three known-destructive ones (WriteCid, ResetCid, ResetAll).
// Exported so the CLI layer can gate on --force (audit C5).
func IsDestructive(name string) bool {
	switch name {
	case "WriteCid", "ResetCid", "ResetAll":
		return true
	}
	return false
}

// validateNumericSet rejects non-integer parameters for the numeric
// Set* functions, since the device itself accepts garbage strings.
func validateNumericSet(name, par1 string) error {
	switch name {
	case "SetReportScanRate", "SetBrcbBufSize":
		if par1 == "" {
			return fmt.Errorf("g61850 %s requires a numeric --par1", name)
		}
		for _, c := range par1 {
			if c < '0' || c > '9' {
				return fmt.Errorf("g61850 %s --par1 must be an integer, got %q", name, par1)
			}
		}
	}
	return nil
}
