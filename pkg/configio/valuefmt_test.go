package configio_test

import (
	"reflect"
	"testing"

	"github.com/gridsociety/mythy/pkg/configio"
	"github.com/gridsociety/mythy/pkg/session"
)

func TestValueToYAMLNumeric(t *testing.T) {
	v := session.Value{Tipo: "UWORD", Number: 42}
	if got := configio.ValueToYAML(v); got != int64(42) {
		t.Errorf("got %v (%T), want int64(42)", got, got)
	}
}

func TestValueToYAMLString(t *testing.T) {
	v := session.Value{Tipo: "STRING", Str: "SAMPLE"}
	if got := configio.ValueToYAML(v); got != "SAMPLE" {
		t.Errorf("got %v, want %q", got, "SAMPLE")
	}
}

func TestValueToYAMLEnumLabel(t *testing.T) {
	v := session.Value{Tipo: "ENUM", Label: "9600 baud"}
	if got := configio.ValueToYAML(v); got != "9600 baud" {
		t.Errorf("got %v, want label", got)
	}
}

func TestValueToYAMLOnOffAsBool(t *testing.T) {
	on := session.Value{Tipo: "ENUM", Label: "ON", EnumName: "ON_OFF"}
	off := session.Value{Tipo: "ENUM", Label: "OFF", EnumName: "ON_OFF"}
	if got := configio.ValueToYAML(on); got != true {
		t.Errorf("ON → %v, want true", got)
	}
	if got := configio.ValueToYAML(off); got != false {
		t.Errorf("OFF → %v, want false", got)
	}
	// A non-ON_OFF enum that happens to have an "ON" label stays as a string.
	other := session.Value{Tipo: "ENUM", Label: "ON", EnumName: "F59N_EnumDa74VT"}
	if got := configio.ValueToYAML(other); got != "ON" {
		t.Errorf("non-ON_OFF ENUM with Label ON → %v, want %q", got, "ON")
	}
}

func TestValueToYAMLArrayFromRaw(t *testing.T) {
	// ARRAY (SPEC § 2.10) has no fixed width — render every register's
	// bytes (high then low) as uppercase hex. Example: a 3-reg MAC.
	v := session.Value{Tipo: "ARRAY", Raw: []uint16{0x0011, 0x2233, 0xAABB}}
	if got := configio.ValueToYAML(v); got != "00112233AABB" {
		t.Errorf("got %v, want %q", got, "00112233AABB")
	}
}

func TestValueToYAMLUnknownTipoFromRaw(t *testing.T) {
	// Forward-compatibility: a TIPO mythy doesn't model individually
	// arrives with Raw populated; render as hex rather than erroring.
	v := session.Value{Tipo: "FUTURE_TIPO", Raw: []uint16{0xDEAD, 0xBEEF}}
	if got := configio.ValueToYAML(v); got != "DEADBEEF" {
		t.Errorf("got %v, want %q", got, "DEADBEEF")
	}
}

func TestValueToYAMLCompound(t *testing.T) {
	v := session.Value{
		Tipo: "TIMER",
		Compound: map[string]session.Value{
			"Valore":     {Tipo: "ULONG", Number: 500},
			"Correzione": {Tipo: "LONG", Number: 0},
		},
	}
	got := configio.ValueToYAML(v)
	want := map[string]any{"Valore": int64(500), "Correzione": int64(0)}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestYAMLToCodec_RoundTripNumeric(t *testing.T) {
	in := int64(42)
	out, err := configio.YAMLToCodec("UWORD", "", in)
	if err != nil {
		t.Fatal(err)
	}
	if out != uint64(42) {
		t.Errorf("got %v (%T), want uint64(42)", out, out)
	}
}

func TestYAMLToCodec_BoolAsOnOff(t *testing.T) {
	out, err := configio.YAMLToCodec("ENUM", "ON_OFF", true)
	if err != nil {
		t.Fatal(err)
	}
	if out != "ON" {
		t.Errorf("got %v, want \"ON\"", out)
	}
}

func TestYAMLToCodec_StringPassthrough(t *testing.T) {
	out, err := configio.YAMLToCodec("ENUM", "EnumBaudrate", "9600 baud")
	if err != nil {
		t.Fatal(err)
	}
	if out != "9600 baud" {
		t.Errorf("got %v", out)
	}
}

func TestYAMLToCodec_ArrayHexRoundTrip(t *testing.T) {
	// "00112233AABB" → 3 regs (high-byte-first per Modbus): 0x0011, 0x2233, 0xAABB.
	out, err := configio.YAMLToCodec("ARRAY", "", "00112233AABB")
	if err != nil {
		t.Fatalf("ARRAY hex: %v", err)
	}
	regs, ok := out.([]uint16)
	if !ok {
		t.Fatalf("got %T %v, want []uint16", out, out)
	}
	want := []uint16{0x0011, 0x2233, 0xAABB}
	if len(regs) != len(want) {
		t.Fatalf("len = %d, want %d", len(regs), len(want))
	}
	for i, w := range want {
		if regs[i] != w {
			t.Errorf("regs[%d] = 0x%04X, want 0x%04X", i, regs[i], w)
		}
	}
}

func TestYAMLToCodec_ArrayRejectsOddHex(t *testing.T) {
	if _, err := configio.YAMLToCodec("ARRAY", "", "ABC"); err == nil {
		t.Error("3-char hex must fail (must be a multiple of 4)")
	}
	if _, err := configio.YAMLToCodec("ARRAY", "", "AB"); err == nil {
		t.Error("2-char hex must fail (less than one full register)")
	}
}
