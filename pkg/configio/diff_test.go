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
	changes, err := configio.Diff(context.Background(), s, cf, configio.DiffOptions{})
	if err != nil {
		t.Fatalf("Diff: %v", err)
	}
	if len(changes) != 0 {
		t.Errorf("expected no changes; got %+v", changes)
	}
}

func TestDiffFiresProgressPerKey(t *testing.T) {
	// Diff issues one Modbus read per file key. Verify Progress fires
	// once before each read and once at the end with done==total and
	// name=="" so the CLI can clear its single-line progress UI.
	f := transport.NewFake()
	f.OnReadHolding(6145, 1, []uint16{5})
	s := mkSession(t, f)

	cf := &configio.ConfigFile{
		MythyVersion: 1,
		Device:       configio.Device{Product: "TEST-VX0-a"},
		Settings:     map[string]any{"MB_address": int64(5)},
	}
	type call struct {
		done, total int
		name        string
	}
	var calls []call
	if _, err := configio.Diff(context.Background(), s, cf, configio.DiffOptions{
		Progress: func(done, total int, name string) {
			calls = append(calls, call{done, total, name})
		},
	}); err != nil {
		t.Fatalf("Diff: %v", err)
	}
	if len(calls) != 2 {
		t.Fatalf("got %d progress calls, want 2: %+v", len(calls), calls)
	}
	if calls[0] != (call{0, 1, "MB_address"}) {
		t.Errorf("first call = %+v, want {0 1 MB_address}", calls[0])
	}
	if calls[1] != (call{1, 1, ""}) {
		t.Errorf("final call = %+v, want {1 1 \"\"}", calls[1])
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
	changes, err := configio.Diff(context.Background(), s, cf, configio.DiffOptions{})
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
