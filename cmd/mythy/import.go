package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/gridsociety/mythy/pkg/configio"
	"github.com/spf13/cobra"
)

func newImportCmd(cf *catalogFlags) *cobra.Command {
	var conn connFlags
	var dryRun, force, noProgress bool
	var includeSkip, includeHidden, includeAll, yesIUnderstand bool
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
			optIns := resolveScopeOptIns(includeSkip, includeHidden, includeAll)
			applyOpts := configio.ApplyOptions{
				Progress:      progress,
				IncludeSkip:   optIns.skip,
				IncludeHidden: optIns.hidden,
			}

			out := cmd.OutOrStdout()

			// Opt-in without --yes-i-understand → pre-flight dry-run to
			// surface exactly which keys would be touched in each opted-
			// in category, then refuse to proceed. The Diff cost is only
			// paid in this safety path; the no-opt-in default flow goes
			// straight to Apply with one Diff pass.
			if optIns.anyRequested() && !yesIUnderstand {
				report, err := configio.ApplyDryRun(ctx, s, parsed, applyOpts)
				if err != nil {
					return err
				}
				printOptInBanner(out, report, optIns, "would write (rerun with --yes-i-understand to proceed)")
				return errors.New("scope opt-in requires --yes-i-understand to proceed")
			}

			if dryRun {
				report, err := configio.ApplyDryRun(ctx, s, parsed, applyOpts)
				if err != nil {
					return err
				}
				if optIns.anyRequested() {
					printOptInBanner(out, report, optIns, "would write")
				}
				printDryRun(out, report, optIns)
				return nil
			}

			report, err := configio.Apply(ctx, s, parsed, applyOpts)
			if err != nil {
				return err
			}
			if optIns.anyRequested() {
				printOptInBanner(out, report, optIns, "WRITING")
			}
			fmt.Fprintf(out, "wrote %d key(s): %v\n", len(report.Applied), report.Applied)
			printSkipSummary(out, report, optIns)
			return nil
		},
	}
	conn.bind(cmd)
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "show what would change; perform no writes")
	cmd.Flags().BoolVar(&force, "force", false, "skip the product-mismatch check")
	cmd.Flags().BoolVar(&noProgress, "no-progress", false, "suppress the progress indicator (auto-suppressed when stderr isn't a TTY)")
	cmd.Flags().BoolVar(&includeSkip, "include-skip", false, "allow writes to SKIP=YES keys (identification, IP/comm config)")
	cmd.Flags().BoolVar(&includeHidden, "include-hidden", false, "allow writes to VISIBILITY=3 keys (Administrator subtree)")
	cmd.Flags().BoolVar(&includeAll, "all", false, "shorthand for --include-skip --include-hidden")
	cmd.Flags().BoolVar(&yesIUnderstand, "yes-i-understand", false, "confirm intent when an --include-* flag is set")
	cmd.Flags().StringVar(&format, "format", "", "human|json|yaml (default: from MYTHY_FORMAT)")
	return cmd
}

type scopeOptIns struct {
	skip   bool
	hidden bool
}

func (s scopeOptIns) anyRequested() bool { return s.skip || s.hidden }

func resolveScopeOptIns(skip, hidden, all bool) scopeOptIns {
	if all {
		return scopeOptIns{skip: true, hidden: true}
	}
	return scopeOptIns{skip: skip, hidden: hidden}
}

// printOptInBanner lists keys that fall under an opt-in flag, grouped
// by reason. Called for both the abort path (no --yes-i-understand)
// and the proceed path (with --yes-i-understand) so the operator
// always sees the full list before writes hit the device.
func printOptInBanner(out io.Writer, rep *configio.Report, optIns scopeOptIns, label string) {
	if optIns.skip && len(rep.InSkipCategory) > 0 {
		fmt.Fprintf(out, "%s %d connection-disruptive key(s) (SKIP=YES — identification, IP/comm config): %v\n",
			label, len(rep.InSkipCategory), rep.InSkipCategory)
	}
	if optIns.hidden && len(rep.InHiddenCategory) > 0 {
		fmt.Fprintf(out, "%s %d hidden key(s) (VISIBILITY=3 — Administrator subtree): %v\n",
			label, len(rep.InHiddenCategory), rep.InHiddenCategory)
	}
}

func printDryRun(out io.Writer, rep *configio.Report, optIns scopeOptIns) {
	fmt.Fprintf(out, "would write %d key(s): %v\n", len(rep.WouldApply), rep.WouldApply)
	if len(rep.Skipped) > 0 {
		fmt.Fprintf(out, "would skip %d READONLY key(s): %v\n", len(rep.Skipped), rep.Skipped)
	}
	if !optIns.skip && len(rep.InSkipCategory) > 0 {
		fmt.Fprintf(out, "would skip %d SKIP key(s) (use --include-skip to write): %v\n",
			len(rep.InSkipCategory), rep.InSkipCategory)
	}
	if !optIns.hidden && len(rep.InHiddenCategory) > 0 {
		fmt.Fprintf(out, "would skip %d hidden key(s) (use --include-hidden to write): %v\n",
			len(rep.InHiddenCategory), rep.InHiddenCategory)
	}
}

func printSkipSummary(out io.Writer, rep *configio.Report, optIns scopeOptIns) {
	if len(rep.Skipped) > 0 {
		fmt.Fprintf(out, "skipped %d READONLY key(s) the file changed: %v\n", len(rep.Skipped), rep.Skipped)
	}
	if !optIns.skip && len(rep.InSkipCategory) > 0 {
		fmt.Fprintf(out, "skipped %d SKIP key(s) (use --include-skip to include): %v\n",
			len(rep.InSkipCategory), rep.InSkipCategory)
	}
	if !optIns.hidden && len(rep.InHiddenCategory) > 0 {
		fmt.Fprintf(out, "skipped %d hidden key(s) (use --include-hidden to include): %v\n",
			len(rep.InHiddenCategory), rep.InHiddenCategory)
	}
}
