package catalog

import (
	"path/filepath"
	"testing"
)

func TestParseEnums(t *testing.T) {
	tpl, err := ParseTemplate(filepath.Join("..", "..", "testdata", "us", "TEST-VB0-a"))
	if err != nil {
		t.Fatalf("ParseTemplate: %v", err)
	}

	if got := len(tpl.Enums); got != 3 {
		t.Fatalf("got %d enums, want 3", got)
	}

	on := tpl.Enums["ON_OFF"]
	if on == nil {
		t.Fatal("missing ON_OFF")
	}
	if v, err := on.LabelFor(1); err != nil || v != "ON" {
		t.Errorf("LabelFor(1) = %q, %v", v, err)
	}
	if v, err := on.ValueFor("OFF"); err != nil || v != 0 {
		t.Errorf("ValueFor(OFF) = %d, %v", v, err)
	}
	if _, err := on.LabelFor(99); err == nil {
		t.Errorf("LabelFor(99) should error")
	}

	g := tpl.Enums["Gst61850_Msg"]
	if g == nil {
		t.Fatal("missing Gst61850_Msg")
	}
	want := map[int]string{0: "WriteCid", 3: "RestartDevice", 7: "GetIedName"}
	for v, lbl := range want {
		got, err := g.LabelFor(v)
		if err != nil || got != lbl {
			t.Errorf("LabelFor(%d) = %q, %v", v, got, err)
		}
	}
}
