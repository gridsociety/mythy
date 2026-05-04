package configio_test

import (
	"context"
	"testing"

	"github.com/gridsociety/mythy/pkg/configio"
	"github.com/gridsociety/mythy/pkg/transport"
)

func TestDiffNoChanges(t *testing.T) {
	f := transport.NewFake()
	f.OnReadHolding(6145, 1, []uint16{5})
	f.OnReadHolding(15362, 1, []uint16{3}) // MB_baudrate = "9600 baud"
	s := mkSession(t, f)

	cf := &configio.ConfigFile{
		MythyVersion: 1,
		Device:       configio.Device{Product: "TEST-VX0-a"},
		Settings: map[string]any{
			"MB_address":  int64(5),
			"MB_baudrate": "9600 baud",
		},
	}
	changes, err := configio.Diff(context.Background(), s, cf)
	if err != nil {
		t.Fatalf("Diff: %v", err)
	}
	if len(changes) != 0 {
		t.Errorf("expected no changes; got %+v", changes)
	}
}

func TestDiffOneChange(t *testing.T) {
	f := transport.NewFake()
	f.OnReadHolding(6145, 1, []uint16{1}) // MB_address = 1
	s := mkSession(t, f)

	cf := &configio.ConfigFile{
		MythyVersion: 1,
		Device:       configio.Device{Product: "TEST-VX0-a"},
		Settings:     map[string]any{"MB_address": int64(5)},
	}
	changes, err := configio.Diff(context.Background(), s, cf)
	if err != nil {
		t.Fatalf("Diff: %v", err)
	}
	if len(changes) != 1 {
		t.Fatalf("got %d changes, want 1", len(changes))
	}
	if changes[0].Name != "MB_address" || changes[0].Current != int64(1) || changes[0].File != int64(5) {
		t.Errorf("Change = %+v", changes[0])
	}
}
