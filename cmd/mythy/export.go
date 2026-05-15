package main

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/gridsociety/mythy/pkg/configio"
	"github.com/gridsociety/mythy/pkg/session"
	"github.com/spf13/cobra"
)

func newExportCmd(cf *catalogFlags) *cobra.Command {
	var conn connFlags
	var scope string
	var includeHidden, includeReadOnly, includeSkip, includeDisabledModules, includeAll, noProgress bool

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
				includeDisabledModules = true
			}
			report := &configio.ExportReport{}
			b, err := configio.Export(ctx, s, configio.ExportOptions{
				Scope:  scope,
				Locale: cf.locale,
				Filter: session.ExportFilter{
					IncludeHidden:          includeHidden,
					IncludeReadOnly:        includeReadOnly,
					IncludeSkip:            includeSkip,
					IncludeDisabledModules: includeDisabledModules,
				},
				Progress: progress,
				Report:   report,
			})
			if err != nil {
				return err
			}
			if err := os.WriteFile(args[0], b, 0o644); err != nil {
				return fmt.Errorf("write %s: %w", args[0], err)
			}
			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "wrote %s (%d bytes)\n", args[0], len(b))
			printExportSkipSummary(out, report)
			return nil
		},
	}
	conn.bind(cmd)
	cmd.Flags().StringVar(&scope, "scope", "", "menu path to export (default: whole device)")
	cmd.Flags().BoolVar(&includeHidden, "include-hidden", false, "include VISIBILITY=3 (Administrator) groups")
	cmd.Flags().BoolVar(&includeReadOnly, "include-readonly", false, "include READONLY=YES entries")
	cmd.Flags().BoolVar(&includeSkip, "include-skip", false, "include SKIP=YES entries (identification, IP/comm config)")
	cmd.Flags().BoolVar(&includeDisabledModules, "include-disabled-modules", false, "include DATA whose MODULE attribute is reported disabled by EnableBoard_*")
	cmd.Flags().BoolVar(&includeAll, "all", false, "shorthand for --include-hidden --include-readonly --include-skip --include-disabled-modules")
	cmd.Flags().BoolVar(&noProgress, "no-progress", false, "suppress the progress indicator (auto-suppressed when stderr isn't a TTY)")
	return cmd
}

// printExportSkipSummary emits one line per non-empty skip category
// so the operator can see why the exported YAML is smaller than the
// catalog suggests it should be. Mirrors the post-write summary
// already emitted by `mythy import`.
func printExportSkipSummary(out io.Writer, r *configio.ExportReport) {
	if r == nil {
		return
	}
	if n := len(r.SkippedReadOnly); n > 0 {
		fmt.Fprintf(out, "skipped %d READONLY=YES key(s) (use --include-readonly to include)\n", n)
	}
	if n := len(r.SkippedHidden); n > 0 {
		fmt.Fprintf(out, "skipped %d VISIBILITY=3 key(s) (use --include-hidden to include)\n", n)
	}
	if n := len(r.SkippedSkip); n > 0 {
		fmt.Fprintf(out, "skipped %d SKIP=YES key(s) (use --include-skip to include)\n", n)
	}
	if n := len(r.SkippedDisabledModules); n > 0 {
		fmt.Fprintf(out, "skipped %d module-disabled key(s) (use --include-disabled-modules to include)\n", n)
	}
}
