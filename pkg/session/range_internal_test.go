package session

import (
	"strings"
	"testing"

	"github.com/gridsociety/mythy/pkg/catalog"
)

func TestValidateAgainstRangeSingleBand(t *testing.T) {
	d := &catalog.Data{
		Name: "test",
		Tipo: "ULONG",
		Range: &catalog.DataRange{Min: 100, Max: 200, Step: 5},
	}
	cases := []struct {
		val     int64
		wantErr string // empty = expect nil
	}{
		{100, ""},                                // lower bound
		{200, ""},                                // upper bound
		{155, ""},                                // (155 - 100) % 5 == 0
		{99, "out of catalog range [100, 200]"},  // below
		{201, "out of catalog range [100, 200]"}, // above
		{102, "violates step=5"},                 // wrong step
	}
	for _, c := range cases {
		err := validateAgainstRange(c.val, d)
		switch {
		case c.wantErr == "" && err != nil:
			t.Errorf("value=%d: unexpected error: %v", c.val, err)
		case c.wantErr != "" && err == nil:
			t.Errorf("value=%d: expected error containing %q, got nil", c.val, c.wantErr)
		case c.wantErr != "" && err != nil && !strings.Contains(err.Error(), c.wantErr):
			t.Errorf("value=%d: error %q does not contain %q", c.val, err.Error(), c.wantErr)
		}
	}
}

func TestValidateAgainstRangeMultiBand(t *testing.T) {
	// Regression for #2: a DATA with multiple <RANGE> children defines
	// piecewise-valid bands with different step sizes. The validator
	// must accept a value matching any band, not just the last one.
	// Bands mirror NV10P-EA0-u's VLineaPrimario_1 in miniature.
	d := &catalog.Data{
		Name: "VLineaPrimario_1",
		Tipo: "ULONG",
		Range: &catalog.DataRange{Min: 50000, Max: 500000, Step: 1000}, // last band, kept for back-compat
		Ranges: []*catalog.DataRange{
			{Min: 50, Max: 499, Step: 1},
			{Min: 500, Max: 4990, Step: 10},
			{Min: 5000, Max: 49900, Step: 100},
			{Min: 50000, Max: 500000, Step: 1000},
		},
	}

	t.Run("accepted in each band", func(t *testing.T) {
		ok := []int64{50, 400, 499, 500, 4990, 5000, 49900, 50000, 500000}
		for _, v := range ok {
			if err := validateAgainstRange(v, d); err != nil {
				t.Errorf("value=%d should be accepted (#2 reproducer for band-1 cases), got %v", v, err)
			}
		}
	})

	t.Run("rejected below all bands", func(t *testing.T) {
		err := validateAgainstRange(int64(49), d)
		if err == nil {
			t.Fatal("expected rejection")
		}
		if !strings.Contains(err.Error(), "not in any of catalog bands") {
			t.Errorf("error %q should mention multi-band", err.Error())
		}
		// Error message must enumerate every band so the operator sees
		// what's actually allowed.
		for _, want := range []string{"[50,499 step 1]", "[500,4990 step 10]",
			"[5000,49900 step 100]", "[50000,500000 step 1000]"} {
			if !strings.Contains(err.Error(), want) {
				t.Errorf("error %q missing band %q", err.Error(), want)
			}
		}
	})

	t.Run("rejected above all bands", func(t *testing.T) {
		if err := validateAgainstRange(int64(500001), d); err == nil {
			t.Error("expected rejection for value above last band")
		}
	})

	t.Run("rejected in a gap (none of the bands match step)", func(t *testing.T) {
		// 401 is in band 1's [50,499] but band 1 has step=1 so 401 IS valid.
		// Try 502: in band 2's [500,4990] but step=10 → (502-500)%10=2 → rejected.
		// 502 also doesn't fit band 1 [50,499]. So it should be rejected.
		if err := validateAgainstRange(int64(502), d); err == nil {
			t.Error("502 is in band-2 bounds but violates its step=10; expected rejection")
		}
	})
}

func TestValidateAgainstRangeNonNumericSkipped(t *testing.T) {
	// STRING and ENUM TIPOs are validated elsewhere; validateAgainstRange
	// must bail out before trying to read Range numerics.
	for _, tipo := range []string{"STRING", "ENUM", "ENUM_BYTE", "ENUM_LONG"} {
		d := &catalog.Data{
			Name: "x",
			Tipo: tipo,
			Range: &catalog.DataRange{Min: 1, Max: 2, Step: 1},
		}
		if err := validateAgainstRange("anything", d); err != nil {
			t.Errorf("TIPO=%s: expected skip, got %v", tipo, err)
		}
	}
}

func TestValidateAgainstRangeNoRangesNoOp(t *testing.T) {
	// DATA without any <RANGE> child must not reject anything; the
	// type-width fallback in encodeForWrite is the only guardrail.
	d := &catalog.Data{Name: "x", Tipo: "ULONG"}
	if err := validateAgainstRange(int64(123456789), d); err != nil {
		t.Errorf("no Range/Ranges: expected nil, got %v", err)
	}
}
