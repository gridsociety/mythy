package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

func newReadCmd(cf *catalogFlags) *cobra.Command {
	var conn connFlags
	var scope string
	var includeHidden bool

	cmd := &cobra.Command{
		Use:   "read [name ...]",
		Short: "Read one or more parameters or measurements from the device",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			s, err := conn.build(ctx, cf)
			if err != nil {
				return err
			}
			defer s.Close()

			out := cmd.OutOrStdout()
			format := cf.global.resolve()

			result := make(map[string]any)
			if scope != "" {
				vals, err := s.ReadScope(ctx, scope, includeHidden)
				if err != nil {
					return err
				}
				for k, v := range vals {
					if format == formatHuman || format == formatUnified || format == "" {
						fmt.Fprintf(out, "%-30s %s\n", k, v.Format())
					} else {
						result[k] = v.Format()
					}
				}
			} else {
				if len(args) == 0 {
					return fmt.Errorf("read: provide names or --scope")
				}
				for _, name := range args {
					v, err := s.Read(ctx, name)
					if err != nil {
						return err
					}
					if format == formatHuman || format == formatUnified || format == "" {
						fmt.Fprintf(out, "%-30s %s\n", name, v.Format())
					} else {
						result[name] = v.Format()
					}
				}
			}
			if format == formatJSON || format == formatYAML {
				return renderStruct(out, format, result)
			}
			return nil
		},
	}
	conn.bind(cmd)
	cmd.Flags().StringVar(&scope, "scope", "", "menu path to read recursively")
	cmd.Flags().BoolVar(&includeHidden, "include-hidden", false, "include VISIBILITY=3 groups")
	return cmd
}
