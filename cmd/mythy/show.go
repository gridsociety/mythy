package main

import (
	"fmt"
	"io"
	"strings"

	"github.com/gridsociety/mythy/pkg/catalog"
	"github.com/spf13/cobra"
)

type showFlags struct {
	depth         int
	includeHidden bool
}

func newShowCmd(cf *catalogFlags) *cobra.Command {
	var sf showFlags
	cmd := &cobra.Command{
		Use:   "show [path]",
		Short: "Print the device's parameter menu tree",
		Long: `Print the menu tree for a device — the same hierarchy ThyVisor
shows in its left pane. Optionally rooted at a slash-separated path
("Set/Base") and limited in depth. Hidden ("Administrator") groups are
skipped unless --include-hidden is set.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			tpl, _, err := cf.load()
			if err != nil {
				return err
			}
			scope := tpl.Menu
			if len(args) == 1 && args[0] != "" {
				scope = scope.FindGroup(args[0])
				if scope == nil {
					return fmt.Errorf("path %q not found", args[0])
				}
			}
			return printGroup(cmd.OutOrStdout(), scope, 0, sf, true)
		},
	}
	cmd.Flags().IntVar(&sf.depth, "depth", 0, "Limit subtree depth (0 = no limit)")
	cmd.Flags().BoolVar(&sf.includeHidden, "include-hidden", false, "Include hidden (VISIBILITY=3) groups")
	return cmd
}

func printGroup(w io.Writer, g *catalog.Group, indent int, sf showFlags, isRoot bool) error {
	if g == nil {
		return nil
	}
	if !sf.includeHidden && g.Visibility == "3" {
		return nil
	}
	pad := strings.Repeat("  ", indent)
	if !isRoot || g.Name != "" {
		fmt.Fprintf(w, "%s%s/\n", pad, g.Name)
	}
	if sf.depth > 0 && indent >= sf.depth {
		return nil
	}
	for _, d := range g.Data {
		fmt.Fprintf(w, "%s  %s", pad, d.Name)
		if d.Description != "" {
			fmt.Fprintf(w, "  -- %s", d.Description)
		}
		fmt.Fprintln(w)
	}
	for _, c := range g.Commands {
		fmt.Fprintf(w, "%s  %s (command)", pad, c.Name)
		if c.Description != "" {
			fmt.Fprintf(w, "  -- %s", c.Description)
		}
		fmt.Fprintln(w)
	}
	for _, child := range g.Children {
		if err := printGroup(w, child, indent+1, sf, false); err != nil {
			return err
		}
	}
	return nil
}
