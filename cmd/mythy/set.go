package main

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

func newSetCmd(cf *catalogFlags) *cobra.Command {
	var conn connFlags
	cmd := &cobra.Command{
		Use:   "set <name>=<value> [<name>=<value> ...]",
		Short: "Write one or more parameters in a single edit transaction",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			pairs, err := parseSetArgs(args)
			if err != nil {
				return err
			}
			s, err := conn.build(ctx, cf)
			if err != nil {
				return err
			}
			defer s.Close()

			if err := s.SetMany(ctx, pairs); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "wrote %d parameter(s)\n", len(pairs))
			return nil
		},
	}
	conn.bind(cmd)
	return cmd
}

// parseSetArgs converts "name=value" tokens into typed map entries.
// Type discovery happens server-side; here we just heuristically parse
// the value as int / uint / leave-as-string. Quoted strings keep quotes.
func parseSetArgs(args []string) (map[string]any, error) {
	out := make(map[string]any, len(args))
	for _, a := range args {
		i := strings.IndexByte(a, '=')
		if i <= 0 {
			return nil, fmt.Errorf("expected name=value, got %q", a)
		}
		k, v := a[:i], a[i+1:]
		out[k] = parseValue(v)
	}
	return out, nil
}

func parseValue(v string) any {
	if u, err := strconv.ParseUint(v, 10, 64); err == nil {
		return u
	}
	if i, err := strconv.ParseInt(v, 10, 64); err == nil {
		return i
	}
	return strings.Trim(v, `"`)
}
