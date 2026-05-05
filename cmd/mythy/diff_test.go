package main

import (
	"strings"
	"testing"
)

func TestDiffCommandRequiresConnection(t *testing.T) {
	out, err := runMythy(t, "diff", "/tmp/nope.yaml")
	if err == nil {
		t.Errorf("diff must error without --host/--serial; got %s", out)
	}
}

func TestRenderDiffTable(t *testing.T) {
	rows := []renderableChange{
		{Name: "MB_address", Path: "Set/Base", Current: int64(1), File: int64(5)},
	}
	out := renderDiffTable(rows)
	for _, want := range []string{"MB_address", "Set/Base", "1", "5"} {
		if !strings.Contains(out, want) {
			t.Errorf("table output missing %q:\n%s", want, out)
		}
	}
}

func TestDiffCommandAllFlagRegistered(t *testing.T) {
	// --all must exist on the diff command and default to false.
	cmd := newDiffCmd(&catalogFlags{})
	flag := cmd.Flags().Lookup("all")
	if flag == nil {
		t.Fatal("--all flag not registered on diff command")
	}
	if flag.Value.String() != "false" {
		t.Errorf("--all default = %q, want \"false\"", flag.Value.String())
	}
}
