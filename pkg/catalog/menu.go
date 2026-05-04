package catalog

import (
	"bytes"
	"encoding/gob"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"strings"
)

// Group is a node in the MENU tree (a <GROUP NAME=> element).
type Group struct {
	Name       string
	Visibility string // "1" / "2" / "3"; "" if absent. "3" = hidden.
	Refresh    bool   // REFRESH="YES"

	Children []*Group
	Data     []*Data
	Commands []*Command

	Parent *Group // nil for the synthetic root; rebuilt on cache load
}

// Data is a <DATA NAME=> leaf — a parameter or measurement.
// NAME links back to a <message name=> in the WSDL layer (cross-linked
// in Task 11).
type Data struct {
	Name        string
	Description string
	Tipo        string // <DATA TIPO=>
	Enum        string // <DATA ENUM=>: name of the <ENUM> to resolve labels
	Valore      string // <DATA VALORE=>: snapshot current value
	Default     string // <DATA DEFAULT=>
	Visibility  string
	Module      string // hardware module gate; empty if none
	ReadOnly    bool   // READONLY="YES"
	ReadAll     bool   // READALL="TRUE"
	Restart     bool   // RESTART="TRUE"
	Skip        bool   // SKIP="YES"
	License     string

	// Linked from WSDL in Task 11.
	Message *Message
}

// Command is a <COMMAND NAME=> leaf.
type Command struct {
	Name        string
	Description string
	Visibility  string

	Message *Message // linked in Task 11
}

// Path returns the slash-separated path from the root to this group,
// e.g. "Set/Base".
func (g *Group) Path() string {
	if g.Parent == nil || g.Parent.Name == "" {
		return g.Name
	}
	return g.Parent.Path() + "/" + g.Name
}

// FindGroup returns the descendant group at the given slash-separated path
// (relative to g). Returns nil if not found.
func (g *Group) FindGroup(path string) *Group {
	if path == "" {
		return g
	}
	cur := g
	for _, part := range strings.Split(path, "/") {
		var next *Group
		for _, c := range cur.Children {
			if c.Name == part {
				next = c
				break
			}
		}
		if next == nil {
			return nil
		}
		cur = next
	}
	return cur
}

// FindData returns the DATA leaf with the given NAME directly under g
// (non-recursive). Use Walk for recursive search.
func (g *Group) FindData(name string) *Data {
	for _, d := range g.Data {
		if d.Name == name {
			return d
		}
	}
	return nil
}

// parseMenu streams the <MENU> subtree out of the template file.
func parseMenu(path string, tpl *Template) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	dec := xml.NewDecoder(f)
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("token: %w", err)
		}
		se, ok := tok.(xml.StartElement)
		if !ok || se.Name.Local != "MENU" {
			continue
		}

		root := &Group{}
		if err := decodeMenu(dec, &se, root); err != nil {
			return err
		}
		tpl.Menu = root
		return nil
	}
	return nil // no <MENU> — leave Menu nil
}

// decodeMenu walks the children of an open <MENU> or <GROUP> StartElement.
func decodeMenu(dec *xml.Decoder, parentStart *xml.StartElement, parent *Group) error {
	for {
		tok, err := dec.Token()
		if err != nil {
			return fmt.Errorf("menu token: %w", err)
		}
		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "GROUP":
				g := &Group{Parent: parent}
				for _, a := range t.Attr {
					switch a.Name.Local {
					case "NAME":
						g.Name = a.Value
					case "VISIBILITY":
						g.Visibility = a.Value
					case "REFRESH":
						g.Refresh = a.Value == "YES"
					}
				}
				if err := decodeMenu(dec, &t, g); err != nil {
					return err
				}
				parent.Children = append(parent.Children, g)
			case "DATA":
				d := decodeData(t)
				// DATA elements may have nested <DATA> children (compound types
				// like CONTATORE / RELE / LED). Skip past them — they're
				// handled by Plan 1 Task 23 catalog extensions; for now just
				// consume them.
				if err := skipChildren(dec); err != nil {
					return err
				}
				parent.Data = append(parent.Data, d)
			case "COMMAND":
				c := decodeCommand(t)
				if err := skipChildren(dec); err != nil {
					return err
				}
				parent.Commands = append(parent.Commands, c)
			default:
				// Unknown element: skip it.
				if err := dec.Skip(); err != nil {
					return err
				}
			}
		case xml.EndElement:
			if t.Name.Local == parentStart.Name.Local {
				return nil
			}
		}
	}
}

func decodeData(se xml.StartElement) *Data {
	d := &Data{}
	for _, a := range se.Attr {
		switch a.Name.Local {
		case "NAME":
			d.Name = a.Value
		case "DSC":
			d.Description = a.Value
		case "TIPO":
			d.Tipo = a.Value
		case "ENUM":
			d.Enum = a.Value
		case "VALORE":
			d.Valore = a.Value
		case "DEFAULT":
			d.Default = a.Value
		case "VISIBILITY":
			d.Visibility = a.Value
		case "MODULE":
			d.Module = a.Value
		case "READONLY":
			d.ReadOnly = a.Value == "YES"
		case "READALL":
			d.ReadAll = a.Value == "TRUE"
		case "RESTART":
			d.Restart = a.Value == "TRUE"
		case "SKIP":
			d.Skip = a.Value == "YES"
		case "LICENSE":
			d.License = a.Value
		}
	}
	return d
}

func decodeCommand(se xml.StartElement) *Command {
	c := &Command{}
	for _, a := range se.Attr {
		switch a.Name.Local {
		case "NAME":
			c.Name = a.Value
		case "DSC":
			c.Description = a.Value
		case "VISIBILITY":
			c.Visibility = a.Value
		}
	}
	return c
}

// skipChildren consumes tokens until the matching EndElement is reached.
func skipChildren(dec *xml.Decoder) error {
	depth := 1
	for depth > 0 {
		tok, err := dec.Token()
		if err != nil {
			return err
		}
		switch tok.(type) {
		case xml.StartElement:
			depth++
		case xml.EndElement:
			depth--
		}
	}
	return nil
}

// groupGob is the on-disk shape used by GobEncode / GobDecode below.
// We round-trip every Group field EXCEPT Parent — back-pointers are
// rebuilt after decode by reparent (see cache.go). encoding/gob does
// not honor struct tags, so this manual exclusion is the canonical
// way to break the Group↔Parent cycle.
type groupGob struct {
	Name       string
	Visibility string
	Refresh    bool
	Children   []*Group
	Data       []*Data
	Commands   []*Command
}

// GobEncode implements gob.GobEncoder for *Group, emitting every field
// except Parent. Without this, gob would recurse infinitely through the
// Group↔Parent back-pointer cycle.
func (g *Group) GobEncode() ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(groupGob{
		Name:       g.Name,
		Visibility: g.Visibility,
		Refresh:    g.Refresh,
		Children:   g.Children,
		Data:       g.Data,
		Commands:   g.Commands,
	}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// GobDecode implements gob.GobDecoder for *Group. Parent is left nil;
// reparent (in cache.go) walks the tree post-decode to re-establish the
// back-pointers.
func (g *Group) GobDecode(data []byte) error {
	var gg groupGob
	if err := gob.NewDecoder(bytes.NewReader(data)).Decode(&gg); err != nil {
		return err
	}
	g.Name = gg.Name
	g.Visibility = gg.Visibility
	g.Refresh = gg.Refresh
	g.Children = gg.Children
	g.Data = gg.Data
	g.Commands = gg.Commands
	return nil
}
