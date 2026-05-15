package session

import (
	"context"
	"fmt"

	"github.com/gridsociety/mythy/pkg/catalog"
	"github.com/gridsociety/mythy/pkg/codec"
)

// Read looks up the named DATA leaf in the catalog, issues the matching
// Modbus read, and decodes the result.
func (s *Session) Read(ctx context.Context, name string) (Value, error) {
	d, _, err := s.findData(name)
	if err != nil {
		return Value{}, err
	}
	return s.readData(ctx, d)
}

func (s *Session) findData(name string) (*catalog.Data, *catalog.Group, error) {
	if s.tpl == nil || s.tpl.Menu == nil {
		return nil, nil, fmt.Errorf("session: no template loaded")
	}
	for _, g := range s.tpl.Menu.WalkGroups(catalog.WalkOptions{IncludeHidden: true}) {
		for _, d := range g.Data {
			if d.Name == name {
				return d, g, nil
			}
		}
	}
	return nil, nil, fmt.Errorf("not found in catalog: %q", name)
}

// readData issues the Modbus read for one DATA leaf and decodes it.
func (s *Session) readData(ctx context.Context, d *catalog.Data) (Value, error) {
	if d.Message == nil {
		return Value{}, fmt.Errorf("DATA %q has no <message> link", d.Name)
	}
	regs, err := s.readRegs(ctx, d.Message.FC(), uint16(d.Message.WireAddr()), uint16(d.Message.Dim))
	if err != nil {
		return Value{}, s.mapErr(err)
	}
	return s.decodeRegs(d, regs)
}

// readRegs is the read-with-retry helper. It is used for data reads
// (Read, ReadScope, Identify's IDENTIFICATION + ENABLE_SEC_MODE
// probes). Trigger-register reads (BeginEdit, Commit, Close,
// MSG_GST61850, GST61850_CMD_REPLY) call s.t.ReadInputRegisters
// directly so they never retry — duplicate triggers are unsafe.
func (s *Session) readRegs(ctx context.Context, fc int, addr, qty uint16) ([]uint16, error) {
	switch fc {
	case 4:
		return s.retryRead(ctx, func() ([]uint16, error) {
			return s.t.ReadInputRegisters(ctx, addr, qty)
		})
	case 3:
		return s.retryRead(ctx, func() ([]uint16, error) {
			return s.t.ReadHoldingRegisters(ctx, addr, qty)
		})
	}
	return nil, fmt.Errorf("session.readRegs: unsupported FC=%d", fc)
}

// decodeRegs converts a register slice into a Value, using the DATA's TIPO.
func (s *Session) decodeRegs(d *catalog.Data, regs []uint16) (Value, error) {
	v := Value{Tipo: d.Tipo}
	switch d.Tipo {
	case "BYTE":
		v.Number = int64(int8(regs[0] & 0xFF))
	case "UBYTE":
		v.Number = int64(regs[0] & 0xFF)
	case "WORD", "INT":
		i, _ := codec.DecodeWORD(regs)
		v.Number = int64(i)
	case "UWORD", "BIT16":
		u, _ := codec.DecodeUWORD(regs)
		v.Number = int64(u)
	case "LONG":
		i, _ := codec.DecodeLONG(regs)
		v.Number = int64(i)
	case "ULONG", "BIT32":
		u, _ := codec.DecodeULONG(regs)
		v.Number = int64(u)
	case "STRING":
		s, _ := codec.DecodeSTRING(regs)
		v.Str = s
	case "ENUM", "ENUM_BYTE", "ENUM_WORD", "ENUM_LONG":
		num := int(regs[0])
		if d.Tipo == "ENUM_LONG" {
			u, _ := codec.DecodeULONG(regs)
			num = int(u)
		}
		v.Number = int64(num)
		v.EnumName = d.Enum
		if d.Enum != "" {
			if e := s.tpl.Enums[d.Enum]; e != nil {
				if lbl, err := e.LabelFor(num); err == nil {
					v.Label = lbl
				}
			}
		}
	default:
		// Compound: walk the CLASS layout when the TIPO is registered.
		cls, ok := s.tpl.Classes[d.Tipo]
		if !ok {
			// ARRAY (SPEC § 2.10) and any other TIPO mythy doesn't model
			// individually are kept as opaque raw register bytes. Renderers
			// emit them as hex; round-trip writes are not supported in v1.
			v.Raw = append([]uint16(nil), regs...)
			return v, nil
		}
		fields, err := codec.DecodeCompound(regs, cls, s.tpl.Enums, d.CompoundOverrides)
		if err != nil {
			return v, err
		}
		// Audit I3: build sub-Value with each ClassVar's actual TIPO so
		// renderers/exporters know whether they're looking at a STRING,
		// an ENUM_BYTE, a BIT32, etc. Walking cls.Vars in order keeps
		// the output stable.
		v.Compound = make(map[string]Value, len(cls.Vars))
		for _, varDef := range cls.Vars {
			raw, ok := fields[varDef.Name]
			if !ok {
				continue
			}
			subVal := Value{Tipo: varDef.Tipo}
			switch x := raw.(type) {
			case int64:
				subVal.Number = x
			case uint64:
				subVal.Number = int64(x)
			case string:
				// ENUM* sub-fields decoded to label; STRING sub-fields
				// decoded to literal string. DecodeCompound returns
				// strings for both, but the right destination differs.
				if varDef.Tipo == "STRING" {
					subVal.Str = x
				} else {
					subVal.Label = x
				}
			}
			v.Compound[varDef.Name] = subVal
		}
	}
	return v, nil
}

// ReadScope walks the catalog under path and reads every kept leaf,
// returning a map keyed by DATA NAME. Hidden groups (VISIBILITY="3")
// are skipped unless includeHidden; module-disabled DATA is dropped
// (audit B6).
//
// Implementation note: each leaf is read with one Modbus request at
// the catalog-declared (addr, qty). Thytronic firmware enforces
// strict register-window boundaries — empirically (against a
// production XV10P) reads with qty != the declared dim, or at any
// non-declared offset, return Modbus exception 0x03 (illegal data
// value). Even reading a single register inside an INFO_MISURA block
// at +1 offset fails. Coalescing-merge optimisations (rospo's
// MergeRanges, SPEC § 5.3) are therefore unsafe for this device
// family. The 7,411-leaf full-device read takes ~30 s on LAN; that's
// the cost of correctness.
func (s *Session) ReadScope(ctx context.Context, path string, includeHidden bool) (map[string]Value, error) {
	if s.tpl == nil || s.tpl.Menu == nil {
		return nil, fmt.Errorf("session: no template loaded")
	}
	scope := s.tpl.Menu
	if path != "" {
		scope = scope.FindGroup(path)
		if scope == nil {
			return nil, fmt.Errorf("scope %q not found", path)
		}
	}
	leaves := scope.WalkData(catalog.WalkOptions{IncludeHidden: includeHidden, IncludeReadOnly: true})

	enabled, err := s.EnabledModules(ctx)
	if err != nil {
		return nil, err
	}

	out := make(map[string]Value, len(leaves))
	for _, d := range leaves {
		if d.Module != "" && enabled != nil && !enabled[d.Module] {
			continue
		}
		if d.Message == nil || d.Message.FC() == 0 {
			continue
		}
		v, err := s.readData(ctx, d)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", d.Name, err)
		}
		out[d.Name] = v
	}
	return out, nil
}
