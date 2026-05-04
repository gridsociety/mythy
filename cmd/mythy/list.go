package main

import (
	"fmt"

	"github.com/gridsociety/mythy/pkg/catalog"
	"github.com/spf13/cobra"
)

type listFlags struct {
	scope         string
	includeHidden bool
}

func newListCmd(cf *catalogFlags) *cobra.Command {
	var lf listFlags
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List every parameter / measurement leaf",
		Long: `Print every <DATA> leaf in the catalog as one line:
  <menu/path>  <NAME>  [<DSC>]  [<TIPO>]  [r/w]
Use --scope to restrict to a subtree.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			tpl, _, err := cf.load()
			if err != nil {
				return err
			}
			scope := tpl.Menu
			if lf.scope != "" {
				scope = scope.FindGroup(lf.scope)
				if scope == nil {
					return fmt.Errorf("scope %q not found", lf.scope)
				}
			}
			leaves := scope.WalkData(catalog.WalkOptions{IncludeHidden: lf.includeHidden})
			out := cmd.OutOrStdout()
			for _, d := range leaves {
				path := ""
				if d.Message != nil {
					path = d.Message.Name
				}
				rw := "rw"
				if d.ReadOnly || d.Tipo == "" {
					rw = "ro"
				}
				fmt.Fprintf(out, "%-30s  %-30s  %-10s  %s\n", path, d.Name, d.Tipo, rw)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&lf.scope, "scope", "", "Restrict to a slash-separated menu path")
	cmd.Flags().BoolVar(&lf.includeHidden, "include-hidden", false, "Include hidden groups")
	return cmd
}
