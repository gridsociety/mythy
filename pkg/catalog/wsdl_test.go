package catalog

import (
	"path/filepath"
	"testing"
)

func TestParseTemplateMessages(t *testing.T) {
	tpl, err := ParseTemplate(filepath.Join("..", "..", "testdata", "us", "TEST-VB0-a"))
	if err != nil {
		t.Fatalf("ParseTemplate: %v", err)
	}

	// Spot-check known entries from the fixture.
	cases := []struct {
		name    string
		fc      int
		wireAdr int
		dim     int
		typ     string
	}{
		{"IDENTIFICATION", 4, 5182, 5, "RREG"},
		{"START_CHANGE_DB", 4, 5122, 1, "RREG"},
		{"MSG_CMD_RESET_DA_PC", 4, 5200, 1, "RREG"},
		{"MB_address", 3, 6145, 1, "WREG"},
		{"NomeLinea", 3, 6199, 10, "WREG"},
		{"UL1", 4, 39999, 2, "RREG"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m, ok := tpl.Messages[tc.name]
			if !ok {
				t.Fatalf("missing message %s", tc.name)
			}
			if m.Type != tc.typ {
				t.Errorf("type = %q", m.Type)
			}
			if m.WireAddr() != tc.wireAdr {
				t.Errorf("wireAddr = %d, want %d", m.WireAddr(), tc.wireAdr)
			}
			if m.Dim != tc.dim {
				t.Errorf("dim = %d", m.Dim)
			}
			if m.FC() != tc.fc {
				t.Errorf("FC = %d", m.FC())
			}

			byAddr, ok := tpl.ByAddr[ByAddrKey{FC: tc.fc, Addr: tc.wireAdr}]
			if !ok {
				t.Fatalf("ByAddr lookup missing")
			}
			if byAddr != m {
				t.Errorf("ByAddr returned different message")
			}
		})
	}

	// Identification response should have its parts.
	resp, ok := tpl.Messages["IDENTIFICATIONResponse"]
	if !ok {
		t.Fatal("missing IDENTIFICATIONResponse")
	}
	wantParts := []struct{ name, typ string }{
		{"Identification", "WORD"},
		{"SerialNumber", "LONG"},
		{"FwRelease", "WORD"},
		{"ProtocolRelease", "WORD"},
	}
	if got := len(resp.Parts); got != len(wantParts) {
		t.Fatalf("parts = %d, want %d", got, len(wantParts))
	}
	for i, want := range wantParts {
		if resp.Parts[i].Name != want.name || resp.Parts[i].Type != want.typ {
			t.Errorf("part %d = %+v, want %+v", i, resp.Parts[i], want)
		}
	}
}
