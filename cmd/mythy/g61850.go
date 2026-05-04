package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newG61850Cmd(cf *catalogFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "g61850",
		Short: "Inspect / invoke the device's G61850 parser (catalog-only in Plan 1)",
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "Print the device's Gst61850_Msg enum (supported parser functions)",
		RunE: func(cmd *cobra.Command, args []string) error {
			tpl, entry, err := cf.load()
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			// Audit D4: gate on the template (Gst61850_Msg enum + the
			// MSG_GST61850 trigger command), NOT on Codifica.xml's
			// IEC61850 flag — that flag is a strict superset (231 of
			// 842 templates have IEC61850=true but no parser enum).
			e, ok := tpl.Enums["Gst61850_Msg"]
			if !ok {
				fmt.Fprintf(out, "g61850 not supported: %s template has no Gst61850_Msg enum\n", entry.Product)
				return nil
			}
			if _, ok := tpl.Messages["MSG_GST61850"]; !ok {
				fmt.Fprintf(out, "g61850 not supported: %s template has no MSG_GST61850 command\n", entry.Product)
				return nil
			}
			if entry.IEC61850 {
				fmt.Fprintf(out, "G61850 functions supported by %s (Codifica IEC61850=true):\n", entry.Product)
			} else {
				fmt.Fprintf(out, "G61850 functions supported by %s (note: Codifica IEC61850=false but template carries the parser):\n", entry.Product)
			}
			for _, x := range e.Entries {
				fmt.Fprintf(out, "  %2d  %s\n", x.Value, x.Label)
			}
			return nil
		},
	})
	return cmd
}
