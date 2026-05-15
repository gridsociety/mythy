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
	got, err := codec.DecodeCompound(regs, cls, tpl.Enums, nil)
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
	got, err := codec.DecodeCompound(regs, cls, tpl.Enums, nil)
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
			regs, err := codec.EncodeCompound(in, cls, tpl.Enums, nil)
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
	regs, err := codec.EncodeCompound(in, cls, tpl.Enums, nil)
	if err != nil {
		t.Fatalf("EncodeCompound: %v", err)
	}
	got, err := codec.DecodeCompound(regs, cls, tpl.Enums, nil)
	if err != nil {
		t.Fatalf("DecodeCompound: %v", err)
	}
	if !reflect.DeepEqual(got, in) {
		t.Errorf("round trip lost data: got %+v, want %+v", got, in)
	}
}

func TestEncodeCompoundHonoursTipoOverrideWideningEnumToEnumLong(t *testing.T) {
	// Regression for #6: <CLASS> declares sub-fields at one TIPO but the
	// per-instance <DATA TIPO="<class>"> can override TIPO via nested
	// <DATA NAME="..."> children. The most common case in current
	// Thytronic templates: SOGLIA has Stato/Tipo as ENUM (1 reg each) in
	// <CLASS>, but every SOGLIA instance overrides to ENUM_LONG (2 reg
	// each). Without honouring the override, the encoded payload is 12
	// regs instead of the 14 the wire region expects, and Pickup-value
	// writes land in the wrong slot.
	cls := &catalog.Class{
		Name: "UNDERFILLED",
		Dim:  3,
		Vars: []catalog.ClassVar{
			{Name: "State", Tipo: "ENUM"},   // 1 reg per CLASS, 2 reg via override
			{Name: "Pickup", Tipo: "UWORD"}, // 1 reg unchanged
		},
	}
	overrides := map[string]*catalog.CompoundFieldOverride{
		"State": {Tipo: "ENUM_LONG"},
	}

	in := map[string]any{"State": int64(1), "Pickup": uint64(42)}
	regs, err := codec.EncodeCompound(in, cls, nil, overrides)
	if err != nil {
		t.Fatalf("EncodeCompound: %v", err)
	}
	// State ENUM_LONG=1 low-word-first → regs[0]=0x0001, regs[1]=0x0000.
	// Pickup UWORD=42 → regs[2]=0x002A.
	want := []uint16{0x0001, 0x0000, 0x002A}
	if !reflect.DeepEqual(regs, want) {
		t.Errorf("regs = %v, want %v (override should widen State to ENUM_LONG)", regs, want)
	}

	// Without the override, the encoder would produce only 2 regs —
	// confirm that's the pre-fix state we're regressing against.
	bare, err := codec.EncodeCompound(in, cls, nil, nil)
	if err != nil {
		t.Fatalf("EncodeCompound (no overrides): %v", err)
	}
	if len(bare) != 2 {
		t.Errorf("without overrides expected 2 regs (class-level widths), got %d: %v", len(bare), bare)
	}
}

func TestEncodeCompoundHandlesEnumWord(t *testing.T) {
	// Issue #9: ENUM_WORD is the 16-bit ENUM family member that was
	// missing from every TIPO switch. Verify codec now treats it as a
	// 1-reg sibling of ENUM and ENUM_BYTE.
	cls := &catalog.Class{
		Name: "WithEnumWord",
		Dim:  2,
		Vars: []catalog.ClassVar{
			{Name: "Idx", Tipo: "ENUM_WORD"},
			{Name: "Pad", Tipo: "UWORD"},
		},
	}
	in := map[string]any{"Idx": int64(5), "Pad": uint64(7)}
	regs, err := codec.EncodeCompound(in, cls, nil, nil)
	if err != nil {
		t.Fatalf("EncodeCompound: %v", err)
	}
	if want := []uint16{0x0005, 0x0007}; !reflect.DeepEqual(regs, want) {
		t.Errorf("regs = %v, want %v", regs, want)
	}
	back, err := codec.DecodeCompound(regs, cls, nil, nil)
	if err != nil {
		t.Fatalf("DecodeCompound: %v", err)
	}
	if back["Idx"] != int64(5) {
		t.Errorf("Idx round-trip = %v, want int64(5)", back["Idx"])
	}
}

func TestEncodeCompoundHandlesINT(t *testing.T) {
	// Issue #10: legacy SIF/SVF templates use TIPO="INT" for 16-bit
	// signed integers. SIF5600.IdentificationResponse declares dim=5
	// with parts INT + LONG + INT + INT — so INT must be 1 reg
	// (1+2+1+1=5). Treat as a WORD synonym.
	cls := &catalog.Class{
		Name: "WithINT",
		Dim:  2,
		Vars: []catalog.ClassVar{
			{Name: "Year", Tipo: "INT"},  // 1 reg
			{Name: "Month", Tipo: "INT"}, // 1 reg
		},
	}
	in := map[string]any{"Year": int64(2026), "Month": int64(5)}
	regs, err := codec.EncodeCompound(in, cls, nil, nil)
	if err != nil {
		t.Fatalf("EncodeCompound: %v", err)
	}
	if want := []uint16{0x07EA, 0x0005}; !reflect.DeepEqual(regs, want) {
		t.Errorf("regs = %v, want %v", regs, want)
	}
	// Negative value round-trip — Thytronic INT is signed.
	in2 := map[string]any{"Year": int64(-1), "Month": int64(0)}
	regs2, err := codec.EncodeCompound(in2, cls, nil, nil)
	if err != nil {
		t.Fatalf("EncodeCompound: %v", err)
	}
	if regs2[0] != 0xFFFF {
		t.Errorf("negative INT encoding: regs[0] = 0x%04X, want 0xFFFF", regs2[0])
	}
	back, err := codec.DecodeCompound(regs2, cls, nil, nil)
	if err != nil {
		t.Fatalf("DecodeCompound: %v", err)
	}
	if back["Year"] != int64(-1) {
		t.Errorf("Year round-trip = %v, want int64(-1)", back["Year"])
	}
}

func TestDecodeCompoundHonoursTipoOverride(t *testing.T) {
	cls := &catalog.Class{
		Name: "UNDERFILLED",
		Dim:  3,
		Vars: []catalog.ClassVar{
			{Name: "State", Tipo: "ENUM"},
			{Name: "Pickup", Tipo: "UWORD"},
		},
	}
	overrides := map[string]*catalog.CompoundFieldOverride{
		"State": {Tipo: "ENUM_LONG"},
	}
	regs := []uint16{0x0001, 0x0000, 0x002A}
	got, err := codec.DecodeCompound(regs, cls, nil, overrides)
	if err != nil {
		t.Fatalf("DecodeCompound: %v", err)
	}
	// ENUM_LONG decoder routes through "ENUM_LONG" arm: int64.
	if got["State"] != int64(1) {
		t.Errorf("State = %v, want int64(1)", got["State"])
	}
	if got["Pickup"] != uint64(42) {
		t.Errorf("Pickup = %v, want uint64(42)", got["Pickup"])
	}
}
