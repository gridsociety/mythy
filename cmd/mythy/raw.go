package main

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/gridsociety/mythy/pkg/session"
	"github.com/spf13/cobra"
)

// writeMultiByName looks up a WREG message by name and writes a register
// payload. Used by clock-set, net-set, etc. — commands that drive WREG
// payloads without going through SetMany.
//
// Audit C7: when the message's CLASS ends in "_PARAM" (persistent flash),
// the write is wrapped in a one-shot edit transaction (START_CHANGE_DB
// / FC16 / END_CHANGE_DB). *_RAM-class messages and unclassified
// trigger-style WREGs go through directly — same rule as SetMany
// (Plan 2 Task 15).
func writeMultiByName(ctx context.Context, s *session.Session, name string, regs []uint16) error {
	tpl := s.Template()
	msg, ok := tpl.Messages[name]
	if !ok {
		return fmt.Errorf("message %q not in catalog", name)
	}
	if msg.Type != "WREG" {
		return fmt.Errorf("message %q is %s, expected WREG", name, msg.Type)
	}
	if len(regs) != msg.Dim {
		return fmt.Errorf("%s expects dim=%d, got %d", name, msg.Dim, len(regs))
	}
	addr := uint16(msg.WireAddr())
	if strings.HasSuffix(msg.Class, "_PARAM") {
		tx, err := s.BeginEdit(ctx)
		if err != nil {
			return err
		}
		if err := s.WriteMultiRaw(ctx, addr, regs); err != nil {
			_ = tx.Close(ctx)
			return err
		}
		return tx.Commit(ctx)
	}
	return s.WriteMultiRaw(ctx, addr, regs)
}

func newRawCmd(cf *catalogFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "raw",
		Short: "Bypass the catalog and issue raw Modbus FC{3,4,6,16} requests",
	}
	cmd.AddCommand(newRawReadCmd(cf))
	cmd.AddCommand(newRawWriteCmd(cf))
	return cmd
}

func newRawReadCmd(cf *catalogFlags) *cobra.Command {
	var conn connFlags
	var fc int
	var addr uint16
	var qty uint16
	cmd := &cobra.Command{
		Use:   "read --fc 3|4 --addr N [--qty K]",
		Short: "Raw FC03/FC04 read",
		RunE: func(cmd *cobra.Command, args []string) error {
			if qty == 0 {
				qty = 1
			}
			ctx := context.Background()
			s, err := conn.build(ctx, cf)
			if err != nil {
				return err
			}
			defer s.Close()

			var regs []uint16
			switch fc {
			case 3:
				regs, err = s.ReadHoldingRaw(ctx, addr, qty)
			case 4:
				regs, err = s.ReadInputRaw(ctx, addr, qty)
			default:
				return fmt.Errorf("--fc must be 3 or 4")
			}
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			for i, r := range regs {
				fmt.Fprintf(out, "0x%04X: 0x%04X (%d)\n", int(addr)+i, r, r)
			}
			return nil
		},
	}
	conn.bind(cmd)
	cmd.Flags().IntVar(&fc, "fc", 4, "function code (3=holding, 4=input)")
	cmd.Flags().Uint16Var(&addr, "addr", 0, "wire address (0-based)")
	cmd.Flags().Uint16Var(&qty, "qty", 1, "register count")
	return cmd
}

func newRawWriteCmd(cf *catalogFlags) *cobra.Command {
	var conn connFlags
	var fc int
	var addr uint16
	var value string
	cmd := &cobra.Command{
		Use:   "write --fc 6|16 --addr N --value V[,V…]",
		Short: "Raw FC06/FC16 write",
		RunE: func(cmd *cobra.Command, args []string) error {
			parts := strings.Split(value, ",")
			vals := make([]uint16, 0, len(parts))
			for _, p := range parts {
				p = strings.TrimSpace(p)
				n, err := strconv.ParseUint(p, 0, 16)
				if err != nil {
					return fmt.Errorf("--value: %q is not a uint16: %w", p, err)
				}
				vals = append(vals, uint16(n))
			}
			ctx := context.Background()
			s, err := conn.build(ctx, cf)
			if err != nil {
				return err
			}
			defer s.Close()

			switch fc {
			case 6:
				if len(vals) != 1 {
					return fmt.Errorf("--fc 6 takes exactly one value")
				}
				return s.WriteSingleRaw(ctx, addr, vals[0])
			case 16:
				return s.WriteMultiRaw(ctx, addr, vals)
			}
			return fmt.Errorf("--fc must be 6 or 16")
		},
	}
	conn.bind(cmd)
	cmd.Flags().IntVar(&fc, "fc", 6, "function code (6 single, 16 multi)")
	cmd.Flags().Uint16Var(&addr, "addr", 0, "wire address (0-based)")
	cmd.Flags().StringVar(&value, "value", "", "comma-separated uint16 values (decimal or 0x…)")
	return cmd
}
