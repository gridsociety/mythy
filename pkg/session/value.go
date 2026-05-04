package session

import (
	"fmt"
	"sort"
	"strings"
)

// Value is the decoded form of a register read, plus enough metadata
// for renderers and the YAML exporter to format it without
// re-traversing the catalog.
//
// One of Number / Str / Label / Compound is set depending on Tipo:
//   - numeric primitives → Number (raw integer; Format applies Scale/Decimals/Unit)
//   - STRING             → Str
//   - ENUM*              → Label (resolved) AND Number (raw value)
//   - compound types     → Compound (sub-field name → Value)
type Value struct {
	Tipo     string
	Number   int64
	Str      string
	Label    string
	EnumName string // <DATA ENUM=…>, e.g. "ON_OFF"; "" for non-ENUM types
	Compound map[string]Value

	// Display metadata sourced from the catalog. Optional.
	Unit     string // e.g. "Hz", "%", "^C"
	Decimals int    // decimal places to render
	Scale    int64  // raw / Scale = displayed value (1 if absent)
}

// Format renders the value as a human-readable string.
func (v Value) Format() string {
	switch v.Tipo {
	case "ENUM", "ENUM_BYTE", "ENUM_LONG":
		if v.Label != "" {
			return v.Label
		}
		return fmt.Sprintf("%d", v.Number)
	case "STRING":
		return fmt.Sprintf("%q", v.Str)
	case "":
		return ""
	}
	if v.Compound != nil {
		// Stable-sorted "k=v ..." rendering.
		keys := make([]string, 0, len(v.Compound))
		for k := range v.Compound {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		parts := make([]string, 0, len(keys))
		for _, k := range keys {
			parts = append(parts, fmt.Sprintf("%s=%s", k, v.Compound[k].Format()))
		}
		return "{" + strings.Join(parts, ", ") + "}"
	}
	// Numeric primitive
	scale := v.Scale
	if scale == 0 {
		scale = 1
	}
	if v.Decimals > 0 {
		f := float64(v.Number) / float64(scale)
		s := fmt.Sprintf("%.*f", v.Decimals, f)
		if v.Unit != "" {
			return s + " " + v.Unit
		}
		return s
	}
	if v.Unit != "" {
		return fmt.Sprintf("%d %s", v.Number/scale, v.Unit)
	}
	return fmt.Sprintf("%d", v.Number)
}
