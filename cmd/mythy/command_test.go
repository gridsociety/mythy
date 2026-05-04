package main

import (
	"strings"
	"testing"
)

func TestCommandList(t *testing.T) {
	out, err := runMythy(t,
		"--templates", testdataRoot(t),
		"--device", "TEST-VX0-a",
		"command", "list")
	if err != nil {
		t.Fatalf("command list: %v\n%s", err, out)
	}
	for _, want := range []string{"MSG_CMD_RESET_DA_PC", "Commands"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q:\n%s", want, out)
		}
	}
}

func TestG61850List(t *testing.T) {
	out, err := runMythy(t,
		"--templates", testdataRoot(t),
		"--device", "TEST-VX0-a",
		"g61850", "list")
	if err != nil {
		t.Fatalf("g61850 list: %v\n%s", err, out)
	}
	// Per audit D4: the template's Gst61850_Msg enum is authoritative.
	// The TEST template carries the enum (added in Task 5 fixture),
	// even though Codifica's IEC61850="false" — so the list IS shown.
	for _, want := range []string{"WriteCid", "RestartDevice", "GetIedName"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected enum entry %q in output:\n%s", want, out)
		}
	}
}
