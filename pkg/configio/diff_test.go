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

func TestDiffFiltersRuntimeByDefault(t *testing.T) {
	// File mixes writable Set/, READONLY Set/, and READONLY Read/ entries.
	// Default filter (IncludeAll=false) must keep only the writable Set/
	// entry. Reads for the other two are NOT pre-scripted: if the filter
	// is broken and Diff tries to read them, the fake transport panics.
	f := transport.NewFake()
	f.OnReadHolding(6145, 1, []uint16{1}) // MB_address current=1, file=5 -> change
	s := mkSession(t, f)

	cf := &configio.ConfigFile{
		MythyVersion: 1,
		Device:       configio.Device{Product: "TEST-VX0-a"},
		Settings: map[string]any{
			"MB_address": int64(5), // Set/Base, writable -> kept
			"NomeLinea":  "SAMPLE", // Set/Base, READONLY  -> filtered
			"UL1":        int64(0), // Read/Measures, RO   -> filtered
		},
	}
	changes, err := configio.Diff(context.Background(), s, cf, configio.DiffOptions{})
	if err != nil {
		t.Fatalf("Diff: %v", err)
	}
	if len(changes) != 1 {
		t.Fatalf("got %d changes, want 1: %+v", len(changes), changes)
	}
	if changes[0].Name != "MB_address" {
		t.Errorf("unexpected change: %+v", changes[0])
	}
}

func TestDiffIncludeAllReadsEverything(t *testing.T) {
	// With IncludeAll=true the filter is off: every file key must hit
	// the device. We pre-script reads for all three (writable Set/,
	// READONLY Set/, READONLY Read/ scalar) and verify all three
	// surface as changes when the file values disagree.
	f := transport.NewFake()
	f.OnReadHolding(6145, 1, []uint16{1}) // MB_address current=1, file=5
	f.OnReadHolding(6199, 10, append([]uint16{0x4D49, 0x4C41, 0x4E4F}, make([]uint16, 7)...)) // NomeLinea current="MILANO", file="SAMPLE"
	f.OnReadInput(39999, 2, []uint16{0x0064, 0x0000}) // UL1 current=100 (low-word-first ULONG, RREG/FC04), file=0
	s := mkSession(t, f)

	cf := &configio.ConfigFile{
		MythyVersion: 1,
		Device:       configio.Device{Product: "TEST-VX0-a"},
		Settings: map[string]any{
			"MB_address": int64(5),
			"NomeLinea":  "SAMPLE",
			"UL1":        int64(0),
		},
	}
	changes, err := configio.Diff(context.Background(), s, cf,
		configio.DiffOptions{IncludeAll: true})
	if err != nil {
		t.Fatalf("Diff: %v", err)
	}
	if len(changes) != 3 {
		t.Fatalf("got %d changes, want 3: %+v", len(changes), changes)
	}
	names := map[string]bool{}
	for _, c := range changes {
		names[c.Name] = true
	}
	for _, want := range []string{"MB_address", "NomeLinea", "UL1"} {
		if !names[want] {
			t.Errorf("missing change for %s; got %v", want, names)
		}
	}
}

func TestDiffProgressTotalReflectsFilter(t *testing.T) {
	// Default-filter Diff over a 3-key file (1 kept, 2 filtered).
	// Progress total must reflect post-filter count (1), not file size.
	f := transport.NewFake()
	f.OnReadHolding(6145, 1, []uint16{5})
	s := mkSession(t, f)

	cf := &configio.ConfigFile{
		MythyVersion: 1,
		Device:       configio.Device{Product: "TEST-VX0-a"},
		Settings: map[string]any{
			"MB_address": int64(5),
			"NomeLinea":  "SAMPLE",
			"UL1":        int64(0),
		},
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
