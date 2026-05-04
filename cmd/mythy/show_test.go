package main

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func runMythy(t *testing.T, args ...string) (string, error) {
	t.Helper()
	cmd := newRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), err
}

func testdataRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", "..", "testdata"))
	if err != nil {
		t.Fatal(err)
	}
	return root
}

func TestShowTopLevel(t *testing.T) {
	out, err := runMythy(t,
		"--templates", testdataRoot(t),
		"--device", "TEST-VX0-a",
		"show")
	if err != nil {
		t.Fatalf("show: %v\n%s", err, out)
	}
	for _, want := range []string{"Read", "Set", "Communication", "Commands"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q in output:\n%s", want, out)
		}
	}
	// Administrator group is hidden by default
	if strings.Contains(out, "Administrator") {
		t.Errorf("Administrator should be hidden by default:\n%s", out)
	}
}

func TestShowPath(t *testing.T) {
	out, err := runMythy(t,
		"--templates", testdataRoot(t),
		"--device", "TEST-VX0-a",
		"show", "Set/Base")
	if err != nil {
		t.Fatalf("show Set/Base: %v\n%s", err, out)
	}
	if !strings.Contains(out, "MB_address") {
		t.Errorf("expected MB_address in output:\n%s", out)
	}
}

func TestShowIncludeHidden(t *testing.T) {
	out, _ := runMythy(t,
		"--templates", testdataRoot(t),
		"--device", "TEST-VX0-a",
		"show", "--include-hidden")
	if !strings.Contains(out, "Administrator") {
		t.Errorf("expected Administrator with --include-hidden:\n%s", out)
	}
}
