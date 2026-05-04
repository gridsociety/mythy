package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func newNetSetCmd(cf *catalogFlags) *cobra.Command {
	var conn connFlags
	var ip, netmask, gateway string
	cmd := &cobra.Command{
		Use:   "net-set --ip … --netmask … --gateway …",
		Short: "Set Ethernet IPv4 parameters (SET_PARAMS_ETH0)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if ip == "" || netmask == "" || gateway == "" {
				return fmt.Errorf("net-set requires --ip, --netmask, --gateway")
			}
			ctx := context.Background()
			s, err := conn.build(ctx, cf)
			if err != nil {
				return err
			}
			defer s.Close()

			// Each field is 15 chars (8 regs); total dim = 24 regs.
			payload := append(append(encodeFixedString(ip, 8), encodeFixedString(netmask, 8)...), encodeFixedString(gateway, 8)...)
			if err := writeMultiByName(ctx, s, "SET_PARAMS_ETH0", payload); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "ip=%s netmask=%s gateway=%s\n", ip, netmask, gateway)
			fmt.Fprintln(cmd.OutOrStdout(), "(reboot required for changes to apply)")
			return nil
		},
	}
	conn.bind(cmd)
	cmd.Flags().StringVar(&ip, "ip", "", "IPv4 address")
	cmd.Flags().StringVar(&netmask, "netmask", "", "Netmask")
	cmd.Flags().StringVar(&gateway, "gateway", "", "Gateway")
	return cmd
}

func encodeFixedString(s string, regs int) []uint16 {
	maxLen := regs * 2
	if len(s) > maxLen {
		s = s[:maxLen]
	}
	s = strings.TrimRight(s, "\x00")
	out := make([]uint16, regs)
	for i := 0; i < len(s); i += 2 {
		hi := byte(s[i])
		var lo byte
		if i+1 < len(s) {
			lo = byte(s[i+1])
		}
		out[i/2] = uint16(hi)<<8 | uint16(lo)
	}
	return out
}
