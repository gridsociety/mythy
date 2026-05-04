package configio_test

import (
	"context"
	"testing"

	"github.com/gridsociety/mythy/pkg/configio"
	"github.com/gridsociety/mythy/pkg/transport"
)

func TestApplyWritesOnlyChanged(t *testing.T) {
	f := transport.NewFake()
	// Pre-read for diff
	f.OnReadHolding(6145, 1, []uint16{1})  // MB_address current = 1
	f.OnReadHolding(15362, 1, []uint16{3}) // MB_baudrate current = 3 ("9600 baud")
	// Edit txn
	f.OnReadInput(0x1402, 1, []uint16{1}) // START_CHANGE_DB
	f.OnWriteSingleOK(6145)               // MB_address ← 5
	f.OnReadInput(0x1403, 1, []uint16{1}) // END_CHANGE_DB
	s := mkSession(t, f)

	cf := &configio.ConfigFile{
		MythyVersion: 1,
		Device:       configio.Device{Product: "TEST-VX0-a"},
		Settings: map[string]any{
			"MB_address":  int64(5),
			"MB_baudrate": "9600 baud", // unchanged
		},
	}
	report, err := configio.Apply(context.Background(), s, cf, configio.ApplyOptions{})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if len(report.Applied) != 1 || report.Applied[0] != "MB_address" {
		t.Errorf("Applied = %v, want [MB_address]", report.Applied)
	}
	if len(f.Writes) != 1 || f.Writes[0].Addr != 6145 || f.Writes[0].Values[0] != 5 {
		t.Errorf("expected exactly one FC06 write of MB_address=5, got %+v", f.Writes)
	}
}

func TestApplySkipsReadOnly(t *testing.T) {
	// NomeLinea is READONLY=YES in the fixture. A YAML that disagrees
	// must NOT trigger a write — Apply puts the name in Report.Skipped
	// and never asks the transport to write it. Without this guard a
	// hand-edited --include-readonly export would FC16 a register the
	// device may not even accept.
	f := transport.NewFake()
	// Diff reads NomeLinea to detect divergence (FC03 wire 6199, dim=10).
	// "SAMPLE" packed: 0x5341 0x4D50 0x4C45 then NULs.
	f.OnReadHolding(6199, 10, append([]uint16{0x5341, 0x4D50, 0x4C45}, make([]uint16, 7)...))
	s := mkSession(t, f)

	cf := &configio.ConfigFile{
		MythyVersion: 1,
		Device:       configio.Device{Product: "TEST-VX0-a"},
		Settings: map[string]any{
			"NomeLinea": "DIFFERENT", // user edited a READONLY field
		},
	}
	report, err := configio.Apply(context.Background(), s, cf, configio.ApplyOptions{})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if len(report.Applied) != 0 {
		t.Errorf("Applied = %v, want []", report.Applied)
	}
	if len(report.Skipped) != 1 || report.Skipped[0] != "NomeLinea" {
		t.Errorf("Skipped = %v, want [NomeLinea]", report.Skipped)
	}
	if len(f.Writes) != 0 {
		t.Errorf("READONLY skip must not write; got %+v", f.Writes)
	}
}

func TestApplyDryRunNoWrites(t *testing.T) {
	f := transport.NewFake()
	f.OnReadHolding(6145, 1, []uint16{1})
	s := mkSession(t, f)

	cf := &configio.ConfigFile{
		MythyVersion: 1,
		Device:       configio.Device{Product: "TEST-VX0-a"},
		Settings:     map[string]any{"MB_address": int64(5)},
	}
	report, err := configio.ApplyDryRun(context.Background(), s, cf, configio.ApplyOptions{})
	if err != nil {
		t.Fatalf("ApplyDryRun: %v", err)
	}
	if len(report.WouldApply) != 1 {
		t.Errorf("WouldApply = %v", report.WouldApply)
	}
	if len(f.Writes) != 0 {
		t.Errorf("dry-run must not write; got %+v", f.Writes)
	}
}
