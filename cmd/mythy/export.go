package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/gridsociety/mythy/pkg/configio"
	"github.com/gridsociety/mythy/pkg/session"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func newExportCmd(cf *catalogFlags) *cobra.Command {
	var conn connFlags
	var scope string
	var includeHidden, includeReadOnly, noProgress bool

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
			if !noProgress && term.IsTerminal(int(os.Stderr.Fd())) {
				progress = makeExportProgress(cmd.ErrOrStderr())
			}

			b, err := configio.Export(ctx, s, configio.ExportOptions{
				Scope: scope,
				Filter: session.ExportFilter{
					IncludeHidden:   includeHidden,
					IncludeReadOnly: includeReadOnly,
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
	cmd.Flags().BoolVar(&noProgress, "no-progress", false, "suppress the progress indicator (auto-suppressed when stderr isn't a TTY)")
	return cmd
}

// makeExportProgress returns a progress reporter that overwrites a
// single line on the writer ~10× per second using \r. The first call
// is always shown so the user gets immediate feedback; subsequent
// updates are throttled. The final sentinel call (name == "" and
// done == total) clears the line so subsequent output doesn't trail
// the progress text.
func makeExportProgress(w io.Writer) func(done, total int, name string) {
	const (
		throttle = 100 * time.Millisecond
		nameW    = 40
		lineW    = 60
	)
	var lastUpdate time.Time
	return func(done, total int, name string) {
		// Final / clear-line call.
		if name == "" && done >= total {
			fmt.Fprintf(w, "\r%-*s\r", lineW, "")
			return
		}
		if !lastUpdate.IsZero() && time.Since(lastUpdate) < throttle && done > 0 {
			return
		}
		lastUpdate = time.Now()
		shown := name
		if len(shown) > nameW {
			shown = shown[:nameW-3] + "..."
		}
		fmt.Fprintf(w, "\r[%5d/%5d] %-*s", done+1, total, nameW, shown)
	}
}
