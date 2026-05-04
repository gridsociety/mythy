package catalog

import (
	"encoding/xml"
	"fmt"
	"io"
	"os"
)

// Message is one entry in <WSDL>.
// Names are case-sensitive symbolic ids that are also used by <DATA NAME=>
// and <COMMAND NAME=> elements in the MENU layer (linked in Task 11).
type Message struct {
	Name        string // <message name=>
	Description string // <message dsc=>
	Type        string // "RREG" or "WREG"; "" for *Response messages
	Class       string // <message CLASS=>, e.g. UBYTE_PARAM
	Level       string // <message level=>, e.g. BY_USER_LEVEL_2
	Num         int    // <message num=> — XML 1-based register number
	Dim         int    // <message dim=> — register count

	// For *Response messages with <part> children describing the layout
	// of a multi-register read.
	Parts []Part
}

// Part is one field inside a multi-register response shape.
type Part struct {
	Name string // <part name=>
	Type string // <part type=>, e.g. WORD, LONG, STRING, ENUM, ARRAY
}

// WireAddr returns the 0-based Modbus address for this message.
// XML "num" is 1-based; wire address is num - 1.
func (m *Message) WireAddr() int { return m.Num - 1 }

// FC returns the Modbus function code that should be used to read this
// register: FC04 for RREG, FC03 for WREG. Returns 0 for response shapes.
func (m *Message) FC() int {
	switch m.Type {
	case "RREG":
		return 4
	case "WREG":
		return 3
	}
	return 0
}

// parseWSDLMessages streams <message> elements out of an open template
// file and populates tpl.Messages and tpl.ByAddr. Streaming is needed
// because real templates are 5–6 MB and loading the whole tree blows up
// allocations.
func parseWSDLMessages(path string, tpl *Template) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	tpl.Messages = make(map[string]*Message)
	tpl.ByAddr = make(map[ByAddrKey]*Message)

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
		if !ok || se.Name.Local != "message" {
			continue
		}

		var raw struct {
			Name  string `xml:"name,attr"`
			Dsc   string `xml:"dsc,attr"`
			Type  string `xml:"type,attr"`
			Class string `xml:"CLASS,attr"`
			Level string `xml:"level,attr"`
			Num   int    `xml:"num,attr"`
			Dim   int    `xml:"dim,attr"`
			Parts []struct {
				Name string `xml:"name,attr"`
				Type string `xml:"type,attr"`
			} `xml:"part"`
		}
		if err := dec.DecodeElement(&raw, &se); err != nil {
			return fmt.Errorf("decode <message>: %w", err)
		}

		msg := &Message{
			Name:        raw.Name,
			Description: raw.Dsc,
			Type:        raw.Type,
			Class:       raw.Class,
			Level:       raw.Level,
			Num:         raw.Num,
			Dim:         raw.Dim,
		}
		for _, p := range raw.Parts {
			msg.Parts = append(msg.Parts, Part{Name: p.Name, Type: p.Type})
		}
		tpl.Messages[msg.Name] = msg
		if fc := msg.FC(); fc != 0 && msg.Num > 0 {
			tpl.ByAddr[ByAddrKey{FC: fc, Addr: msg.WireAddr()}] = msg
		}
	}
}
