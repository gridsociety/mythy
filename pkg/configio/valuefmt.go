package configio

import (
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/gridsociety/mythy/pkg/session"
)

// ValueToYAML converts a session.Value (the result of session.Read) into
// the YAML-friendly any-typed value used in ConfigFile.Settings.
//
// Rules: numeric → int64; STRING → string; ENUM → label (or int if
// label missing); ON_OFF enum → bool; compound → map[string]any.
func ValueToYAML(v session.Value) any {
	switch v.Tipo {
	case "STRING":
		return v.Str
	case "ENUM", "ENUM_BYTE", "ENUM_LONG":
		switch v.Label {
		case "ON":
			return true
		case "OFF":
			return false
		}
		if v.Label != "" {
			return v.Label
		}
		return v.Number
	}
	if v.Compound != nil {
		out := make(map[string]any, len(v.Compound))
		for k, sub := range v.Compound {
			out[k] = ValueToYAML(sub)
		}
		return out
	}
	if strings.HasPrefix(v.Tipo, "BIT") || v.Tipo == "ARRAY" {
		// Render as uppercase hex.
		return hexFromInt(v.Number)
	}
	return v.Number
}

func hexFromInt(n int64) string {
	b := make([]byte, 4)
	b[0] = byte(n >> 24)
	b[1] = byte(n >> 16)
	b[2] = byte(n >> 8)
	b[3] = byte(n)
	return strings.ToUpper(hex.EncodeToString(b))
}

// YAMLToCodec converts a YAML-loaded any into the typed value that
// session.SetMany accepts: uint64 / int64 / string / map[string]any.
//
// tipo is the catalog TIPO of the target. enumName is the value of
// <DATA ENUM=> when tipo is ENUM*; "" otherwise.
func YAMLToCodec(tipo, enumName string, in any) (any, error) {
	switch tipo {
	case "UBYTE", "UWORD", "ULONG", "BIT16", "BIT32":
		switch x := in.(type) {
		case int64:
			if x < 0 {
				return nil, fmt.Errorf("YAMLToCodec %s: negative int %d", tipo, x)
			}
			return uint64(x), nil
		case int:
			if x < 0 {
				return nil, fmt.Errorf("YAMLToCodec %s: negative int %d", tipo, x)
			}
			return uint64(x), nil
		case uint64:
			return x, nil
		}
		return nil, fmt.Errorf("YAMLToCodec %s: expected int, got %T", tipo, in)
	case "BYTE", "WORD", "LONG":
		switch x := in.(type) {
		case int64:
			return x, nil
		case int:
			return int64(x), nil
		case uint64:
			return int64(x), nil
		}
		return nil, fmt.Errorf("YAMLToCodec %s: expected int, got %T", tipo, in)
	case "STRING":
		s, ok := in.(string)
		if !ok {
			return nil, fmt.Errorf("YAMLToCodec STRING: expected string, got %T", in)
		}
		return s, nil
	case "ENUM", "ENUM_BYTE", "ENUM_LONG":
		// Bool only valid for ON_OFF.
		if b, ok := in.(bool); ok {
			if enumName != "ON_OFF" {
				return nil, fmt.Errorf("YAMLToCodec ENUM %s: bool only valid for ON_OFF", enumName)
			}
			if b {
				return "ON", nil
			}
			return "OFF", nil
		}
		// String labels pass through; numerics also acceptable.
		switch x := in.(type) {
		case string:
			return x, nil
		case int64:
			return x, nil
		case int:
			return int64(x), nil
		}
		return nil, fmt.Errorf("YAMLToCodec ENUM: expected string/bool/int, got %T", in)
	}
	// Compound (TIMER, RELE, …) — caller passes a nested map.
	if m, ok := in.(map[string]any); ok {
		return m, nil
	}
	return nil, fmt.Errorf("YAMLToCodec: unsupported tipo %q (value type %T)", tipo, in)
}
