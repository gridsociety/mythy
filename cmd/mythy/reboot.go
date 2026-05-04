package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

func newRebootCmd(cf *catalogFlags) *cobra.Command {
	var conn connFlags
	var noWait bool
	cmd := &cobra.Command{
		Use:   "reboot",
		Short: "Restart the device (= G61850 RestartDevice)",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			s, err := conn.build(ctx, cf)
			if err != nil {
				return err
			}
			defer s.Close()

			res, err := s.Reboot(ctx, !noWait)
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			fmt.Fprintln(out, "reboot triggered")
			if !noWait {
				fmt.Fprintf(out, "device returned after %s\n", res.Outage)
			}
			return nil
		},
	}
	conn.bind(cmd)
	cmd.Flags().BoolVar(&noWait, "no-wait", false, "fire and forget; don't poll for return")
	return cmd
}
