package catalog

import (
	"encoding/xml"
	"fmt"
	"io"
	"os"
)

// parseModules streams <MODULES>/<MODULE> entries from the template file.
func parseModules(path string, tpl *Template) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

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
		if !ok || se.Name.Local != "MODULE" {
			continue
		}
		var raw struct {
			Name      string `xml:"name,attr"`
			Variabile string `xml:"variabile,attr"`
			Dsc       string `xml:"dsc,attr"`
		}
		if err := dec.DecodeElement(&raw, &se); err != nil {
			return fmt.Errorf("decode <MODULE>: %w", err)
		}
		tpl.Modules = append(tpl.Modules, ModuleInfo{
			Name:        raw.Name,
			Variabile:   raw.Variabile,
			Description: raw.Dsc,
		})
	}
}
