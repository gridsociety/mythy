package codec

import "testing"

type fakeEnum struct {
	entries []struct {
		v int
		l string
	}
}

func (f *fakeEnum) LabelFor(v int) (string, error) {
	for _, e := range f.entries {
		if e.v == v {
			return e.l, nil
		}
	}
	return "", &EnumLookupError{Value: v}
}

func TestDecodeEnumOK(t *testing.T) {
	e := &fakeEnum{entries: []struct {
		v int
		l string
	}{{0, "OFF"}, {1, "ON"}}}
	got, err := DecodeEnum([]uint16{1}, e)
	if err != nil || got != "ON" {
		t.Errorf("DecodeEnum = %q, %v", got, err)
	}
}

func TestDecodeEnumUnknownValue(t *testing.T) {
	e := &fakeEnum{}
	_, err := DecodeEnum([]uint16{42}, e)
	if err == nil {
		t.Error("expected error for unknown enum value")
	}
}
