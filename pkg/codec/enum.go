package codec

import "fmt"

// EnumResolver is satisfied by anything that can map a numeric enum value
// to a human label — in practice, *catalog.Enum.
type EnumResolver interface {
	LabelFor(int) (string, error)
}

// EnumLookupError is returned when the enum has no entry for the value.
type EnumLookupError struct {
	Value int
	Enum  string
}

func (e *EnumLookupError) Error() string {
	if e.Enum != "" {
		return fmt.Sprintf("enum %s: no label for %d", e.Enum, e.Value)
	}
	return fmt.Sprintf("enum: no label for %d", e.Value)
}

// DecodeEnum decodes the register(s) as an integer and resolves it to its
// label via the provided resolver. Most enums are 1 reg; ENUM_LONG is 2 regs
// (low-word-first). The caller picks via the regs slice length.
func DecodeEnum(regs []uint16, e EnumResolver) (string, error) {
	var v int
	switch len(regs) {
	case 1:
		v = int(regs[0])
	case 2:
		u, err := DecodeULONG(regs)
		if err != nil {
			return "", err
		}
		v = int(u)
	default:
		return "", fmt.Errorf("DecodeEnum: %w (need 1 or 2 regs, got %d)", ErrLength, len(regs))
	}
	return e.LabelFor(v)
}
