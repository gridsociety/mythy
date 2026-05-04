package catalog

import (
	"errors"
	"path/filepath"
	"testing"
)

func TestSubstituteLocaleLetter(t *testing.T) {
	tests := []struct {
		template, locale, want string
		wantErr                error
	}{
		{"PROX-VX0-e", "en", "PROX-VB0-e", nil},
		{"PROX-VX0-e", "it", "PROX-VA0-e", nil},
		{"PROX-VX0-e", "es", "PROX-VS0-e", nil},
		{"PROX-VX0-e", "ru", "PROX-VR0-e", nil},
		{"PROX-VX0-e", "tr", "PROX-VT0-e", nil},
		{"NA016-a", "en", "NA016-a", nil},          // no X to replace — passthrough
		{"PROX-VX0-e", "de", "", ErrUnknownLocale}, // unsupported locale
	}
	for _, tc := range tests {
		t.Run(tc.template+"/"+tc.locale, func(t *testing.T) {
			got, err := substituteLocaleLetter(tc.template, tc.locale)
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("err = %v, want %v", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestResolveTemplatePath(t *testing.T) {
	root := filepath.Join("..", "..", "testdata")
	got, err := ResolveTemplatePath(root, "TEST-VX0-a", "en")
	if err != nil {
		t.Fatalf("ResolveTemplatePath: %v", err)
	}
	want := filepath.Join(root, "us", "TEST-VB0-a")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
