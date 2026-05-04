package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

func newIdentifyCmd(cf *catalogFlags) *cobra.Command {
	var conn connFlags
	cmd := &cobra.Command{
		Use:   "identify",
		Short: "Connect to a device, run discovery, print identification + secure-mode state",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			s, err := conn.build(ctx, cf)
			if err != nil {
				return err
			}
			defer s.Close()

			id := s.Ident()
			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "Product:           %s\n", s.Entry().Product)
			fmt.Fprintf(out, "Identification:    %d\n", id.Identification)
			fmt.Fprintf(out, "SerialNumber:     %d\n", id.SerialNumber)
			fmt.Fprintf(out, "FwRelease:         0x%04X\n", id.FwRelease)
			fmt.Fprintf(out, "ProtocolRelease:   0x%04X\n", id.ProtocolRelease)
			fmt.Fprintf(out, "Family:            %s\n", s.Entry().Family)
			fmt.Fprintf(out, "Template revision: %s\n", deriveRevision(s.Entry().Product))
			fmt.Fprintf(out, "Authentication:    %s\n", authStatus(s))
			return nil
		},
	}
	conn.bind(cmd)
	return cmd
}

// deriveRevision pulls the trailing "-X" off "PROX-VX0-e".
func deriveRevision(product string) string {
	for i := len(product) - 1; i > 0; i-- {
		if product[i] == '-' {
			return product[i+1:]
		}
	}
	return ""
}

// authStatus uses the catalog (capability) plus the live read (state).
func authStatus(s interface{ SecureMode() bool }) string {
	if s.SecureMode() {
		return "ON (mythy v1 has no auth flow — escalate via ThyVisor first)"
	}
	return "OFF"
}
