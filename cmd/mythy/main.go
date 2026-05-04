// Package main is the mythy CLI entry point.
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

const version = "0.0.1"

func newRootCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "mythy",
		Short:   "CLI for Thytronic protection relays",
		Long:    "mythy talks to Thytronic Pro-X / Pro-N / XMR protection relays the way ThyVisor does, but smarter.",
		Version: version,
	}
}

func main() {
	if err := newRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
