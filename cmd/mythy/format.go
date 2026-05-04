package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// outputFormat is "human" | "json" | "yaml" | "unified".
type outputFormat string

const (
	formatHuman   outputFormat = "human"
	formatJSON    outputFormat = "json"
	formatYAML    outputFormat = "yaml"
	formatUnified outputFormat = "unified"
)

// formatFlags holds the global --format flag plus its --json / --yaml aliases.
type formatFlags struct {
	format string
	asJSON bool
	asYAML bool
}

// bind attaches --format / --json / --yaml as persistent flags on cmd.
// Call once on the root command in newRootCmd; subcommands inherit.
func (f *formatFlags) bind(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVar(&f.format, "format", "",
		"output format: human|json|yaml|unified (default: from MYTHY_FORMAT or human)")
	cmd.PersistentFlags().BoolVar(&f.asJSON, "json", false, "alias for --format=json")
	cmd.PersistentFlags().BoolVar(&f.asYAML, "yaml", false, "alias for --format=yaml")
}

// resolve returns the effective output format following the global
// precedence: explicit --format > --json/--yaml > MYTHY_FORMAT > "human".
func (f *formatFlags) resolve() outputFormat {
	if f == nil {
		return formatHuman
	}
	if f.format != "" {
		return outputFormat(f.format)
	}
	if f.asJSON {
		return formatJSON
	}
	if f.asYAML {
		return formatYAML
	}
	if v := os.Getenv("MYTHY_FORMAT"); v != "" {
		return outputFormat(v)
	}
	return formatHuman
}

// renderStruct emits any value using the requested format. Used by
// commands whose data is naturally structured (read, identify, diff).
func renderStruct(w io.Writer, format outputFormat, v any) error {
	switch format {
	case formatJSON:
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(v)
	case formatYAML:
		return yaml.NewEncoder(w).Encode(v)
	}
	return fmt.Errorf("renderStruct: format %q not supported here", format)
}
