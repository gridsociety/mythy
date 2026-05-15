package main

import (
	"fmt"
	"io"

	"github.com/gridsociety/mythy/pkg/catalog"
	"github.com/spf13/cobra"
)

func newDescribeCmd(cf *catalogFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "describe <name>",
		Short: "Print everything the catalog knows about one entry",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			tpl, _, err := cf.load()
			if err != nil {
				return err
			}
			d, group := findData(tpl, args[0])
			if d == nil {
				return fmt.Errorf("not found: %q", args[0])
			}
			return printDescribe(cmd.OutOrStdout(), tpl, d, group)
		},
	}
}

func findData(tpl *catalog.Template, name string) (*catalog.Data, *catalog.Group) {
	if tpl.Menu == nil {
		return nil, nil
	}
	for _, g := range tpl.Menu.WalkGroups(catalog.WalkOptions{IncludeHidden: true}) {
		for _, d := range g.Data {
			if d.Name == name {
				return d, g
			}
		}
	}
	return nil, nil
}

func printDescribe(w io.Writer, tpl *catalog.Template, d *catalog.Data, g *catalog.Group) error {
	fmt.Fprintf(w, "name: %s\n", d.Name)
	if d.Description != "" {
		fmt.Fprintf(w, "description: %s\n", d.Description)
	}
	fmt.Fprintf(w, "path: %s\n", g.Path())
	if t := d.DisplayTipo(); t != "" {
		fmt.Fprintf(w, "tipo: %s\n", t)
	}
	if d.Default != "" {
		fmt.Fprintf(w, "default: %s\n", d.Default)
	}
	if d.Valore != "" {
		fmt.Fprintf(w, "snapshot: %s\n", d.Valore)
	}
	if d.Message != nil {
		fmt.Fprintf(w, "wire: FC=%d  addr=0x%04X(%d)  qty=%d  type=%s\n",
			d.Message.FC(), d.Message.WireAddr(), d.Message.WireAddr(),
			d.Message.Dim, d.Message.Type)
		if d.Message.Level != "" {
			fmt.Fprintf(w, "level: %s\n", d.Message.Level)
		}
	}
	if d.Module != "" {
		fmt.Fprintf(w, "module: %s\n", d.Module)
	}
	if d.ReadOnly {
		fmt.Fprintln(w, "readonly: yes")
	}
	if d.Restart {
		fmt.Fprintln(w, "restart: required")
	}
	if d.Range != nil {
		if d.Range.Unit != "" {
			fmt.Fprintf(w, "range: [%d, %d] step=%d  unit=%s  decimals=%d  scale=%d\n",
				d.Range.Min, d.Range.Max, d.Range.Step, d.Range.Unit, d.Range.Decimals, d.Range.Scale)
		} else {
			fmt.Fprintf(w, "range: [%d, %d] step=%d\n", d.Range.Min, d.Range.Max, d.Range.Step)
		}
	}
	if d.Info != nil {
		fmt.Fprintf(w, "info: unit=%s  decimals=%d  scale=%g\n", d.Info.Unit, d.Info.Decimals, d.Info.Scale)
	}
	if d.InfoVis != nil {
		fmt.Fprintf(w, "visible-when: %s\n", d.InfoVis.Select)
	}
	if d.Enum != "" {
		fmt.Fprintf(w, "enum: %s\n", d.Enum)
		if e := tpl.Enums[d.Enum]; e != nil {
			for _, ent := range e.Entries {
				fmt.Fprintf(w, "  %d=%s\n", ent.Value, ent.Label)
			}
		}
	}
	return nil
}
