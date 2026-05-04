package main

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateOK(t *testing.T) {
	root := testdataRoot(t)
	out, err := runMythy(t,
		"--templates", root,
		"--device", "TEST-VX0-a",
		"validate", filepath.Join(root, "valid.yaml"))
	if err != nil {
		t.Fatalf("validate: %v\n%s", err, out)
	}
	if !strings.Contains(out, "OK") {
		t.Errorf("expected 'OK':\n%s", out)
	}
}

func TestValidateUnknownKey(t *testing.T) {
	root := testdataRoot(t)
	_, err := runMythy(t,
		"--templates", root,
		"--device", "TEST-VX0-a",
		"validate", filepath.Join(root, "unknown-key.yaml"))
	if err == nil {
		t.Error("expected error for unknown key TotallyMadeUp")
	}
}

func TestValidateProductMismatch(t *testing.T) {
	root := testdataRoot(t)
	_, err := runMythy(t,
		"--templates", root,
		"--device", "TEST-VX0-a",
		"validate", filepath.Join(root, "wrong-product.yaml"))
	if err == nil {
		t.Error("expected error for mismatched product")
	}
}
