package main

import (
	"context"
	"fmt"
	"os"

	"github.com/gridsociety/mythy/pkg/configio"
	"github.com/gridsociety/mythy/pkg/session"
	"github.com/spf13/cobra"
)

func newExportCmd(cf *catalogFlags) *cobra.Command {
	var conn connFlags
	var scope string
	var includeHidden, includeReadOnly, includeSkip, includeAll, noProgress bool

	cmd := &cobra.Command{
		Use:   "export <file>",
		Short: "Read the device's settings into a YAML config file",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			s, err := conn.build(ctx, cf)
			if err != nil {
				return err
			}
			defer s.Close()

			var progress func(done, total int, name string)
			if !noProgress {
				progress = makeReadProgress(cmd.ErrOrStderr(), false)
			}

			if includeAll {
				includeHidden = true
				includeReadOnly = true
				includeSkip = true
			}
			b, err := configio.Export(ctx, s, configio.ExportOptions{
				Scope: scope,
				Filter: session.ExportFilter{
					IncludeHidden:   includeHidden,
					IncludeReadOnly: includeReadOnly,
					IncludeSkip:     includeSkip,
				},
				Progress: progress,
			})
			if err != nil {
				return err
			}
			if err := os.WriteFile(args[0], b, 0o644); err != nil {
				return fmt.Errorf("write %s: %w", args[0], err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "wrote %s (%d bytes)\n", args[0], len(b))
			return nil
		},
	}
	conn.bind(cmd)
	cmd.Flags().StringVar(&scope, "scope", "", "menu path to export (default: whole device)")
	cmd.Flags().BoolVar(&includeHidden, "include-hidden", false, "include VISIBILITY=3 (Administrator) groups")
	cmd.Flags().BoolVar(&includeReadOnly, "include-readonly", false, "include READONLY=YES entries")
	cmd.Flags().BoolVar(&includeSkip, "include-skip", false, "include SKIP=YES entries (identification, IP/comm config)")
	cmd.Flags().BoolVar(&includeAll, "all", false, "shorthand for --include-hidden --include-readonly --include-skip")
	cmd.Flags().BoolVar(&noProgress, "no-progress", false, "suppress the progress indicator (auto-suppressed when stderr isn't a TTY)")
	return cmd
}
