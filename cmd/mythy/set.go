package main

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/gridsociety/mythy/pkg/session"
	"github.com/spf13/cobra"
)

func newSetCmd(cf *catalogFlags) *cobra.Command {
	var conn connFlags
	cmd := &cobra.Command{
		Use:   "set <name>=<value> [<name>=<value> ...]",
		Short: "Write one or more parameters in a single edit transaction",
		Long: `Write parameters to the device. Each argument is name=value.

Compound types (TIMER, RELE, LED, CONTATORE, …) can be addressed by
sub-field with dotted-path syntax: name.subfield=value. Multiple
sub-fields of the same compound coalesce into a single read-modify-
write so the unmodified sub-fields are preserved:

  mythy set RELE_K1.Logica=De-energized
  mythy set RELE_K1.Modo=NormalOpen RELE_K1.Logica=Energized
  mythy set MB_address=5 EnF81_TSc.Valore=2000

The write happens inside one edit transaction (auto-bundled).`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			pairs, err := parseSetArgs(args)
			if err != nil {
				return err
			}
			s, err := conn.build(ctx, cf)
			if err != nil {
				return err
			}
			defer s.Close()

			expanded, err := expandCompoundMutations(ctx, s, pairs)
			if err != nil {
				return err
			}

			if err := s.SetMany(ctx, expanded); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "wrote %d parameter(s)\n", len(args))
			return nil
		},
	}
	conn.bind(cmd)
	return cmd
}

// parseSetArgs converts "name=value" tokens into typed map entries.
// Type discovery happens server-side; here we just heuristically parse
// the value as int / uint / leave-as-string. Quoted strings keep quotes.
// Dotted keys (name.subfield) are stored as-is and expanded by
// expandCompoundMutations once we have a Session to read the current
// compound from.
func parseSetArgs(args []string) (map[string]any, error) {
	out := make(map[string]any, len(args))
	for _, a := range args {
		i := strings.IndexByte(a, '=')
		if i <= 0 {
			return nil, fmt.Errorf("expected name=value, got %q", a)
		}
		k, v := a[:i], a[i+1:]
		out[k] = parseValue(v)
	}
	return out, nil
}

func parseValue(v string) any {
	if u, err := strconv.ParseUint(v, 10, 64); err == nil {
		return u
	}
	if i, err := strconv.ParseInt(v, 10, 64); err == nil {
		return i
	}
	return strings.Trim(v, `"`)
}

// expandCompoundMutations rewrites dotted keys (name.subfield) into a
// single full-compound entry per name, by reading the current compound
// from the device and merging the user-supplied sub-field changes. The
// unmodified sub-fields are preserved so the FC16 write doesn't blow
// them away.
//
// Plain (non-dotted) keys pass through unchanged.
func expandCompoundMutations(ctx context.Context, s *session.Session, args map[string]any) (map[string]any, error) {
	plain := make(map[string]any, len(args))
	mutations := map[string]map[string]any{}
	for k, v := range args {
		if dot := strings.IndexByte(k, '.'); dot > 0 {
			name, sub := k[:dot], k[dot+1:]
			if _, ok := mutations[name]; !ok {
				mutations[name] = map[string]any{}
			}
			mutations[name][sub] = v
		} else {
			plain[k] = v
		}
	}
	if len(mutations) == 0 {
		return plain, nil
	}

	tpl := s.Template()
	for name, subMutations := range mutations {
		current, err := s.Read(ctx, name)
		if err != nil {
			return nil, fmt.Errorf("set %s.…: read current value: %w", name, err)
		}
		if current.Compound == nil {
			return nil, fmt.Errorf("set %s.…: %s is not a compound type (TIPO=%s)", name, name, current.Tipo)
		}
		cls, ok := tpl.Classes[current.Tipo]
		if !ok {
			return nil, fmt.Errorf("set %s: TIPO=%s not in catalog Classes", name, current.Tipo)
		}
		validSub := make(map[string]bool, len(cls.Vars))
		for _, v := range cls.Vars {
			validSub[v.Name] = true
		}
		full := make(map[string]any, len(current.Compound))
		for k, sub := range current.Compound {
			full[k] = subValueToCodecAny(sub)
		}
		for sub, v := range subMutations {
			if !validSub[sub] {
				return nil, fmt.Errorf("set %s.%s: %s is not a sub-field of %s", name, sub, sub, current.Tipo)
			}
			full[sub] = v
		}
		plain[name] = full
	}
	return plain, nil
}

// subValueToCodecAny converts a session.Value sub-field (as returned
// by Session.Read for a compound) back into the form codec.EncodeCompound
// expects: uint64 for unsigned numerics, int64 for signed numerics,
// the resolved label string (or numeric fallback) for ENUM*, the
// literal Go string for STRING.
func subValueToCodecAny(v session.Value) any {
	switch v.Tipo {
	case "ENUM", "ENUM_BYTE", "ENUM_LONG":
		if v.Label != "" {
			return v.Label
		}
		return v.Number
	case "STRING":
		return v.Str
	}
	if strings.HasPrefix(v.Tipo, "U") || v.Tipo == "BIT16" || v.Tipo == "BIT32" {
		return uint64(v.Number)
	}
	return v.Number
}
