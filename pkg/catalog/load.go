package catalog

import (
	"fmt"
	"path/filepath"
)

// LoadOptions selects which device template to load from a catalog root.
// Either Identification or Product must be set; if both are set,
// Identification wins and Product is treated as a sanity check.
type LoadOptions struct {
	Root           string // path to the Templates/ folder
	Locale         string // "en" / "it" / "es" / "ru" / "tr"
	Identification int    // device IDENTIFICATION value (preferred)
	Product        string // PRODUCT name from Codifica.xml (alternative)
}

// Load is the one-stop entry point used by the CLI. It parses Codifica.xml,
// resolves the device entry, locates the localized template file, and parses
// the full template (root + WSDL + ENUMs + MENU + linking).
func Load(opts LoadOptions) (*Template, DeviceEntry, error) {
	if opts.Identification == 0 && opts.Product == "" {
		return nil, DeviceEntry{}, fmt.Errorf("Load: need Identification or Product")
	}
	if opts.Locale == "" {
		opts.Locale = "en"
	}

	idx, err := LoadCodifica(filepath.Join(opts.Root, "Codifica.xml"))
	if err != nil {
		return nil, DeviceEntry{}, err
	}

	var entry DeviceEntry
	switch {
	case opts.Identification != 0:
		entry, err = idx.ByIdentification(opts.Identification)
		if err != nil {
			return nil, DeviceEntry{}, err
		}
		if opts.Product != "" && opts.Product != entry.Product {
			return nil, DeviceEntry{}, fmt.Errorf("product mismatch: id=%d resolves to %q but caller requested %q",
				opts.Identification, entry.Product, opts.Product)
		}
	default:
		for _, d := range idx.Devices {
			if d.Product == opts.Product {
				entry = d
				break
			}
		}
		if entry.Product == "" {
			return nil, DeviceEntry{}, fmt.Errorf("product %q not in Codifica.xml", opts.Product)
		}
	}

	path, err := ResolveTemplatePath(opts.Root, entry.Template, opts.Locale)
	if err != nil {
		return nil, DeviceEntry{}, err
	}

	tpl, err := ParseTemplate(path)
	if err != nil {
		return nil, DeviceEntry{}, err
	}
	return tpl, entry, nil
}
