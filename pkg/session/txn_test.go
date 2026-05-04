package session_test

import (
	"context"
	"testing"

	"github.com/gridsociety/mythy/pkg/transport"
)

func TestBeginEditCommit(t *testing.T) {
	f := transport.NewFake()
	f.OnReadInput(0x1402, 1, []uint16{1}) // START_CHANGE_DB
	f.OnReadInput(0x1403, 1, []uint16{1}) // END_CHANGE_DB
	s := mkSession(t, f)

	tx, err := s.BeginEdit(context.Background())
	if err != nil {
		t.Fatalf("BeginEdit: %v", err)
	}
	if err := tx.Commit(context.Background()); err != nil {
		t.Fatalf("Commit: %v", err)
	}
	// Closing after Commit must be a no-op.
	if err := tx.Close(context.Background()); err != nil {
		t.Fatalf("Close after Commit: %v", err)
	}
}

func TestBeginEditRollbackOnClose(t *testing.T) {
	f := transport.NewFake()
	f.OnReadInput(0x1402, 1, []uint16{1}) // START_CHANGE_DB
	f.OnReadInput(0x142C, 1, []uint16{1}) // RESET_CHANGE_DB
	s := mkSession(t, f)

	tx, err := s.BeginEdit(context.Background())
	if err != nil {
		t.Fatalf("BeginEdit: %v", err)
	}
	if err := tx.Close(context.Background()); err != nil {
		t.Fatalf("Close: %v", err)
	}
}

func TestBeginEditTwiceFails(t *testing.T) {
	f := transport.NewFake()
	f.OnReadInput(0x1402, 1, []uint16{1})
	f.OnReadInput(0x142C, 1, []uint16{1})
	s := mkSession(t, f)

	tx1, err := s.BeginEdit(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.BeginEdit(context.Background()); err == nil {
		t.Error("nested BeginEdit must error")
	}
	_ = tx1.Close(context.Background())
}
