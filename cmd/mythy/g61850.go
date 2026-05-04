package main

import (
	"context"
	"fmt"

	"github.com/gridsociety/mythy/pkg/session"
	"github.com/spf13/cobra"
)

func newG61850Cmd(cf *catalogFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "g61850",
		Short: "Inspect or invoke the device's G61850 parser",
	}
	cmd.AddCommand(newG61850ListCmd(cf))
	cmd.AddCommand(newG61850InvokeCmd(cf))
	return cmd
}

func newG61850ListCmd(cf *catalogFlags) *cobra.Command {
	return &cobra.Command{
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
	}
}

func newG61850InvokeCmd(cf *catalogFlags) *cobra.Command {
	var conn connFlags
	var par1, par2 string
	var force bool
	cmd := &cobra.Command{
		Use:   "invoke <function>",
		Short: "Invoke a Gst61850_Msg function (Get* / Set* / RestartDevice)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			fn := args[0]
			// Destructive functions (WriteCid, ResetCid, ResetAll) need --force.
			// session.IsDestructive is the single source of truth; the session
			// layer doesn't refuse these calls itself.
			if session.IsDestructive(fn) && !force {
				return fmt.Errorf("g61850 %s is destructive; pass --force to acknowledge", fn)
			}
			ctx := context.Background()
			s, err := conn.build(ctx, cf)
			if err != nil {
				return err
			}
			defer s.Close()
			reply, err := s.G61850(ctx, fn, par1, par2)
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			if reply != "" {
				fmt.Fprintln(out, reply)
			} else {
				fmt.Fprintf(out, "%s: ok\n", fn)
			}
			return nil
		},
	}
	conn.bind(cmd)
	cmd.Flags().StringVar(&par1, "par1", "", "first parameter (ASCII string)")
	cmd.Flags().StringVar(&par2, "par2", "", "second parameter (ASCII string)")
	cmd.Flags().BoolVar(&force, "force", false, "allow destructive functions (WriteCid, ResetCid, ResetAll)")
	return cmd
}
