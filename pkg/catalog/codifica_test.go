package catalog

import (
	"errors"
	"path/filepath"
	"testing"
)

func TestLoadCodifica(t *testing.T) {
	idx, err := LoadCodifica(filepath.Join("..", "..", "testdata", "Codifica.xml"))
	if err != nil {
		t.Fatalf("LoadCodifica: %v", err)
	}

	if idx.Version != "2.14" {
		t.Errorf("Version = %q, want 2.14", idx.Version)
	}
	if got, want := len(idx.Devices), 3; got != want {
		t.Fatalf("Devices count = %d, want %d", got, want)
	}
}

func TestCodificaByIdentification(t *testing.T) {
	idx, err := LoadCodifica(filepath.Join("..", "..", "testdata", "Codifica.xml"))
	if err != nil {
		t.Fatalf("LoadCodifica: %v", err)
	}

	tests := []struct {
		ident       int
		wantProduct string
		wantTpl     string
		wantFamily  string
		wantIEC     bool
	}{
		{4660, "PROX-VX0-e", "PROX-VX0-e", "PROX", true},
		{27004, "PROX-VX0-d", "PROX-VX0-d", "PROX", true},
		{100, "NA016", "NA016-a", "NA016", false},
	}
	for _, tc := range tests {
		t.Run(tc.wantProduct, func(t *testing.T) {
			d, err := idx.ByIdentification(tc.ident)
			if err != nil {
				t.Fatalf("ByIdentification(%d): %v", tc.ident, err)
			}
			if d.Product != tc.wantProduct || d.Template != tc.wantTpl ||
				d.Family != tc.wantFamily || d.IEC61850 != tc.wantIEC {
				t.Errorf("got %+v", d)
			}
		})
	}
}

func TestCodificaByIdentificationMissing(t *testing.T) {
	idx, _ := LoadCodifica(filepath.Join("..", "..", "testdata", "Codifica.xml"))
	if _, err := idx.ByIdentification(99999); !errors.Is(err, ErrUnknownIdentification) {
		t.Errorf("want ErrUnknownIdentification, got %v", err)
	}
}
