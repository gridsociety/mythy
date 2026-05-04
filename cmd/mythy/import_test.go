package main

import (
	"strings"
	"testing"
)

func TestImportRequiresConnection(t *testing.T) {
	out, err := runMythy(t, "import", "/tmp/nope.yaml")
	if err == nil {
		t.Errorf("import must error without --host/--serial; got %s", out)
	}
}

func TestImportRequiresArg(t *testing.T) {
	out, err := runMythy(t, "import")
	if err == nil {
		t.Errorf("import requires a file argument; got %s", out)
	}
	if !strings.Contains(out, "arg") && !strings.Contains(out, "argument") {
		t.Errorf("error message must mention argument; got %s", out)
	}
}
