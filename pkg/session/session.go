package session

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/gridsociety/mythy/pkg/catalog"
	"github.com/gridsociety/mythy/pkg/transport"
)

// Session is the live-device handle the CLI consumes.
//
// One Session = one open Modbus connection + one loaded device template.
type Session struct {
	t       transport.Transport
	tpl     *catalog.Template
	entry   catalog.DeviceEntry
	ident   *Identification // populated by Identify (Task 10)
	secMode bool            // populated by Identify (Task 10); read of ENABLE_SEC_MODE

	mu      sync.Mutex
	txnOpen bool

	// Retry policy for data reads (audit C8 / SPEC § 3.0). Defaults to
	// 2 retries with 200ms backoff; set via SetRetryPolicy. Writes and
	// trigger-register reads NEVER retry — duplicate writes can corrupt
	// state and duplicate triggers (END_CHANGE_DB twice, RestartDevice
	// twice) are unsafe.
	retries int
	backoff time.Duration
}

// NewWithTransport builds a Session from an already-open Transport plus a
// pre-loaded template. Useful for tests that pass a fake; production code
// goes through Connect (Task 10) which composes Open + Identify + load.
func NewWithTransport(t transport.Transport, tpl *catalog.Template, entry catalog.DeviceEntry) (*Session, error) {
	if t == nil {
		return nil, fmt.Errorf("session: nil transport")
	}
	return &Session{
		t:       t,
		tpl:     tpl,
		entry:   entry,
		retries: 2,
		backoff: 200 * time.Millisecond,
	}, nil
}

// SetRetryPolicy configures the retry behavior for data reads.
// retries=0 disables retries entirely. backoff is the sleep between
// attempts. Writes and trigger-register reads never retry regardless.
// Audit C8 / SPEC § 3.0 / brainstorming E6.
func (s *Session) SetRetryPolicy(retries int, backoff time.Duration) {
	s.retries = retries
	s.backoff = backoff
}

// retryRead invokes fn up to (1 + s.retries) times, sleeping s.backoff
// between attempts. Only retries on transient errors (timeouts, EOF,
// connection-reset). Modbus exceptions are device decisions and never
// retry.
func (s *Session) retryRead(ctx context.Context, fn func() ([]uint16, error)) ([]uint16, error) {
	var lastErr error
	for attempt := 0; attempt <= s.retries; attempt++ {
		regs, err := fn()
		if err == nil {
			return regs, nil
		}
		lastErr = err
		if !transport.IsTransient(err) {
			return nil, err
		}
		if attempt == s.retries {
			break
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(s.backoff):
		}
	}
	return nil, lastErr
}

// Close tears down the underlying transport. Idempotent.
func (s *Session) Close() error {
	if s.t == nil {
		return nil
	}
	err := s.t.Close()
	s.t = nil
	return err
}

// Template returns the loaded device template.
func (s *Session) Template() *catalog.Template { return s.tpl }

// Entry returns the Codifica entry for the device.
func (s *Session) Entry() catalog.DeviceEntry { return s.entry }

// SecureMode reports whether the device's ENABLE_SEC_MODE register
// reads ON. Populated by Identify; defaults to false until then.
func (s *Session) SecureMode() bool { return s.secMode }

// txnState exposes the current transaction flag without leaking the lock.
func (s *Session) txnState() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.txnOpen
}

// mapErr is a Session method (not a free function) so callers don't have
// to thread (txnOpen, secMode) themselves.
func (s *Session) mapErr(err error) error {
	return MapException(err, s.txnState(), s.secMode)
}

// WriteMultiRaw issues FC16 directly. Used by command wrappers that
// need to drive WREG payloads outside SetMany's catalog-typed path.
func (s *Session) WriteMultiRaw(ctx context.Context, addr uint16, regs []uint16) error {
	return s.mapErr(s.t.WriteMultipleRegisters(ctx, addr, regs))
}

// ReadInputRaw / ReadHoldingRaw / WriteSingleRaw expose the underlying
// transport directly for the `mythy raw` escape hatch. They still go
// through mapErr so typed errors are produced.
func (s *Session) ReadInputRaw(ctx context.Context, addr, qty uint16) ([]uint16, error) {
	regs, err := s.t.ReadInputRegisters(ctx, addr, qty)
	return regs, s.mapErr(err)
}
func (s *Session) ReadHoldingRaw(ctx context.Context, addr, qty uint16) ([]uint16, error) {
	regs, err := s.t.ReadHoldingRegisters(ctx, addr, qty)
	return regs, s.mapErr(err)
}
func (s *Session) WriteSingleRaw(ctx context.Context, addr, value uint16) error {
	return s.mapErr(s.t.WriteSingleRegister(ctx, addr, value))
}
