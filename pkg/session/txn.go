package session

import (
	"context"
	"fmt"
)

const (
	addrStartChangeDB uint16 = 0x1402 // num 5123
	addrEndChangeDB   uint16 = 0x1403 // num 5124
	addrResetChangeDB uint16 = 0x142C // num 5165
)

// Edit is the RAII handle for an open edit transaction (§ 2.7).
//
// BeginEdit issues START_CHANGE_DB. Commit issues END_CHANGE_DB.
// Close, if called before Commit, issues RESET_CHANGE_DB to roll back.
//
// Only one Edit may be open per Session at a time.
type Edit struct {
	s         *Session
	committed bool
	closed    bool
}

// BeginEdit opens a new edit transaction.
func (s *Session) BeginEdit(ctx context.Context) (*Edit, error) {
	s.mu.Lock()
	if s.txnOpen {
		s.mu.Unlock()
		return nil, fmt.Errorf("session: edit transaction already open")
	}
	s.mu.Unlock()

	regs, err := s.t.ReadInputRegisters(ctx, addrStartChangeDB, 1)
	if err != nil {
		return nil, fmt.Errorf("START_CHANGE_DB: %w", s.mapErr(err))
	}
	if len(regs) != 1 || regs[0] != 1 {
		return nil, fmt.Errorf("START_CHANGE_DB: device returned %v, want [1]", regs)
	}

	s.mu.Lock()
	s.txnOpen = true
	s.mu.Unlock()
	return &Edit{s: s}, nil
}

// Commit ends the edit transaction successfully.
func (e *Edit) Commit(ctx context.Context) error {
	if e.committed || e.closed {
		return nil
	}
	regs, err := e.s.t.ReadInputRegisters(ctx, addrEndChangeDB, 1)
	if err != nil {
		return fmt.Errorf("END_CHANGE_DB: %w", e.s.mapErr(err))
	}
	if len(regs) != 1 || regs[0] != 1 {
		return fmt.Errorf("END_CHANGE_DB: device returned %v, want [1]", regs)
	}
	e.committed = true
	e.s.mu.Lock()
	e.s.txnOpen = false
	e.s.mu.Unlock()
	return nil
}

// Close rolls back if not already committed. Safe to call multiple times.
func (e *Edit) Close(ctx context.Context) error {
	if e.committed || e.closed {
		return nil
	}
	e.closed = true
	_, err := e.s.t.ReadInputRegisters(ctx, addrResetChangeDB, 1)
	e.s.mu.Lock()
	e.s.txnOpen = false
	e.s.mu.Unlock()
	if err != nil {
		return fmt.Errorf("RESET_CHANGE_DB: %w", e.s.mapErr(err))
	}
	return nil
}
