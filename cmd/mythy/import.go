package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/gridsociety/mythy/pkg/configio"
	"github.com/spf13/cobra"
)

func newImportCmd(cf *catalogFlags) *cobra.Command {
	var conn connFlags
	var dryRun, force, noProgress bool
	var format string

	cmd := &cobra.Command{
		Use:   "import <file>",
		Short: "Apply a YAML config file to the device (one edit transaction)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			b, err := os.ReadFile(args[0])
			if err != nil {
				return err
			}
			parsed, err := configio.Parse(b)
			if err != nil {
				return err
			}
			s, err := conn.build(ctx, cf)
			if err != nil {
				return err
			}
			defer s.Close()

			if err := configio.Validate(parsed, s.Template(), s.Entry().Product); err != nil {
				var pm *configio.ProductMismatchError
				if errors.As(err, &pm) && force {
					fmt.Fprintf(cmd.OutOrStdout(), "warning: %s (--force, continuing)\n", pm)
				} else {
					return err
				}
			}

			var progress func(done, total int, name string)
			if !noProgress {
				progress = makeReadProgress(cmd.ErrOrStderr(), false)
			}
			applyOpts := configio.ApplyOptions{Progress: progress}

			out := cmd.OutOrStdout()
			if dryRun {
				report, err := configio.ApplyDryRun(ctx, s, parsed, applyOpts)
				if err != nil {
					return err
				}
				fmt.Fprintf(out, "would write %d key(s): %v\n", len(report.WouldApply), report.WouldApply)
				if len(report.Skipped) > 0 {
					fmt.Fprintf(out, "would skip %d READONLY key(s): %v\n", len(report.Skipped), report.Skipped)
				}
				return nil
			}
			report, err := configio.Apply(ctx, s, parsed, applyOpts)
			if err != nil {
				return err
			}
			fmt.Fprintf(out, "wrote %d key(s): %v\n", len(report.Applied), report.Applied)
			if len(report.Skipped) > 0 {
				fmt.Fprintf(out, "skipped %d READONLY key(s) the file changed: %v\n", len(report.Skipped), report.Skipped)
			}
			return nil
		},
	}
	conn.bind(cmd)
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "show what would change; perform no writes")
	cmd.Flags().BoolVar(&force, "force", false, "skip the product-mismatch check")
	cmd.Flags().BoolVar(&noProgress, "no-progress", false, "suppress the progress indicator (auto-suppressed when stderr isn't a TTY)")
	cmd.Flags().StringVar(&format, "format", "", "human|json|yaml (default: from MYTHY_FORMAT)")
	return cmd
}
