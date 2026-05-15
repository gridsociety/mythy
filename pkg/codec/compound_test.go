package codec_test

import (
	"path/filepath"
	"reflect"
	"testing"

	"github.com/gridsociety/mythy/pkg/catalog"
	"github.com/gridsociety/mythy/pkg/codec"
)

func loadTpl(t *testing.T) *catalog.Template {
	t.Helper()
	tpl, err := catalog.ParseTemplate(filepath.Join("..", "..", "testdata", "us", "TEST-VB0-a"))
	if err != nil {
		t.Fatalf("ParseTemplate: %v", err)
	}
	return tpl
}

func TestDecodeContatore(t *testing.T) {
	tpl := loadTpl(t)
	cls, ok := tpl.Classes["CONTATORE"]
	if !ok {
		t.Fatal("CONTATORE class missing from fixture (Plan 1 must already parse <CLASS>)")
	}
	// regs from the live snapshot: Tipo=1 (low-word-first ENUM_LONG), MaxValore=9999
	regs := []uint16{0x0001, 0x0000, 0x270F}
	got, err := codec.DecodeCompound(regs, cls, tpl.Enums)
	if err != nil {
		t.Fatalf("DecodeCompound: %v", err)
	}
	if got["Tipo"] != int64(1) {
		t.Errorf("Tipo = %v, want 1", got["Tipo"])
	}
	if got["MaxValore"] != uint64(9999) {
		t.Errorf("MaxValore = %v, want 9999", got["MaxValore"])
	}
}

func TestDecodeTimer(t *testing.T) {
	tpl := loadTpl(t)
	cls := tpl.Classes["TIMER"]
	regs := []uint16{0x01F4, 0x0000, 0x0000, 0x0000} // Valore=500, Correzione=0
	got, err := codec.DecodeCompound(regs, cls, tpl.Enums)
	if err != nil {
		t.Fatalf("DecodeCompound: %v", err)
	}
	if got["Valore"] != uint64(500) {
		t.Errorf("Valore = %v", got["Valore"])
	}
	if got["Correzione"] != int64(0) {
		t.Errorf("Correzione = %v", got["Correzione"])
	}
}

func TestEncodeCompoundAcceptsIntForEnumSubField(t *testing.T) {
	// Regression for #3: YAML decoders typically produce `int` for small
	// integers like `Tipo: 1`. enumNum used to handle only int64/uint64,
	// so an int value silently became 0 — flipping a SOGLIA from MASSIMA
	// to MINIMA, or a RELE_PARAM from ECCITATO to DISECCITATO, on import.
	tpl := loadTpl(t)
	cls := tpl.Classes["CONTATORE"] // Tipo (ENUM_LONG) + MaxValore (UWORD)

	cases := []struct {
		name string
		tipo any
	}{
		{"int", int(1)},
		{"int64", int64(1)},
		{"uint64", uint64(1)},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			in := map[string]any{"Tipo": c.tipo, "MaxValore": uint64(9999)}
			regs, err := codec.EncodeCompound(in, cls, tpl.Enums)
			if err != nil {
				t.Fatalf("EncodeCompound: %v", err)
			}
			// ENUM_LONG is low-word-first; Tipo=1 → reg[0]=0x0001, reg[1]=0x0000.
			// UWORD MaxValore=9999 → reg[2]=0x270F.
			want := []uint16{0x0001, 0x0000, 0x270F}
			if !reflect.DeepEqual(regs, want) {
				t.Errorf("Tipo=%v(%T): regs = %v, want %v", c.tipo, c.tipo, regs, want)
			}
		})
	}
}

func TestEncodeDecodeRoundTrip(t *testing.T) {
	tpl := loadTpl(t)
	cls := tpl.Classes["TIMER"]
	in := map[string]any{"Valore": uint64(1500), "Correzione": int64(-3)}
	regs, err := codec.EncodeCompound(in, cls, tpl.Enums)
	if err != nil {
		t.Fatalf("EncodeCompound: %v", err)
	}
	got, err := codec.DecodeCompound(regs, cls, tpl.Enums)
	if err != nil {
		t.Fatalf("DecodeCompound: %v", err)
	}
	if !reflect.DeepEqual(got, in) {
		t.Errorf("round trip lost data: got %+v, want %+v", got, in)
	}
}
