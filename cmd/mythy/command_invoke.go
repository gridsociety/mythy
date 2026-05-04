package main

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

func newCommandInvokeCmd(cf *catalogFlags) *cobra.Command {
	var conn connFlags
	var argFlags []string
	cmd := &cobra.Command{
		Use:   "invoke <name>",
		Short: "Invoke a <COMMAND> on the device (e.g. MSG_CMD_RESET_DA_PC, SET_RTC)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			parsedArgs, err := parseArgFlags(argFlags)
			if err != nil {
				return err
			}
			ctx := context.Background()
			s, err := conn.build(ctx, cf)
			if err != nil {
				return err
			}
			defer s.Close()
			if err := s.Command(ctx, args[0], parsedArgs); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%s: ok\n", args[0])
			return nil
		},
	}
	conn.bind(cmd)
	cmd.Flags().StringArrayVar(&argFlags, "arg", nil, "structured argument: --arg name=value (repeatable)")
	return cmd
}

func parseArgFlags(in []string) (map[string]any, error) {
	out := make(map[string]any, len(in))
	for _, a := range in {
		i := strings.IndexByte(a, '=')
		if i <= 0 {
			return nil, fmt.Errorf("--arg expects name=value, got %q", a)
		}
		k, v := a[:i], a[i+1:]
		if u, err := strconv.ParseUint(v, 10, 64); err == nil {
			out[k] = u
			continue
		}
		out[k] = strings.Trim(v, `"`)
	}
	return out, nil
}
