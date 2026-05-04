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
			if scope != "" {
				vals, err := s.ReadScope(ctx, scope, includeHidden)
				if err != nil {
					return err
				}
				for k, v := range vals {
					fmt.Fprintf(out, "%-30s %s\n", k, v.Format())
				}
				return nil
			}
			if len(args) == 0 {
				return fmt.Errorf("read: provide names or --scope")
			}
			for _, name := range args {
				v, err := s.Read(ctx, name)
				if err != nil {
					return err
				}
				fmt.Fprintf(out, "%-30s %s\n", name, v.Format())
			}
			return nil
		},
	}
	conn.bind(cmd)
	cmd.Flags().StringVar(&scope, "scope", "", "menu path to read recursively")
	cmd.Flags().BoolVar(&includeHidden, "include-hidden", false, "include VISIBILITY=3 groups")
	return cmd
}
