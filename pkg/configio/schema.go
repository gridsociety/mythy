// Package configio implements mythy's YAML configuration file format
// (SPEC § 4): export, import, diff, and apply.
package configio

import "fmt"

// ConfigFile is the on-disk schema. Keys at the top level:
//   mythy_version: 1
//   device:        { product, identification, ... }
//   settings:      { <DATA NAME>: <decoded value>, ... }
type ConfigFile struct {
	MythyVersion int                    `yaml:"mythy_version"`
	Device       Device                 `yaml:"device"`
	Settings     map[string]interface{} `yaml:"settings"`
}

// Device captures the device identification fields written to / read
// from the file. None are required at parse time; Apply uses Product
// to refuse mismatched imports unless --force.
type Device struct {
	Product         string `yaml:"product,omitempty"`
	Identification  int    `yaml:"identification,omitempty"`
	SerialNumber    int64  `yaml:"serial_number,omitempty"`
	FwRelease       string `yaml:"fw_release,omitempty"`
	ProtocolRelease string `yaml:"protocol_release,omitempty"`
	ExportedFrom    string `yaml:"exported_from,omitempty"`
	ExportedAt      string `yaml:"exported_at,omitempty"`
	// Locale carries the --locale value the export ran under, recorded
	// so import / diff / validate can reconcile against the CLI's
	// --locale and refuse mismatches that would otherwise silently
	// remap enum labels to zero (issue #4). Empty in YAML files
	// produced by mythy builds prior to the issue-#4 fix; the import
	// path treats that as "legacy, fall back to the CLI flag".
	Locale string `yaml:"locale,omitempty"`
}

// Check validates the structural invariants. Detailed key/value
// validation against the catalog lives in Validate (parse.go).
func (f *ConfigFile) Check() error {
	if f.MythyVersion != 1 {
		return fmt.Errorf("unsupported mythy_version %d (this build understands version 1)", f.MythyVersion)
	}
	if f.Settings == nil {
		f.Settings = map[string]any{}
	}
	return nil
}
