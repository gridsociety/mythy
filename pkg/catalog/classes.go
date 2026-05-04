package catalog

import (
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"strconv"
)

// parseTypedefsAndClasses streams both <TYPEDEF> and <CLASS> elements.
// Typedefs are simple (NAME TIPO) aliases; Classes carry nested <VAR>
// children that describe compound layouts.
func parseTypedefsAndClasses(path string, tpl *Template) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	tpl.Typedefs = make(map[string]*Typedef)
	tpl.Classes = make(map[string]*Class)

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
		if !ok {
			continue
		}
		switch se.Name.Local {
		case "TYPEDEF":
			var raw struct {
				Name string `xml:"NAME,attr"`
				Tipo string `xml:"TIPO,attr"`
			}
			if err := dec.DecodeElement(&raw, &se); err != nil {
				return fmt.Errorf("decode <TYPEDEF>: %w", err)
			}
			tpl.Typedefs[raw.Name] = &Typedef{Name: raw.Name, Tipo: raw.Tipo}
		case "CLASS":
			cls, err := decodeClass(dec, &se)
			if err != nil {
				return err
			}
			if cls != nil {
				tpl.Classes[cls.Name] = cls
			}
		}
	}
}

// decodeClass parses one <CLASS NAME=… DIM=…> element with its <VAR>
// children. Inside each <VAR>, child <RANGE> elements declare either
// (a) a STRING length when the VAR's TIPO is STRING, or (b) inline
// enum entries when the VAR's TIPO is one of ENUM/ENUM_BYTE/ENUM_LONG.
func decodeClass(dec *xml.Decoder, se *xml.StartElement) (*Class, error) {
	cls := &Class{}
	for _, a := range se.Attr {
		switch a.Name.Local {
		case "NAME":
			cls.Name = a.Value
		case "DIM":
			n, _ := strconv.Atoi(a.Value)
			cls.Dim = n
		}
	}
	for {
		tok, err := dec.Token()
		if err != nil {
			return nil, fmt.Errorf("class %s: %w", cls.Name, err)
		}
		switch t := tok.(type) {
		case xml.StartElement:
			if t.Name.Local != "VAR" {
				if err := dec.Skip(); err != nil {
					return nil, err
				}
				continue
			}
			v := ClassVar{}
			for _, a := range t.Attr {
				switch a.Name.Local {
				case "NAME":
					v.Name = a.Value
				case "TIPO":
					v.Tipo = a.Value
				}
			}
			// Walk <RANGE> children of this VAR.
			ranges, err := collectRanges(dec, &t)
			if err != nil {
				return nil, err
			}
			applyVarRanges(&v, ranges)
			cls.Vars = append(cls.Vars, v)
		case xml.EndElement:
			if t.Name.Local == "CLASS" {
				return cls, nil
			}
		}
	}
}

// rangeRow is one parsed <RANGE> element (value plus optional override).
type rangeRow struct {
	Value    string
	Override string // "" if absent
}

// collectRanges reads every <RANGE> child of an open StartElement and
// returns them in order; closes when the matching EndElement arrives.
func collectRanges(dec *xml.Decoder, parent *xml.StartElement) ([]rangeRow, error) {
	var out []rangeRow
	for {
		tok, err := dec.Token()
		if err != nil {
			return nil, err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			if t.Name.Local != "RANGE" {
				if err := dec.Skip(); err != nil {
					return nil, err
				}
				continue
			}
			r := rangeRow{}
			for _, a := range t.Attr {
				switch a.Name.Local {
				case "VALUE":
					r.Value = a.Value
				case "OVERRIDE":
					r.Override = a.Value
				}
			}
			if err := dec.Skip(); err != nil {
				return nil, err
			}
			out = append(out, r)
		case xml.EndElement:
			if t.Name.Local == parent.Name.Local {
				return out, nil
			}
		}
	}
}

// applyVarRanges interprets the RANGE rows according to the VAR's TIPO:
// STRING + single RANGE with no OVERRIDE = char count; ENUM* + multiple
// RANGEs with OVERRIDE = inline enum.
func applyVarRanges(v *ClassVar, rows []rangeRow) {
	if len(rows) == 0 {
		return
	}
	switch v.Tipo {
	case "STRING":
		if len(rows) == 1 {
			n, _ := strconv.Atoi(rows[0].Value)
			v.StringLen = n
		}
	case "ENUM", "ENUM_BYTE", "ENUM_LONG":
		e := &Enum{Name: v.Name + "Inline"}
		for _, r := range rows {
			ov, _ := strconv.Atoi(r.Override)
			e.Entries = append(e.Entries, EnumEntry{Value: ov, Label: r.Value})
		}
		v.InlineEnum = e
	}
}
