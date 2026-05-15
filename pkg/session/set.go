package session

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/gridsociety/mythy/pkg/catalog"
	"github.com/gridsociety/mythy/pkg/codec"
)

// Set writes a single value, opening (and committing) an edit transaction
// if the target is a *_PARAM register. *_RAM targets are written directly.
func (s *Session) Set(ctx context.Context, name string, value any) error {
	return s.SetMany(ctx, map[string]any{name: value})
}

// SetMany writes several values atomically — opens one edit transaction,
// performs all *_PARAM writes inside it, and commits. *_RAM writes are
// performed outside the transaction.
//
// Order is deterministic (alphabetical by key) for reproducible
// transcripts in tests.
func (s *Session) SetMany(ctx context.Context, pairs map[string]any) error {
	type plan struct {
		d     *catalog.Data
		regs  []uint16
		isRAM bool
	}
	plans := make([]plan, 0, len(pairs))
	keys := make([]string, 0, len(pairs))
	for k := range pairs {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		d, _, err := s.findData(k)
		if err != nil {
			return err
		}
		if d.Message == nil {
			return fmt.Errorf("DATA %q has no <message> link", k)
		}
		regs, err := s.encodeForWrite(d, pairs[k])
		if err != nil {
			return err
		}
		isRAM := strings.HasSuffix(d.Message.Class, "_RAM")
		plans = append(plans, plan{d: d, regs: regs, isRAM: isRAM})
	}

	// Split into two phases: RAM (no txn) and PARAM (one txn for all).
	var ramPlans, paramPlans []plan
	for _, p := range plans {
		if p.isRAM {
			ramPlans = append(ramPlans, p)
		} else {
			paramPlans = append(paramPlans, p)
		}
	}

	for _, p := range ramPlans {
		if err := s.writeRegs(ctx, p.d, p.regs); err != nil {
			return err
		}
	}
	if len(paramPlans) == 0 {
		return nil
	}

	tx, err := s.BeginEdit(ctx)
	if err != nil {
		return err
	}
	for _, p := range paramPlans {
		if err := s.writeRegs(ctx, p.d, p.regs); err != nil {
			_ = tx.Close(ctx)
			return err
		}
	}
	return tx.Commit(ctx)
}

// writeRegs picks FC06 (single reg) or FC16 (multi reg) based on length.
func (s *Session) writeRegs(ctx context.Context, d *catalog.Data, regs []uint16) error {
	addr := uint16(d.Message.WireAddr())
	if len(regs) == 1 {
		return s.mapErr(s.t.WriteSingleRegister(ctx, addr, regs[0]))
	}
	return s.mapErr(s.t.WriteMultipleRegisters(ctx, addr, regs))
}

// encodeForWrite converts a user value to the register representation,
// validating bounds / enum membership / string length before sending.
//
// Audit I6: when the catalog has a <DATA><RANGE> declaration with
// numeric bounds, that takes precedence over the type-width fallback.
// e.g. MB_address has TIPO=UBYTE (0..255) but RANGE=0,247,1 → reject
// 248..255 even though they fit the type.
func (s *Session) encodeForWrite(d *catalog.Data, value any) ([]uint16, error) {
	// Validate against catalog <RANGE> first if present.
	if d.Range != nil {
		if err := validateAgainstRange(value, d); err != nil {
			return nil, err
		}
	}
	switch d.Tipo {
	case "UBYTE":
		u, err := asUint(value, 0xFF)
		if err != nil {
			return nil, fmt.Errorf("set %s: %w", d.Name, err)
		}
		return []uint16{uint16(u)}, nil
	case "BYTE":
		i, err := asInt(value, -128, 127)
		if err != nil {
			return nil, fmt.Errorf("set %s: %w", d.Name, err)
		}
		return []uint16{uint16(int16(i)) & 0xFF}, nil
	case "UWORD", "BIT16":
		u, err := asUint(value, 0xFFFF)
		if err != nil {
			return nil, fmt.Errorf("set %s: %w", d.Name, err)
		}
		return []uint16{uint16(u)}, nil
	case "WORD":
		i, err := asInt(value, -32768, 32767)
		if err != nil {
			return nil, fmt.Errorf("set %s: %w", d.Name, err)
		}
		return []uint16{uint16(int16(i))}, nil
	case "ULONG", "BIT32":
		u, err := asUint(value, 0xFFFFFFFF)
		if err != nil {
			return nil, fmt.Errorf("set %s: %w", d.Name, err)
		}
		return []uint16{uint16(u & 0xFFFF), uint16((u >> 16) & 0xFFFF)}, nil
	case "LONG":
		i, err := asInt(value, -1<<31, (1<<31)-1)
		if err != nil {
			return nil, fmt.Errorf("set %s: %w", d.Name, err)
		}
		u := uint64(uint32(int32(i)))
		return []uint16{uint16(u & 0xFFFF), uint16((u >> 16) & 0xFFFF)}, nil
	case "STRING":
		str, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("set %s: STRING needs string, got %T", d.Name, value)
		}
		maxLen := d.Message.Dim * 2
		if len(str) > maxLen {
			return nil, fmt.Errorf("set %s: string %q exceeds %d-char limit", d.Name, str, maxLen)
		}
		return encodeStringRegs(str, d.Message.Dim), nil
	case "ENUM", "ENUM_BYTE", "ENUM_LONG":
		num, err := s.resolveEnumWriteValue(d, value)
		if err != nil {
			return nil, err
		}
		if d.Tipo == "ENUM_LONG" {
			u := uint64(num)
			return []uint16{uint16(u & 0xFFFF), uint16((u >> 16) & 0xFFFF)}, nil
		}
		return []uint16{uint16(num)}, nil
	}
	// Compound: treat value as map[string]any
	if cls, ok := s.tpl.Classes[d.Tipo]; ok {
		m, ok := value.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("set %s: compound TIPO=%s needs map[string]any, got %T", d.Name, d.Tipo, value)
		}
		return codec.EncodeCompound(m, cls, s.tpl.Enums)
	}
	// ARRAY (and any other unmodelled TIPO that arrives as a raw
	// register slice from configio.YAMLToCodec): pass through after
	// validating the length matches the catalog dim.
	if regs, ok := value.([]uint16); ok {
		if len(regs) != d.Message.Dim {
			return nil, fmt.Errorf("set %s: TIPO=%s expects %d regs, got %d", d.Name, d.Tipo, d.Message.Dim, len(regs))
		}
		return regs, nil
	}
	return nil, fmt.Errorf("set %s: unsupported TIPO %q (value type %T)", d.Name, d.Tipo, value)
}

