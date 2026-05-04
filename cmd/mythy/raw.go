package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/gridsociety/mythy/pkg/session"
)

// writeMultiByName looks up a WREG message by name and writes a register
// payload. Used by clock-set, net-set, etc. — commands that drive WREG
// payloads without going through SetMany.
//
// Audit C7: when the message's CLASS ends in "_PARAM" (persistent flash),
// the write is wrapped in a one-shot edit transaction (START_CHANGE_DB
// / FC16 / END_CHANGE_DB). *_RAM-class messages and unclassified
// trigger-style WREGs go through directly — same rule as SetMany
// (Plan 2 Task 15).
func writeMultiByName(ctx context.Context, s *session.Session, name string, regs []uint16) error {
	tpl := s.Template()
	msg, ok := tpl.Messages[name]
	if !ok {
		return fmt.Errorf("message %q not in catalog", name)
	}
	if msg.Type != "WREG" {
		return fmt.Errorf("message %q is %s, expected WREG", name, msg.Type)
	}
	if len(regs) != msg.Dim {
		return fmt.Errorf("%s expects dim=%d, got %d", name, msg.Dim, len(regs))
	}
	addr := uint16(msg.WireAddr())
	if strings.HasSuffix(msg.Class, "_PARAM") {
		tx, err := s.BeginEdit(ctx)
		if err != nil {
			return err
		}
		if err := s.WriteMultiRaw(ctx, addr, regs); err != nil {
			_ = tx.Close(ctx)
			return err
		}
		return tx.Commit(ctx)
	}
	return s.WriteMultiRaw(ctx, addr, regs)
}
