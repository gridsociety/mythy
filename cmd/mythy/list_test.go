package main

import (
	"strings"
	"testing"
)

func TestListAll(t *testing.T) {
	out, err := runMythy(t,
		"--templates", testdataRoot(t),
		"--device", "TEST-VX0-a",
		"list")
	if err != nil {
		t.Fatalf("list: %v\n%s", err, out)
	}
	for _, want := range []string{"MB_address", "NomeLinea", "MB_baudrate"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q in output:\n%s", want, out)
		}
	}
}

func TestListScope(t *testing.T) {
	out, err := runMythy(t,
		"--templates", testdataRoot(t),
		"--device", "TEST-VX0-a",
		"list", "--scope", "Set/Base")
	if err != nil {
		t.Fatalf("list scope: %v\n%s", err, out)
	}
	if !strings.Contains(out, "MB_address") {
		t.Errorf("expected MB_address:\n%s", out)
	}
	if strings.Contains(out, "MB_baudrate") {
		t.Errorf("MB_baudrate should be out of scope:\n%s", out)
	}
}
