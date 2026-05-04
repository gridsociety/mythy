package catalog

import (
	"encoding/xml"
	"fmt"
	"io"
	"os"
)

// Enum is a parsed <ENUM name="..."> with its <RANGE VALUE/OVERRIDE> entries.
//
// The wire-side numeric value is OVERRIDE; the human label is VALUE.
// (Yes, the attribute names are inverted relative to what you'd expect —
// see the actual XML, e.g. <RANGE VALUE="ON" OVERRIDE="1"/>.)
type Enum struct {
	Name    string
	Entries []EnumEntry // in document order
}

// EnumEntry is one (numeric value, human label) pair.
type EnumEntry struct {
	Value int
	Label string
}

// LabelFor returns the human label for a numeric value.
func (e *Enum) LabelFor(v int) (string, error) {
	for _, x := range e.Entries {
		if x.Value == v {
			return x.Label, nil
		}
	}
	return "", fmt.Errorf("enum %s: no label for value %d", e.Name, v)
}

// ValueFor returns the numeric value for a human label (case-sensitive).
func (e *Enum) ValueFor(label string) (int, error) {
	for _, x := range e.Entries {
		if x.Label == label {
			return x.Value, nil
		}
	}
	return 0, fmt.Errorf("enum %s: no value for label %q", e.Name, label)
}

// parseEnums streams <ENUM> elements from the template file. Note <ENUM> is
// uppercase but its 'name' attribute is lowercase. Inside, <RANGE VALUE=
// OVERRIDE=> entries each declare one enum value.
func parseEnums(path string, tpl *Template) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	tpl.Enums = make(map[string]*Enum)

	dec := xml.NewDecoder(f)
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return fmt.Errorf("token: %w", err)
		}
		se, ok := tok.(xml.StartElement)
		if !ok || se.Name.Local != "ENUM" {
			continue
		}

		var raw struct {
			Name    string `xml:"name,attr"`
			Entries []struct {
				Value    string `xml:"VALUE,attr"`
				Override int    `xml:"OVERRIDE,attr"`
			} `xml:"RANGE"`
		}
		if err := dec.DecodeElement(&raw, &se); err != nil {
			return fmt.Errorf("decode <ENUM>: %w", err)
		}
		e := &Enum{Name: raw.Name}
		for _, x := range raw.Entries {
			e.Entries = append(e.Entries, EnumEntry{Value: x.Override, Label: x.Value})
		}
		tpl.Enums[e.Name] = e
	}
}
