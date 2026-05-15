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

	Messages map[string]*Message    // by symbolic name (Task 7)
	ByAddr   map[ByAddrKey]*Message // (FC, wireAddr) lookup (Task 7)
	Enums    map[string]*Enum       // by name (Task 8)
	Menu     *Group                 // root menu (Task 9)

	Modules  []ModuleInfo        // <MODULES>/<MODULE> (Task 23)
	Typedefs map[string]*Typedef // <TYPEDEF> (Task 23)
	Classes  map[string]*Class   // <CLASS> (Task 23)

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

// ParseTemplate reads the <DEVICE> root attributes and the <WSDL> messages.
// Later tasks add ENUMs (Task 8) and the MENU tree (Task 9).
func ParseTemplate(path string) (*Template, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	var raw rawTemplate
	dec := xml.NewDecoder(f)
	if err := dec.Decode(&raw); err != nil {
		f.Close()
		return nil, fmt.Errorf("decode root %s: %w", path, err)
	}
	f.Close()

	tpl := &Template{
		Name:            raw.Name,
		Identification:  raw.Identification,
		Family:          raw.Family,
		ProtocolRelease: raw.ProtocolRelease,
		XMLRelease:      raw.XMLRelease,
		SourcePath:      path,
	}
	if err := parseWSDLMessages(path, tpl); err != nil {
		return nil, fmt.Errorf("parse WSDL: %w", err)
	}
	if err := parseEnums(path, tpl); err != nil {
		return nil, fmt.Errorf("parse ENUMs: %w", err)
	}
	if err := parseModules(path, tpl); err != nil {
		return nil, fmt.Errorf("parse MODULES: %w", err)
	}
	if err := parseTypedefsAndClasses(path, tpl); err != nil {
		return nil, fmt.Errorf("parse TYPEDEFs/CLASSes: %w", err)
	}
	if err := parseMenu(path, tpl); err != nil {
		return nil, fmt.Errorf("parse MENU: %w", err)
	}
	linkMenuToMessages(tpl)
	tpl.resolveTypedefs()
	return tpl, nil
}

// ByAddrKey is the lookup key for a register by its on-the-wire (FC, address).
// Wire address is 0-based (XML num - 1). FC is 3 (WREG) or 4 (RREG).
type ByAddrKey struct {
	FC   int
	Addr int
}
