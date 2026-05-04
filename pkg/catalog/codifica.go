// Package catalog parses the Thytronic ThyVisor template catalog
// (the Templates/ folder shipped with ThyVisor).
package catalog

import (
	"encoding/xml"
	"errors"
	"fmt"
	"os"
)

// ErrUnknownIdentification means the IDENTIFICATION value is not in Codifica.xml.
var ErrUnknownIdentification = errors.New("unknown IDENTIFICATION")

// Codifica is the parsed Codifica.xml master index.
type Codifica struct {
	Version string        // <IDENTIFICATIONS Version="2.14">
	Devices []DeviceEntry // flattened across all <DEVICES FAMILY="…"> groups
}

// DeviceEntry is one product entry in Codifica.xml.
type DeviceEntry struct {
	Family         string // from the parent <DEVICES FAMILY="…">
	Product        string // PRODUCT="…"
	Identification int    // IDENTIFICATION="…"
	Template       string // TEMPLATE="…", may contain "X" as locale placeholder
	Rule           string // optional RULE="…"
	IEC61850       bool   // IEC61850="true"
}

// LoadCodifica reads and parses a Codifica.xml file.
func LoadCodifica(path string) (*Codifica, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	var raw struct {
		XMLName xml.Name `xml:"IDENTIFICATIONS"`
		Version string   `xml:"Version,attr"`
		Devices []struct {
			Family  string `xml:"FAMILY,attr"`
			Devices []struct {
				Product        string `xml:"PRODUCT,attr"`
				Identification int    `xml:"IDENTIFICATION,attr"`
				Template       string `xml:"TEMPLATE,attr"`
				Rule           string `xml:"RULE,attr"`
				IEC61850       string `xml:"IEC61850,attr"`
			} `xml:"DEVICE"`
		} `xml:"DEVICES"`
	}

	if err := xml.NewDecoder(f).Decode(&raw); err != nil {
		return nil, fmt.Errorf("decode %s: %w", path, err)
	}

	out := &Codifica{Version: raw.Version}
	for _, group := range raw.Devices {
		for _, d := range group.Devices {
			out.Devices = append(out.Devices, DeviceEntry{
				Family:         group.Family,
				Product:        d.Product,
				Identification: d.Identification,
				Template:       d.Template,
				Rule:           d.Rule,
				IEC61850:       d.IEC61850 == "true",
			})
		}
	}
	return out, nil
}

// ByIdentification looks up a device by its IDENTIFICATION value
// (as returned by FC04 read of register 0x143E word 0).
func (c *Codifica) ByIdentification(ident int) (DeviceEntry, error) {
	for _, d := range c.Devices {
		if d.Identification == ident {
			return d, nil
		}
	}
	return DeviceEntry{}, fmt.Errorf("%w: %d", ErrUnknownIdentification, ident)
}
