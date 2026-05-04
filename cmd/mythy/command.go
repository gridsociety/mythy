package main

import (
	"fmt"

	"github.com/gridsociety/mythy/pkg/catalog"
	"github.com/spf13/cobra"
)

func newCommandCmd(cf *catalogFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "command",
		Short: "Inspect or invoke <COMMAND> entries in the catalog",
	}
	cmd.AddCommand(newCommandListCmd(cf))
	cmd.AddCommand(newCommandInvokeCmd(cf))
	return cmd
}

// newCommandListCmd is the Plan-1 list subcommand, factored out so a
// sibling `invoke` verb can join it under the same parent.
func newCommandListCmd(cf *catalogFlags) *cobra.Command {
	return &cobra.Command{
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
	}
}
