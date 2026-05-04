package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

var resetTargets = map[string]string{
	"faults":   "MSG_RESET_GUASTI",
	"counters": "MSG_RESET_CONTATORI_AZZERABILI",
	"measures": "MSG_CMD_RESET_MEAN_MEASURE_DA_PC",
	"defaults": "SET_DB_DEFAULT",
}

func newResetCmd(cf *catalogFlags) *cobra.Command {
	var conn connFlags
	var force bool
	cmd := &cobra.Command{
		Use:       "reset {faults|counters|measures|defaults}",
		Short:     "Trigger one of the canonical reset commands",
		Args:      cobra.ExactArgs(1),
		ValidArgs: []string{"faults", "counters", "measures", "defaults"},
		RunE: func(cmd *cobra.Command, args []string) error {
			cmdName, ok := resetTargets[args[0]]
			if !ok {
				return fmt.Errorf("unknown reset target %q (valid: faults|counters|measures|defaults)", args[0])
			}
			if args[0] == "defaults" && !force {
				return fmt.Errorf("'reset defaults' restores factory settings — pass --force to confirm")
			}
			ctx := context.Background()
			s, err := conn.build(ctx, cf)
			if err != nil {
				return err
			}
			defer s.Close()
			if err := s.Command(ctx, cmdName, nil); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%s: ok\n", cmdName)
			return nil
		},
	}
	conn.bind(cmd)
	cmd.Flags().BoolVar(&force, "force", false, "required for 'reset defaults'")
	return cmd
}
