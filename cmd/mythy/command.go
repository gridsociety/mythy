package main

import (
	"fmt"

	"github.com/gridsociety/mythy/pkg/catalog"
	"github.com/spf13/cobra"
)

func newCommandCmd(cf *catalogFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "command",
		Short: "Inspect device commands (catalog-only in Plan 1)",
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List every <COMMAND> entry in the catalog",
		RunE: func(cmd *cobra.Command, args []string) error {
			tpl, _, err := cf.load()
			if err != nil {
				return err
			}
			groups := tpl.Menu.WalkGroups(catalog.WalkOptions{IncludeHidden: true})
			out := cmd.OutOrStdout()
			for _, g := range groups {
				if len(g.Commands) == 0 {
					continue
				}
				fmt.Fprintf(out, "%s/\n", g.Path())
				for _, c := range g.Commands {
					if c.Description != "" {
						fmt.Fprintf(out, "  %-30s  %s\n", c.Name, c.Description)
					} else {
						fmt.Fprintf(out, "  %s\n", c.Name)
					}
				}
			}
			return nil
		},
	})
	return cmd
}
