package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestResolveFormatPrecedence(t *testing.T) {
	// --format wins over --yaml
	f := &formatFlags{format: "json", asYAML: true}
	if got := f.resolve(); got != formatJSON {
		t.Errorf("--format wins: got %q, want json", got)
	}
	// --yaml wins over env
	t.Setenv("MYTHY_FORMAT", "json")
	f2 := &formatFlags{asYAML: true}
	if got := f2.resolve(); got != formatYAML {
		t.Errorf("--yaml beats env: got %q, want yaml", got)
	}
	// env wins when nothing else
	t.Setenv("MYTHY_FORMAT", "json")
	f3 := &formatFlags{}
	if got := f3.resolve(); got != formatJSON {
		t.Errorf("env: got %q, want json", got)
	}
	// default human
	t.Setenv("MYTHY_FORMAT", "")
	f4 := &formatFlags{}
	if got := f4.resolve(); got != formatHuman {
		t.Errorf("default: got %q, want human", got)
	}
}

func TestRenderStructJSON(t *testing.T) {
	var buf bytes.Buffer
	type sample struct{ A int }
	if err := renderStruct(&buf, formatJSON, sample{A: 1}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), `"A": 1`) {
		t.Errorf("unexpected json:\n%s", buf.String())
	}
}
