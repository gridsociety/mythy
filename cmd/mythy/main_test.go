package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRootCommandHelp(t *testing.T) {
	cmd := newRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("--help should not error: %v", err)
	}
	got := out.String()
	for _, want := range []string{"mythy", "Thytronic"} {
		if !strings.Contains(got, want) {
			t.Errorf("help output missing %q\nGot:\n%s", want, got)
		}
	}
}

func TestRootCommandVersion(t *testing.T) {
	cmd := newRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--version"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("--version should not error: %v", err)
	}
	if !strings.Contains(out.String(), "mythy version") {
		t.Errorf("expected version output, got %q", out.String())
	}
}
