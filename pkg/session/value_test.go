package session_test

import (
	"strings"
	"testing"

	"github.com/gridsociety/mythy/pkg/session"
)

func TestValueFormatNumeric(t *testing.T) {
	v := session.Value{Tipo: "ULONG", Number: 49942, Unit: "Hz", Decimals: 3, Scale: 1000}
	if got, want := v.Format(), "49.942 Hz"; got != want {
		t.Errorf("Format() = %q, want %q", got, want)
	}
}

func TestValueFormatNoUnit(t *testing.T) {
	v := session.Value{Tipo: "UWORD", Number: 5}
	if got, want := v.Format(), "5"; got != want {
		t.Errorf("Format() = %q, want %q", got, want)
	}
}

func TestValueFormatEnum(t *testing.T) {
	v := session.Value{Tipo: "ENUM", Label: "ON"}
	if got, want := v.Format(), "ON"; got != want {
		t.Errorf("Format() = %q, want %q", got, want)
	}
}

func TestValueFormatString(t *testing.T) {
	v := session.Value{Tipo: "STRING", Str: "SAMPLE-IED"}
	if got, want := v.Format(), `"SAMPLE-IED"`; got != want {
		t.Errorf("Format() = %q, want %q", got, want)
	}
}

func TestValueFormatCompound(t *testing.T) {
	v := session.Value{
		Tipo: "TIMER",
		Compound: map[string]session.Value{
			"Valore":     {Tipo: "ULONG", Number: 500, Unit: "ms"},
			"Correzione": {Tipo: "LONG", Number: 0, Unit: "ms"},
		},
	}
	got := v.Format()
	for _, want := range []string{"Valore=500 ms", "Correzione=0 ms"} {
		if !strings.Contains(got, want) {
			t.Errorf("Format() = %q must contain %q", got, want)
		}
	}
}
