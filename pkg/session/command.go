package session

import (
	"context"
	"fmt"

	"github.com/gridsociety/mythy/pkg/catalog"
)

// Command looks up the named <COMMAND> entry, optionally writes its
// parameters, and triggers it.
//
// Most commands are pure FC04-read triggers (MSG_CMD_RESET_DA_PC,
// MSG_CMD_AP_DA_PC, ...). A handful are WREG with structured input
// (SET_RTC, SET_PARAMS_ETH0). For the WREG case, args is a map keyed by
// <part name>; the encoder fills the multi-reg payload before triggering.
//
// Returns an error if the trigger response isn't 1.
func (s *Session) Command(ctx context.Context, name string, args map[string]any) error {
	cmd := s.findCommand(name)
	if cmd == nil {
		return fmt.Errorf("command not found in catalog: %q", name)
	}
	msg := cmd.Message
	if msg == nil {
		return fmt.Errorf("command %q has no <message> link", name)
	}

	switch msg.Type {
	case "RREG":
		// Pure trigger.
		regs, err := s.t.ReadInputRegisters(ctx, uint16(msg.WireAddr()), uint16(msg.Dim))
		if err != nil {
			return s.mapErr(err)
		}
		if len(regs) >= 1 && regs[0] != 1 {
			return fmt.Errorf("command %s returned %d, want 1", name, regs[0])
		}
		return nil
	case "WREG":
		// Build the multi-reg payload from the message's <part> list.
		regs := make([]uint16, 0, msg.Dim)
		for _, p := range msg.Parts {
			val, ok := args[p.Name]
			if !ok {
				val = nil // missing parts encode as zero
			}
			subRegs, err := encodePart(p.Type, val)
			if err != nil {
				return fmt.Errorf("command %s arg %s: %w", name, p.Name, err)
			}
			regs = append(regs, subRegs...)
		}
		// If the message has no parts (zero-arg WREG), pad with zeros.
		if len(regs) == 0 {
			regs = make([]uint16, msg.Dim)
		}
		if len(regs) != msg.Dim {
			return fmt.Errorf("command %s: encoded %d regs, message expects %d", name, len(regs), msg.Dim)
		}
		if msg.Dim == 1 {
			return s.mapErr(s.t.WriteSingleRegister(ctx, uint16(msg.WireAddr()), regs[0]))
		}
		return s.mapErr(s.t.WriteMultipleRegisters(ctx, uint16(msg.WireAddr()), regs))
	}
	return fmt.Errorf("command %s: unsupported message type %q", name, msg.Type)
}

// findCommand locates a COMMAND entry in any group of the menu tree
// (including hidden groups, since some COMMANDs live under Administrator).
func (s *Session) findCommand(name string) *catalog.Command {
	if s.tpl == nil || s.tpl.Menu == nil {
		return nil
	}
	for _, g := range s.tpl.Menu.WalkGroups(catalog.WalkOptions{IncludeHidden: true}) {
		for _, c := range g.Commands {
			if c.Name == name {
				return c
			}
		}
	}
	return nil
}

// encodePart converts one <part> value into its register slice.
// Width and encoding match the catalog primitive table (§ 2.10).
func encodePart(tipo string, val any) ([]uint16, error) {
	if val == nil {
		switch tipo {
		case "LONG", "ULONG", "BIT32":
			return []uint16{0, 0}, nil
		}
		return []uint16{0}, nil
	}
	switch tipo {
	case "BYTE", "UBYTE", "WORD", "UWORD", "BIT16":
		return []uint16{uint16(asUint64(val))}, nil
	case "LONG", "ULONG", "BIT32":
		u := asUint64(val)
		return []uint16{uint16(u & 0xFFFF), uint16((u >> 16) & 0xFFFF)}, nil
	case "STRING":
		s, ok := val.(string)
		if !ok {
			return nil, fmt.Errorf("STRING expects string, got %T", val)
		}
		out := make([]uint16, (len(s)+1)/2)
		for i := 0; i < len(s); i += 2 {
			hi := byte(s[i])
			var lo byte
			if i+1 < len(s) {
				lo = byte(s[i+1])
			}
			out[i/2] = uint16(hi)<<8 | uint16(lo)
		}
		return out, nil
	}
	return nil, fmt.Errorf("encodePart: unsupported TIPO %q", tipo)
}

func asUint64(v any) uint64 {
	switch x := v.(type) {
	case uint64:
		return x
	case int64:
		return uint64(x)
	case int:
		return uint64(x)
	}
	return 0
}
