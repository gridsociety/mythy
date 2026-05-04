// Package main is the mythy CLI entry point.
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

const version = "0.0.1"

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:     "mythy",
		Short:   "CLI for Thytronic protection relays",
		Long:    "mythy talks to Thytronic Pro-X / Pro-N / XMR protection relays the way ThyVisor does, but smarter.",
		Version: version,
	}
	cf := &catalogFlags{}
	cf.bind(root)
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
	return root
}

func main() {
	if err := newRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
