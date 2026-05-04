package session_test

import (
	"context"
	"errors"
	"testing"

	"github.com/gridsociety/mythy/pkg/catalog"
	"github.com/gridsociety/mythy/pkg/session"
	"github.com/gridsociety/mythy/pkg/transport"
)

func TestNewSessionRequiresTransport(t *testing.T) {
	_, err := session.NewWithTransport(nil, nil, catalog.DeviceEntry{})
	if err == nil {
		t.Error("expected error with nil transport")
	}
}

func TestSessionCloseIsIdempotent(t *testing.T) {
	f := transport.NewFake()
	s, err := session.NewWithTransport(f, &catalog.Template{}, catalog.DeviceEntry{})
	if err != nil {
		t.Fatalf("NewWithTransport: %v", err)
	}
	if err := s.Close(); err != nil {
		t.Fatalf("first Close: %v", err)
	}
	if err := s.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}
}

func TestErrTransactionNotOpen(t *testing.T) {
	exc := &transport.ModbusException{FC: 6, Code: 0x04}
	mapped := session.MapException(exc, false /* txnOpen */, false /* secMode */)
	var target *session.ErrTransactionNotOpen
	if !errors.As(mapped, &target) {
		t.Errorf("expected ErrTransactionNotOpen, got %T %v", mapped, mapped)
	}
}

func TestErrAccessLevelRequired(t *testing.T) {
	exc := &transport.ModbusException{FC: 6, Code: 0x04}
	mapped := session.MapException(exc, true /* txnOpen */, true /* secMode */)
	var target *session.ErrAccessLevelRequired
	if !errors.As(mapped, &target) {
		t.Errorf("expected ErrAccessLevelRequired, got %T %v", mapped, mapped)
	}
}

func TestPassThroughNonException(t *testing.T) {
	plain := errors.New("connection reset")
	mapped := session.MapException(plain, false, false)
	if !errors.Is(mapped, plain) {
		t.Errorf("non-exception errors must pass through; got %v", mapped)
	}
}

var _ = context.Background // keep import; used in later tasks
