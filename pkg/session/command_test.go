package session_test

import (
	"context"
	"testing"

	"github.com/gridsociety/mythy/pkg/transport"
)

func TestCommandTrigger(t *testing.T) {
	f := transport.NewFake()
	// MSG_CMD_RESET_DA_PC at wire 5200, FC04 trigger
	f.OnReadInput(5200, 1, []uint16{1})
	s := mkSession(t, f)

	if err := s.Command(context.Background(), "MSG_CMD_RESET_DA_PC", nil); err != nil {
		t.Fatalf("Command: %v", err)
	}
}

func TestCommandUnknownErrors(t *testing.T) {
	f := transport.NewFake()
	s := mkSession(t, f)
	if err := s.Command(context.Background(), "NotARealCommand", nil); err == nil {
		t.Error("expected error for unknown command")
	}
}

func TestCommandTriggerNonOneFails(t *testing.T) {
	f := transport.NewFake()
	f.OnReadInput(5200, 1, []uint16{0}) // device returned 0, not 1
	s := mkSession(t, f)
	if err := s.Command(context.Background(), "MSG_CMD_RESET_DA_PC", nil); err == nil {
		t.Error("expected error when trigger returns non-1")
	}
}
