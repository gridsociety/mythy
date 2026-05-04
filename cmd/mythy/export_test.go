package main

import "testing"

func TestExportCommandRequiresConnection(t *testing.T) {
	out, err := runMythy(t, "export", "/tmp/should-not-write.yaml")
	if err == nil {
		t.Errorf("export must error without --host/--serial; got %s", out)
	}
}
