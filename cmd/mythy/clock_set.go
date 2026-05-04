package main

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

func newClockSetCmd(cf *catalogFlags) *cobra.Command {
	var conn connFlags
	var atRFC string
	cmd := &cobra.Command{
		Use:   "clock-set",
		Short: "Set the device RTC (default: now)",
		RunE: func(cmd *cobra.Command, args []string) error {
			t := time.Now()
			if atRFC != "" {
				parsed, err := time.Parse(time.RFC3339, atRFC)
				if err != nil {
					return fmt.Errorf("--at: %w", err)
				}
				t = parsed
			}
			ctx := context.Background()
			s, err := conn.build(ctx, cf)
			if err != nil {
				return err
			}
			defer s.Close()

			// SET_RTC is WREG num=5169 dim=6 with parts day/month/year/hour/minute/second.
			payload := []uint16{
				uint16(t.Day()), uint16(t.Month()), uint16(t.Year() % 100),
				uint16(t.Hour()), uint16(t.Minute()), uint16(t.Second()),
			}
			if err := writeMultiByName(ctx, s, "SET_RTC", payload); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "RTC set to %s\n", t.Format(time.RFC3339))
			return nil
		},
	}
	conn.bind(cmd)
	cmd.Flags().StringVar(&atRFC, "at", "", "RFC3339 timestamp (default: now)")
	return cmd
}
