package codec

import (
	"fmt"
	"strings"

	"github.com/gridsociety/mythy/pkg/catalog"
)

// DecodeCompound walks the VAR layout of a Class and decodes the supplied
// registers into a map keyed by sub-field NAME. Numeric sub-fields decode
// to uint64 (unsigned) or int64 (signed). Strings decode to Go strings.
// Inline-enum sub-fields decode to their resolved label (string), or fall
// back to the numeric value if the label is missing.
func DecodeCompound(regs []uint16, cls *catalog.Class, _ map[string]*catalog.Enum) (map[string]any, error) {
	if cls == nil {
		return nil, fmt.Errorf("DecodeCompound: nil class")
	}
	out := make(map[string]any, len(cls.Vars))
	cursor := 0
	for _, v := range cls.Vars {
		w := varWidth(v)
		if w == 0 {
			return nil, fmt.Errorf("DecodeCompound: cannot determine width of VAR %s (TIPO=%s)", v.Name, v.Tipo)
		}
		if cursor+w > len(regs) {
			return nil, fmt.Errorf("DecodeCompound: %s wants %d regs starting at %d, have %d", v.Name, w, cursor, len(regs))
		}
		slice := regs[cursor : cursor+w]
		val, err := decodeVar(v, slice)
		if err != nil {
			return nil, fmt.Errorf("decode %s: %w", v.Name, err)
		}
		out[v.Name] = val
		cursor += w
	}
	return out, nil
}

// EncodeCompound is the inverse of DecodeCompound. Used by Set.
func EncodeCompound(values map[string]any, cls *catalog.Class, _ map[string]*catalog.Enum) ([]uint16, error) {
	if cls == nil {
		return nil, fmt.Errorf("EncodeCompound: nil class")
	}
	out := make([]uint16, 0, cls.Dim)
	for _, v := range cls.Vars {
		w := varWidth(v)
		if w == 0 {
			return nil, fmt.Errorf("EncodeCompound: cannot determine width of VAR %s", v.Name)
		}
		val, ok := values[v.Name]
		if !ok {
			val = nil
		}
		regs, err := encodeVar(v, val, w)
		if err != nil {
			return nil, fmt.Errorf("encode %s: %w", v.Name, err)
		}
		out = append(out, regs...)
	}
	return out, nil
}

// varWidth returns the register count for a sub-field, in registers.
func varWidth(v catalog.ClassVar) int {
	switch strings.ToUpper(v.Tipo) {
	case "BYTE", "UBYTE", "WORD", "UWORD", "BIT16", "ENUM", "ENUM_BYTE":
		return 1
	case "LONG", "ULONG", "BIT32", "ENUM_LONG":
		return 2
	case "STRING":
		if v.StringLen <= 0 {
			return 0
		}
		return (v.StringLen + 1) / 2 // ceil(N/2)
	}
	return 0
}

func decodeVar(v catalog.ClassVar, regs []uint16) (any, error) {
	switch strings.ToUpper(v.Tipo) {
	case "BYTE":
		return int64(int8(regs[0] & 0xFF)), nil
	case "UBYTE":
		return uint64(regs[0] & 0xFF), nil
	case "WORD":
		i, _ := DecodeWORD(regs)
		return int64(i), nil
	case "UWORD", "BIT16":
		u, _ := DecodeUWORD(regs)
		return uint64(u), nil
	case "LONG":
		i, _ := DecodeLONG(regs)
		return int64(i), nil
	case "ULONG", "BIT32":
		u, _ := DecodeULONG(regs)
		return uint64(u), nil
	case "STRING":
		return DecodeSTRING(regs)
	case "ENUM", "ENUM_BYTE":
		return resolveEnum(int(regs[0]), v), nil
	case "ENUM_LONG":
		u, _ := DecodeULONG(regs)
		return resolveEnum(int(u), v), nil
	}
	return nil, fmt.Errorf("decodeVar: unknown TIPO %q", v.Tipo)
}

func resolveEnum(num int, v catalog.ClassVar) any {
	if v.InlineEnum != nil {
		if lbl, err := v.InlineEnum.LabelFor(num); err == nil {
			return lbl
		}
	}
	return int64(num)
}

func encodeVar(v catalog.ClassVar, val any, width int) ([]uint16, error) {
	if val == nil {
		return make([]uint16, width), nil
	}
	switch strings.ToUpper(v.Tipo) {
	case "BYTE":
		return []uint16{uint16(toInt64(val) & 0xFF)}, nil
	case "UBYTE":
		return []uint16{uint16(toUint64(val) & 0xFF)}, nil
	case "WORD":
		return []uint16{uint16(toInt64(val) & 0xFFFF)}, nil
	case "UWORD", "BIT16":
		return []uint16{uint16(toUint64(val) & 0xFFFF)}, nil
	case "LONG":
		i := toInt64(val)
		return []uint16{uint16(i & 0xFFFF), uint16((i >> 16) & 0xFFFF)}, nil
	case "ULONG", "BIT32":
		u := toUint64(val)
		return []uint16{uint16(u & 0xFFFF), uint16((u >> 16) & 0xFFFF)}, nil
	case "STRING":
		s, _ := val.(string)
		return encodeStringChars(s, width), nil
	case "ENUM", "ENUM_BYTE":
		return []uint16{uint16(enumNum(val, v) & 0xFFFF)}, nil
	case "ENUM_LONG":
		u := uint64(enumNum(val, v))
		return []uint16{uint16(u & 0xFFFF), uint16((u >> 16) & 0xFFFF)}, nil
	}
	return nil, fmt.Errorf("encodeVar: unknown TIPO %q", v.Tipo)
}

func toInt64(v any) int64 {
	switch x := v.(type) {
	case int64:
		return x
	case uint64:
		return int64(x)
	case int:
		return int64(x)
	}
	return 0
}

func toUint64(v any) uint64 {
	switch x := v.(type) {
	case uint64:
		return x
	case int64:
		return uint64(x)
	case int:
		return uint64(x)
	}
	return 0
}

func enumNum(val any, v catalog.ClassVar) int {
	switch x := val.(type) {
	case string:
		if v.InlineEnum != nil {
			if n, err := v.InlineEnum.ValueFor(x); err == nil {
				return n
			}
		}
		return 0
	case int:
		// YAML decoders typically produce `int` for small integers; the
		// previous version of this switch only knew int64/uint64, so an
		// integer enum value silently became 0.
		return x
	case int64:
		return int(x)
	case uint64:
		return int(x)
	}
	return 0
}

// encodeStringChars encodes a string as `width` registers of NUL-padded
// 2-chars-per-register. The character count from the catalog is
// approximated as `width*2`; values longer than that are truncated.
func encodeStringChars(s string, width int) []uint16 {
	out := make([]uint16, width)
	maxBytes := width * 2
	if len(s) > maxBytes {
		s = s[:maxBytes]
	}
	for i := 0; i < len(s); i += 2 {
		hi := byte(s[i])
		var lo byte
		if i+1 < len(s) {
			lo = byte(s[i+1])
		}
		out[i/2] = uint16(hi)<<8 | uint16(lo)
	}
	return out
}
