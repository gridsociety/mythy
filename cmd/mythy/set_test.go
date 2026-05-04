package main

import (
	"reflect"
	"testing"
)

func TestParseSetArgs(t *testing.T) {
	got, err := parseSetArgs([]string{`MB_address=5`, `NomeLinea="SAMPLE"`, `MB_baudrate=19200 baud`})
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]any{
		"MB_address":  uint64(5),
		"NomeLinea":   "SAMPLE",
		"MB_baudrate": "19200 baud",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestParseSetArgsBadFormat(t *testing.T) {
	if _, err := parseSetArgs([]string{`bad`}); err == nil {
		t.Error("expected error for non-key=value arg")
	}
}
