package configio

import (
	"fmt"

	"github.com/gridsociety/mythy/pkg/catalog"
	"gopkg.in/yaml.v3"
)

// Parse reads a YAML byte stream into a ConfigFile and runs structural
// checks (Check). Detailed catalog validation is in Validate.
func Parse(in []byte) (*ConfigFile, error) {
	var cf ConfigFile
	if err := yaml.Unmarshal(in, &cf); err != nil {
		return nil, fmt.Errorf("parse YAML: %w", err)
	}
	if err := cf.Check(); err != nil {
		return nil, err
	}
	return &cf, nil
}

// UnknownKeyError lists keys in the file that don't exist in the catalog.
type UnknownKeyError struct{ Keys []string }

func (e *UnknownKeyError) Error() string {
	return fmt.Sprintf("unknown keys for this product: %v", e.Keys)
}

// ProductMismatchError signals device.product != live device product.
type ProductMismatchError struct {
	FromFile, FromDevice string
}

func (e *ProductMismatchError) Error() string {
	return fmt.Sprintf("product mismatch: file says %q, live device is %q (use --force to override)",
		e.FromFile, e.FromDevice)
}

// Validate runs the catalog-side checks: every settings key must
// resolve to a DATA in the catalog; every value must round-trip through
// the codec; device.product must match expectedProduct (unless empty).
func Validate(cf *ConfigFile, tpl *catalog.Template, expectedProduct string) error {
	if expectedProduct != "" && cf.Device.Product != "" && cf.Device.Product != expectedProduct {
		return &ProductMismatchError{FromFile: cf.Device.Product, FromDevice: expectedProduct}
	}

	known := make(map[string]*catalog.Data)
	for _, g := range tpl.Menu.WalkGroups(catalog.WalkOptions{IncludeHidden: true, IncludeReadOnly: true}) {
		for _, d := range g.Data {
			known[d.Name] = d
		}
	}

	var unknown []string
	for k := range cf.Settings {
		if _, ok := known[k]; !ok {
			unknown = append(unknown, k)
		}
	}
	if len(unknown) > 0 {
		return &UnknownKeyError{Keys: unknown}
	}

	// Per-value typed validation.
	for k, v := range cf.Settings {
		d := known[k]
		if _, err := YAMLToCodec(d.Tipo, d.Enum, v); err != nil {
			return fmt.Errorf("invalid value for %s: %w", k, err)
		}
		// Enum-label membership check.
		if (d.Tipo == "ENUM" || d.Tipo == "ENUM_LONG" || d.Tipo == "ENUM_BYTE") && d.Enum != "" {
			if s, ok := v.(string); ok {
				if e := tpl.Enums[d.Enum]; e != nil {
					if _, err := e.ValueFor(s); err != nil {
						return fmt.Errorf("invalid value for %s: %q is not a member of enum %s", k, s, d.Enum)
					}
				}
			}
		}
	}
	return nil
}
