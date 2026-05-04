package main

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/gridsociety/mythy/pkg/catalog"
	"github.com/gridsociety/mythy/pkg/session"
	"github.com/gridsociety/mythy/pkg/transport"
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

func mkExpandSession(t *testing.T, f *transport.Fake) *session.Session {
	t.Helper()
	tpl, err := catalog.ParseTemplate("../../testdata/us/TEST-VB0-a")
	if err != nil {
		t.Fatal(err)
	}
	s, err := session.NewWithTransport(f, tpl, catalog.DeviceEntry{Product: "TEST-VX0-a"})
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestExpandCompoundMutationsPreservesSiblings(t *testing.T) {
	// EnF81_TSc is TIMER (Valore:ULONG, Correzione:LONG), wire 40019, dim=4.
	// Current value: Valore=500, Correzione=0.
	f := transport.NewFake()
	f.OnReadHolding(40019, 4, []uint16{0x01F4, 0x0000, 0x0000, 0x0000})
	s := mkExpandSession(t, f)

	pairs := map[string]any{"EnF81_TSc.Valore": uint64(2000)}
	expanded, err := expandCompoundMutations(context.Background(), s, pairs)
	if err != nil {
		t.Fatalf("expand: %v", err)
	}

	full, ok := expanded["EnF81_TSc"].(map[string]any)
	if !ok {
		t.Fatalf("EnF81_TSc = %T %v, want map", expanded["EnF81_TSc"], expanded["EnF81_TSc"])
	}
	if full["Valore"] != uint64(2000) {
		t.Errorf("Valore (mutated) = %v, want uint64(2000)", full["Valore"])
	}
	if full["Correzione"] != int64(0) {
		t.Errorf("Correzione (preserved) = %v, want int64(0)", full["Correzione"])
	}
}

func TestExpandCompoundMutationsCoalescesSameCompound(t *testing.T) {
	f := transport.NewFake()
	f.OnReadHolding(40019, 4, []uint16{0x01F4, 0x0000, 0x0000, 0x0000})
	s := mkExpandSession(t, f)

	// Two dotted args targeting the same compound bundle into one read.
	pairs := map[string]any{
		"EnF81_TSc.Valore":     uint64(2000),
		"EnF81_TSc.Correzione": int64(-3),
	}
	expanded, err := expandCompoundMutations(context.Background(), s, pairs)
	if err != nil {
		t.Fatalf("expand: %v", err)
	}
	if len(expanded) != 1 {
		t.Errorf("expected 1 compound entry, got %d: %+v", len(expanded), expanded)
	}
	full := expanded["EnF81_TSc"].(map[string]any)
	if full["Valore"] != uint64(2000) || full["Correzione"] != int64(-3) {
		t.Errorf("merged compound = %+v", full)
	}
}

func TestExpandCompoundMutationsScalarRejects(t *testing.T) {
	f := transport.NewFake()
	// MB_address is UBYTE (scalar) wire 6145.
	f.OnReadHolding(6145, 1, []uint16{1})
	s := mkExpandSession(t, f)

	pairs := map[string]any{"MB_address.Foo": uint64(5)}
	_, err := expandCompoundMutations(context.Background(), s, pairs)
	if err == nil || !strings.Contains(err.Error(), "not a compound type") {
		t.Errorf("expected 'not a compound type' error, got %v", err)
	}
}

func TestExpandCompoundMutationsUnknownSubField(t *testing.T) {
	f := transport.NewFake()
	f.OnReadHolding(40019, 4, []uint16{0x01F4, 0x0000, 0x0000, 0x0000})
	s := mkExpandSession(t, f)

	pairs := map[string]any{"EnF81_TSc.NotARealField": uint64(0)}
	_, err := expandCompoundMutations(context.Background(), s, pairs)
	if err == nil || !strings.Contains(err.Error(), "is not a sub-field of TIMER") {
		t.Errorf("expected unknown-sub-field error, got %v", err)
	}
}

func TestExpandCompoundMutationsPlainArgsPassThrough(t *testing.T) {
	f := transport.NewFake()
	s := mkExpandSession(t, f)

	pairs := map[string]any{"MB_address": uint64(5), "NomeLinea": "SAMPLE"}
	expanded, err := expandCompoundMutations(context.Background(), s, pairs)
	if err != nil {
		t.Fatalf("expand: %v", err)
	}
	if !reflect.DeepEqual(pairs, expanded) {
		t.Errorf("plain args must pass through unchanged: got %+v, want %+v", expanded, pairs)
	}
}
