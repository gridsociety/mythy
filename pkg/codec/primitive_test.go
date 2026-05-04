package codec

import (
	"testing"
)

func TestDecodeUWORD(t *testing.T) {
	if got, err := DecodeUWORD([]uint16{0x1234}); err != nil || got != 4660 {
		t.Errorf("DecodeUWORD = %d, %v", got, err)
	}
	if _, err := DecodeUWORD(nil); err == nil {
		t.Error("DecodeUWORD(nil) should error")
	}
	if _, err := DecodeUWORD([]uint16{1, 2}); err == nil {
		t.Error("DecodeUWORD with 2 regs should error")
	}
}

func TestDecodeWORD(t *testing.T) {
	if got, _ := DecodeWORD([]uint16{0xFFFF}); got != -1 {
		t.Errorf("DecodeWORD(0xFFFF) = %d, want -1", got)
	}
	if got, _ := DecodeWORD([]uint16{0x7FFF}); got != 32767 {
		t.Errorf("DecodeWORD(0x7FFF) = %d, want 32767", got)
	}
}

func TestDecodeULONG(t *testing.T) {
	// SerialNumber from the discovery capture: bytes 86 a0 00 01.
	// Low-word-first: registers are 0x86A0 (low), 0x0005 (high),
	// reassembled as (high<<16)|low ⇒ 0x000186A0 = 100000.
	if got, _ := DecodeULONG([]uint16{0x86A0, 0x0001}); got != 100000 {
		t.Errorf("DecodeULONG = %d, want 100000", got)
	}
	if _, err := DecodeULONG([]uint16{1, 2, 3}); err == nil {
		t.Error("DecodeULONG with 3 regs should error")
	}
}

func TestDecodeSTRING(t *testing.T) {
	// "AB"  packed into one register as 0x4142 (high byte 'A', low byte 'B').
	regs := []uint16{0x4142, 0x4344, 0x0000}
	if got, _ := DecodeSTRING(regs); got != "ABCD" {
		t.Errorf("DecodeSTRING = %q, want %q", got, "ABCD")
	}
	if got, _ := DecodeSTRING([]uint16{0x4100}); got != "A" {
		t.Errorf("DecodeSTRING NUL-trim = %q, want %q", got, "A")
	}
}
