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
