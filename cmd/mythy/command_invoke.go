package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

func newCommandInvokeCmd(cf *catalogFlags) *cobra.Command {
	var conn connFlags
	cmd := &cobra.Command{
		Use:   "invoke <name>",
		Short: "Invoke a <COMMAND> on the device (e.g. MSG_CMD_RESET_DA_PC)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			s, err := conn.build(ctx, cf)
			if err != nil {
				return err
			}
			defer s.Close()
			if err := s.Command(ctx, args[0], nil); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%s: ok\n", args[0])
			return nil
		},
	}
	conn.bind(cmd)
	return cmd
}
