package session

import (
	"context"
	"fmt"

	"github.com/gridsociety/mythy/pkg/catalog"
	"github.com/gridsociety/mythy/pkg/codec"
	"github.com/gridsociety/mythy/pkg/transport"
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
	case "WORD":
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
	case "ENUM", "ENUM_BYTE", "ENUM_LONG":
		num := int(regs[0])
		if d.Tipo == "ENUM_LONG" {
			u, _ := codec.DecodeULONG(regs)
			num = int(u)
		}
		v.Number = int64(num)
		if d.Enum != "" {
			if e := s.tpl.Enums[d.Enum]; e != nil {
				if lbl, err := e.LabelFor(num); err == nil {
					v.Label = lbl
				}
			}
		}
	default:
		// Compound: walk the CLASS layout.
		cls, ok := s.tpl.Classes[d.Tipo]
		if !ok {
			return v, fmt.Errorf("readData(%s): unknown TIPO %q", d.Name, d.Tipo)
		}
		fields, err := codec.DecodeCompound(regs, cls, s.tpl.Enums)
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

// ReadScope walks the catalog under path, batch-reads every leaf,
// and returns a map keyed by DATA NAME. Hidden groups (VISIBILITY="3")
// are skipped unless includeHidden.
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

	plans := make([]transport.RangePlan, 0, len(leaves))
	type slot struct {
		d    *catalog.Data
		addr int
		qty  int
	}
	slots := make([]slot, 0, len(leaves))
	for _, d := range leaves {
		if d.Message == nil || d.Message.FC() == 0 {
			continue
		}
		fc := uint8(d.Message.FC())
		addr := uint16(d.Message.WireAddr())
		qty := uint16(d.Message.Dim)
		plans = append(plans, transport.RangePlan{FC: fc, StartAddr: addr, Count: qty})
		slots = append(slots, slot{d: d, addr: int(addr), qty: int(qty)})
	}
	merged := transport.MergeRanges(plans, transport.MergeOptions{})

	// Issue each batch and stash regs in a (FC, addr) → reg map for slicing.
	got := make(map[uint8]map[int]uint16)
	for _, b := range merged {
		regs, err := s.readRegs(ctx, int(b.FC), b.StartAddr, b.Count)
		if err != nil {
			return nil, s.mapErr(err)
		}
		fcMap, ok := got[b.FC]
		if !ok {
			fcMap = make(map[int]uint16, len(regs))
			got[b.FC] = fcMap
		}
		for i, r := range regs {
			fcMap[int(b.StartAddr)+i] = r
		}
	}

	// Slice each leaf's window out of the captured registers.
	out := make(map[string]Value, len(slots))
	for _, sl := range slots {
		fc := uint8(sl.d.Message.FC())
		regs := make([]uint16, sl.qty)
		for i := 0; i < sl.qty; i++ {
			regs[i] = got[fc][sl.addr+i]
		}
		v, err := s.decodeRegs(sl.d, regs)
		if err != nil {
			return nil, fmt.Errorf("decode %s: %w", sl.d.Name, err)
		}
		out[sl.d.Name] = v
	}
	return out, nil
}
