// Package codec encodes and decodes Modbus register words to and from
// Go values, following the TIPO conventions used by Thytronic templates.
//
// Modbus register words are 16-bit big-endian. This package operates on
// []uint16 (already-decoded register values), not raw bytes. Endianness
// at the byte level is the transport's responsibility.
//
// Word order for multi-register types (LONG, ULONG): low word FIRST,
// high word SECOND. Verified against the captured IDENTIFICATION
// response (the spec § 2.3 / § 2.10): bytes 86 a0 00 01 ⇒ registers
// (0x86A0, 0x0001) ⇒ value 0x000186A0 = 100000 (the live device serial).
package codec

import (
	"errors"
	"fmt"
	"strings"
)

// ErrLength is returned when the register slice has the wrong length for the
// requested type.
var ErrLength = errors.New("wrong register count")

// DecodeUWORD decodes 1 register as an unsigned 16-bit value.
func DecodeUWORD(regs []uint16) (uint16, error) {
	if len(regs) != 1 {
		return 0, fmt.Errorf("%w: UWORD needs 1 reg, got %d", ErrLength, len(regs))
	}
	return regs[0], nil
}

// DecodeWORD decodes 1 register as a signed 16-bit value.
func DecodeWORD(regs []uint16) (int16, error) {
	v, err := DecodeUWORD(regs)
	return int16(v), err
}

// DecodeULONG decodes 2 registers as an unsigned 32-bit value
// using low-word-first order.
func DecodeULONG(regs []uint16) (uint32, error) {
	if len(regs) != 2 {
		return 0, fmt.Errorf("%w: ULONG needs 2 regs, got %d", ErrLength, len(regs))
	}
	return uint32(regs[1])<<16 | uint32(regs[0]), nil
}

// DecodeLONG decodes 2 registers as a signed 32-bit value
// using low-word-first order.
func DecodeLONG(regs []uint16) (int32, error) {
	v, err := DecodeULONG(regs)
	return int32(v), err
}

// DecodeSTRING decodes a string from its register-packed form: each register
// holds two ASCII bytes (high byte first), trailing NULs are trimmed.
func DecodeSTRING(regs []uint16) (string, error) {
	var b strings.Builder
	for _, r := range regs {
		hi := byte(r >> 8)
		lo := byte(r & 0xFF)
		if hi != 0 {
			b.WriteByte(hi)
		}
		if lo != 0 {
			b.WriteByte(lo)
		}
	}
	return b.String(), nil
}
