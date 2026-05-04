package main

import (
	"fmt"
	"os"

	"github.com/gridsociety/mythy/pkg/catalog"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// configFile is the on-disk YAML schema mythy uses for export/import.
// Plan 3 adds full encode/decode; Plan 1 only validates the keys.
type configFile struct {
	MythyVersion int                    `yaml:"mythy_version"`
	Device       configDevice           `yaml:"device"`
	Settings     map[string]interface{} `yaml:"settings"`
}

type configDevice struct {
	Product string `yaml:"product"`
}

func newValidateCmd(cf *catalogFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "validate <file>",
		Short: "Validate a config YAML against the device's catalog (no connection)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			tpl, entry, err := cf.load()
			if err != nil {
				return err
			}
			data, err := os.ReadFile(args[0])
			if err != nil {
				return fmt.Errorf("read %s: %w", args[0], err)
			}
			var cfg configFile
			if err := yaml.Unmarshal(data, &cfg); err != nil {
				return fmt.Errorf("parse YAML: %w", err)
			}
			if cfg.MythyVersion != 1 {
				return fmt.Errorf("unsupported mythy_version %d", cfg.MythyVersion)
			}
			if cfg.Device.Product != entry.Product {
				return fmt.Errorf("device.product mismatch: file says %q, catalog says %q",
					cfg.Device.Product, entry.Product)
			}

			// Index every <DATA NAME> the catalog exposes.
			known := make(map[string]*catalog.Data)
			for _, g := range tpl.Menu.WalkGroups(catalog.WalkOptions{IncludeHidden: true}) {
				for _, d := range g.Data {
					known[d.Name] = d
				}
			}

			var unknown []string
			for k := range cfg.Settings {
				if _, ok := known[k]; !ok {
					unknown = append(unknown, k)
				}
			}
			if len(unknown) > 0 {
				return fmt.Errorf("unknown key(s) for %s: %v", entry.Product, unknown)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "OK: %d settings, all match %s\n",
				len(cfg.Settings), entry.Product)
			return nil
		},
	}
}
