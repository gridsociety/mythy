package catalog

import (
	"encoding/xml"
	"fmt"
	"os"
)

// Template is one fully parsed Thytronic device template.
// Sub-resources (messages, types, menu) are filled in by later parsers
// (wsdl.go, types.go, menu.go) — see Load for the entry point.
type Template struct {
	Name            string // <DEVICE NAME>
	Identification  int    // <DEVICE IDENTIFICATION>
	Family          string // <DEVICE FAMILY>
	ProtocolRelease string // <DEVICE PROTOCOLRELEASE>
	XMLRelease      string // <DEVICE XMLRELEASE>

	// SourcePath is the file the template was loaded from.
	SourcePath string
}

type rawTemplate struct {
	XMLName         xml.Name `xml:"DEVICE"`
	Name            string   `xml:"NAME,attr"`
	Identification  int      `xml:"IDENTIFICATION,attr"`
	Family          string   `xml:"FAMILY,attr"`
	ProtocolRelease string   `xml:"PROTOCOLRELEASE,attr"`
	XMLRelease      string   `xml:"XMLRELEASE,attr"`
}

// ParseTemplate reads the <DEVICE> root attributes only. Later tasks add
// streaming parsers for <WSDL> and <MENU> that operate on the same file.
func ParseTemplate(path string) (*Template, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	var raw rawTemplate
	if err := xml.NewDecoder(f).Decode(&raw); err != nil {
		return nil, fmt.Errorf("decode %s: %w", path, err)
	}
	return &Template{
		Name:            raw.Name,
		Identification:  raw.Identification,
		Family:          raw.Family,
		ProtocolRelease: raw.ProtocolRelease,
		XMLRelease:      raw.XMLRelease,
		SourcePath:      path,
	}, nil
}

// ByAddrKey is the lookup key for a register by its on-the-wire (FC, address).
// Wire address is 0-based (XML num - 1). FC is 3 (WREG) or 4 (RREG).
type ByAddrKey struct {
	FC   int
	Addr int
}
