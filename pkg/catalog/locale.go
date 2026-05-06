package catalog

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ErrUnknownLocale is returned by locale-handling functions when the locale
// code is not one of the supported letters.
var ErrUnknownLocale = errors.New("unknown locale")

// localeLetters maps a locale code to (folder, letter) for the catalog.
// X is the placeholder used in Codifica.xml; we substitute the locale letter
// at load time.
var localeLetters = map[string]struct {
	folder, letter string
}{
	"en": {"us", "B"},
	"it": {"it", "A"},
	"es": {"es", "S"},
	"ru": {"ru", "R"},
	"tr": {"tr", "T"},
}

// SupportedLocales returns the list of locale codes in stable order.
func SupportedLocales() []string {
	return []string{"en", "it", "es", "ru", "tr"}
}

// substituteLocaleLetter rewrites the X placeholder in a TEMPLATE name (as
// found in Codifica.xml) to the actual locale letter. If the template name
// has no X placeholder it is returned unchanged.
func substituteLocaleLetter(template, locale string) (string, error) {
	loc, ok := localeLetters[locale]
	if !ok {
		return "", fmt.Errorf("%w: %q", ErrUnknownLocale, locale)
	}
	// Codifica.xml entries always carry the placeholder as a literal "X" in the
	// locale slot, e.g. "PROX-VX0-e" (where the second X is the locale slot;
	// the first X is part of the family name "PROX"). Other entries
	// (e.g. "NA016-a") are not localized — passthrough.
	idx := strings.LastIndex(template, "X")
	if idx == -1 {
		return template, nil
	}
	return template[:idx] + loc.letter + template[idx+1:], nil
}

// ResolveTemplatePath returns the absolute path to the device template file
// for the given (Codifica TEMPLATE name, locale) pair under the catalog root.
// Falls back to <root>/default/<localized-name> if the locale folder doesn't
// have it, matching ThyVisor's behavior.
func ResolveTemplatePath(root, templateName, locale string) (string, error) {
	loc, ok := localeLetters[locale]
	if !ok {
		return "", fmt.Errorf("%w: %q", ErrUnknownLocale, locale)
	}
	resolved, err := substituteLocaleLetter(templateName, locale)
	if err != nil {
		return "", err
	}
	primary := filepath.Join(root, loc.folder, resolved)
	if _, err := os.Stat(primary); err == nil {
		return primary, nil
	}
	fallback := filepath.Join(root, "default", resolved)
	if _, err := os.Stat(fallback); err == nil {
		return fallback, nil
	}
	return "", fmt.Errorf("template %q not found under %s/{%s,default}", resolved, root, loc.folder)
}
