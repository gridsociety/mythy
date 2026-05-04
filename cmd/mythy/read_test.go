package main

import (
	"strings"
	"testing"
)

func TestReadCommandShape(t *testing.T) {
	out, err := runMythy(t, "read", "MB_address")
	if err == nil {
		t.Errorf("expected error without --host/--serial; got %s", out)
	}
	if !strings.Contains(out, "host") && !strings.Contains(out, "serial") {
		t.Errorf("error must mention connection; got %s", out)
	}
}