func encodeStringRegs(str string, dim int) []uint16 {
	regs := make([]uint16, dim)
	for i := 0; i < len(str); i += 2 {
		hi := byte(str[i])
		var lo byte
		if i+1 < len(str) {
			lo = byte(str[i+1])
		}
		regs[i/2] = uint16(hi)<<8 | uint16(lo)
	}
	return regs
}

func (s *Session) resolveEnumWriteValue(d *catalog.Data, value any) (int, error) {
	if d.Enum == "" {
		return asAnyInt(value)
	}
	e := s.tpl.Enums[d.Enum]
	if e == nil {
		return asAnyInt(value)
	}
	if str, ok := value.(string); ok {
		n, err := e.ValueFor(str)
		if err == nil {
			return n, nil
		}
		return 0, fmt.Errorf("set %s: %q not a member of enum %s", d.Name, str, d.Enum)
	}
	return asAnyInt(value)
}

func asUint(value any, maxVal uint64) (uint64, error) {
	switch v := value.(type) {
	case uint64:
		if v > maxVal {
			return 0, fmt.Errorf("value %d exceeds limit %d", v, maxVal)
		}
		return v, nil
	case int64:
		if v < 0 || uint64(v) > maxVal {
			return 0, fmt.Errorf("value %d out of [0,%d]", v, maxVal)
		}
		return uint64(v), nil
	case int:
		return asUint(int64(v), maxVal)
	}
	return 0, fmt.Errorf("unsupported value type %T", value)
}

func asInt(value any, lo, hi int64) (int64, error) {
	var i int64
	switch v := value.(type) {
	case int64:
		i = v
	case uint64:
		i = int64(v)
	case int:
		i = int64(v)
	default:
		return 0, fmt.Errorf("unsupported value type %T", value)
	}
	if i < lo || i > hi {
		return 0, fmt.Errorf("value %d out of [%d,%d]", i, lo, hi)
	}
	return i, nil
}

func asAnyInt(value any) (int, error) {
	switch v := value.(type) {
	case int:
		return v, nil
	case int64:
		return int(v), nil
	case uint64:
		return int(v), nil
	}
	return 0, fmt.Errorf("expected integer, got %T", value)
}

// validateAgainstRange enforces the catalog's <DATA><RANGE> bounds
// before the value is encoded. Numeric ranges only — STRING / ENUM
// types validate elsewhere. Audit I6.
func validateAgainstRange(value any, d *catalog.Data) error {
	// Skip non-numeric TIPOs; the RANGE on STRING DATA is for character
	// count, not a comma-triple, so DataRange is nil there anyway.
	switch d.Tipo {
	case "STRING", "ENUM", "ENUM_BYTE", "ENUM_LONG":
		return nil
	}
	// Multi-band DATA (e.g. VLineaPrimario_1): the catalog declares
	// several <RANGE> children describing disjoint bands with
	// different step sizes. Accept the value if it matches any band.
	ranges := d.Ranges
	if len(ranges) == 0 && d.Range != nil {
		ranges = []*catalog.DataRange{d.Range}
	}
	if len(ranges) == 0 {
		return nil
	}
	n, err := asAnyInt(value)
	if err != nil {
		return fmt.Errorf("set %s: %w", d.Name, err)
	}
	v := int64(n)
	for _, r := range ranges {
		if v < r.Min || v > r.Max {
			continue
		}
		if r.Step > 1 && (v-r.Min)%r.Step != 0 {
			continue
		}
		return nil
	}
	// Build a helpful error message listing all bands.
	if len(ranges) == 1 {
		r := ranges[0]
		if v < r.Min || v > r.Max {
			return fmt.Errorf("set %s: %d out of catalog range [%d, %d]", d.Name, n, r.Min, r.Max)
		}
		return fmt.Errorf("set %s: %d violates step=%d (offsets from %d allowed)", d.Name, n, r.Step, r.Min)
	}
	parts := make([]string, 0, len(ranges))
	for _, r := range ranges {
		parts = append(parts, fmt.Sprintf("[%d,%d step %d]", r.Min, r.Max, r.Step))
	}
	return fmt.Errorf("set %s: %d not in any of catalog bands %s", d.Name, n, strings.Join(parts, " "))
}
