package main

import (
	"fmt"

	"github.com/gridsociety/mythy/pkg/catalog"
	"github.com/spf13/cobra"
)

// catalogFlags holds --templates / --device / --locale; reused by every
// catalog-only subcommand (show, list, describe, command, g61850, validate).
type catalogFlags struct {
	templatesRoot string
	device        string // PRODUCT name, e.g. "PROX-VX0-e"
	locale        string
	global        *formatFlags // populated by newRootCmd; nil-safe via formatFlags.resolve()
}

func (f *catalogFlags) bind(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVar(&f.templatesRoot, "templates", envOrString("MYTHY_TEMPLATES", ""),
		"Path to the ThyVisor Templates/ folder (or set MYTHY_TEMPLATES)")
	cmd.PersistentFlags().StringVar(&f.device, "device", envOrString("MYTHY_DEVICE", ""),
		"Device PRODUCT code (e.g. PROX-VX0-e). Required for catalog-only commands. (or set MYTHY_DEVICE)")
	cmd.PersistentFlags().StringVar(&f.locale, "locale", envOrString("MYTHY_LOCALE", "en"),
		"Locale (en|it|es|ru|tr; or set MYTHY_LOCALE)")
}

func (f *catalogFlags) load() (*catalog.Template, catalog.DeviceEntry, error) {
	if f.templatesRoot == "" {
		return nil, catalog.DeviceEntry{}, fmt.Errorf("--templates is required (or set MYTHY_TEMPLATES)")
	}
	if f.device == "" {
		return nil, catalog.DeviceEntry{}, fmt.Errorf("--device is required for catalog-only commands")
	}
	return catalog.Load(catalog.LoadOptions{
		Root:    f.templatesRoot,
		Locale:  f.locale,
		Product: f.device,
	})
}
