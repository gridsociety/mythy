// Package main is the mythy CLI entry point.
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// version is overridden at build time via -ldflags "-X main.version=...".
// The release workflow injects the git tag; local builds keep the default.
// The leading "v" (Go module convention, e.g. "v1.0.0") is stripped on
// display because cobra renders this as "mythy version <V>" — the word
// "version" already encodes the "v".
var version = "development-build"

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:     "mythy",
		Short:   "CLI for Thytronic protection relays",
		Long:    "Command-line tool to configure, monitor, and manage Thytronic protection relays.",
		Version: strings.TrimPrefix(version, "v"),
	}
	cf := &catalogFlags{}
	cf.bind(root)

	// Global --format flag (with --json / --yaml aliases). Subcommands
	// read cf.global.resolve() to pick the renderer.
	gf := &formatFlags{}
	gf.bind(root)
	cf.global = gf

	root.AddCommand(newShowCmd(cf))
	root.AddCommand(newListCmd(cf))
	root.AddCommand(newDescribeCmd(cf))
	root.AddCommand(newCommandCmd(cf))
	root.AddCommand(newG61850Cmd(cf))
	root.AddCommand(newValidateCmd(cf))
	root.AddCommand(newIdentifyCmd(cf))
	root.AddCommand(newReadCmd(cf))
	root.AddCommand(newSetCmd(cf))
	root.AddCommand(newRebootCmd(cf))
	root.AddCommand(newResetCmd(cf))
	root.AddCommand(newClockSetCmd(cf))
	root.AddCommand(newNetSetCmd(cf))
	root.AddCommand(newRawCmd(cf))
	root.AddCommand(newExportCmd(cf))
	root.AddCommand(newDiffCmd(cf))
	root.AddCommand(newImportCmd(cf))
	return root
}

func main() {
	if err := newRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
