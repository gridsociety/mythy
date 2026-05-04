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
	report, err := configio.Apply(context.Background(), s, cf)
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

func TestApplyDryRunNoWrites(t *testing.T) {
	f := transport.NewFake()
	f.OnReadHolding(6145, 1, []uint16{1})
	s := mkSession(t, f)

	cf := &configio.ConfigFile{
		MythyVersion: 1,
		Device:       configio.Device{Product: "TEST-VX0-a"},
		Settings:     map[string]any{"MB_address": int64(5)},
	}
	report, err := configio.ApplyDryRun(context.Background(), s, cf)
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
