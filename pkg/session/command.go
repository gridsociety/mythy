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
		// Parameterized command — write the payload, no edit txn (CMD-style
		// WREGs the device serves are RAM-class).
		// Plan 2 supports the no-arg case explicitly; structured arg encoding
		// lands in Plan 3 alongside compound encoding for export/import.
		if len(args) > 0 {
			return fmt.Errorf("command %s takes parameters; structured args land in Plan 3", name)
		}
		if msg.Dim == 1 {
			return s.mapErr(s.t.WriteSingleRegister(ctx, uint16(msg.WireAddr()), 0))
		}
		regs := make([]uint16, msg.Dim)
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
