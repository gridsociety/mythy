package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/gridsociety/mythy/pkg/configio"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// renderableChange is the per-row shape used by the human / json / yaml renderers.
type renderableChange struct {
	Name    string `json:"name"     yaml:"name"`
	Path    string `json:"path"     yaml:"path"`
	Current any    `json:"current"  yaml:"current"`
	File    any    `json:"file"     yaml:"file"`
}

func newDiffCmd(cf *catalogFlags) *cobra.Command {
	var conn connFlags
	var format string
	var noProgress bool
	var includeAll bool
	var forceLocale bool

	cmd := &cobra.Command{
		Use:   "diff <file>",
		Short: "Compare the live device against a YAML config file",
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
			if err := reconcileLocale(cmd, parsed.Device.Locale, cf, forceLocale); err != nil {
				return err
			}
			s, err := conn.build(ctx, cf)
			if err != nil {
				return err
			}
			defer s.Close()
			if err := configio.Validate(parsed, s.Template(), s.Entry().Product); err != nil {
				return err
			}
			var progress func(done, total int, name string)
			if !noProgress {
				progress = makeReadProgress(cmd.ErrOrStderr(), false)
			}
			changes, err := configio.Diff(ctx, s, parsed, configio.DiffOptions{
				Progress:   progress,
				IncludeAll: includeAll,
			})
			if err != nil {
				return err
			}
			rows := make([]renderableChange, len(changes))
			for i, c := range changes {
				rows[i] = renderableChange(c)
			}
			return renderDiff(cmd.OutOrStdout(), rows, resolveFormat(format))
		},
	}
	conn.bind(cmd)
	cmd.Flags().StringVar(&format, "format", "", "human|json|yaml|unified (default: from MYTHY_FORMAT or human)")
	cmd.Flags().BoolVar(&noProgress, "no-progress", false, "suppress the progress indicator (auto-suppressed when stderr isn't a TTY)")
	cmd.Flags().BoolVar(&includeAll, "all", false,
		"include runtime/state items (READONLY=YES, paths under Read/) — defaults to filtered to surface only configuration drift")
	cmd.Flags().BoolVar(&forceLocale, "force-locale", false, "proceed even if --locale differs from the YAML's device.locale")
	return cmd
}

func renderDiff(w io.Writer, rows []renderableChange, format string) error {
	if len(rows) == 0 {
		fmt.Fprintln(w, "no differences")
		return nil
	}
	switch format {
	case "json":
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(rows)
	case "yaml":
		return yaml.NewEncoder(w).Encode(rows)
	case "unified":
		for _, r := range rows {
			fmt.Fprintf(w, "--- %s/%s\n+++ %s/%s\n-%v\n+%v\n", r.Path, r.Name, r.Path, r.Name, r.Current, r.File)
		}
		return nil
	}
	fmt.Fprint(w, renderDiffTable(rows))
	return nil
}

func renderDiffTable(rows []renderableChange) string {
	var b strings.Builder
	fmt.Fprintln(&b, "PATH                       NAME              CURRENT          FILE")
	for _, r := range rows {
		fmt.Fprintf(&b, "%-26s %-17s %-16v %v\n", r.Path, r.Name, r.Current, r.File)
	}
	fmt.Fprintf(&b, "%d differences\n", len(rows))
	return b.String()
}

// resolveFormat is a stub for Task 11's global format flag. Until that
// lands, it consults --format then $MYTHY_FORMAT.
func resolveFormat(s string) string {
	if s != "" {
		return s
	}
	return os.Getenv("MYTHY_FORMAT")
}
