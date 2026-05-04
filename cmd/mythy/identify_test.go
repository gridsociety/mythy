package main

import (
	"strings"
	"testing"
)

// runMythy is defined in show_test.go (Plan 1).

func TestIdentifyCommandShape(t *testing.T) {
	// Without --host/--serial the command must error; this is purely a
	// smoke test that the command is wired into the root.
	out, err := runMythy(t, "identify")
	if err == nil {
		t.Errorf("expected error without --host/--serial; got %s", out)
	}
	if !strings.Contains(out, "host") {
		t.Errorf("error output should mention --host; got %s", out)
	}
}
