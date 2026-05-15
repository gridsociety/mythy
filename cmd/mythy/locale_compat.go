package main

import (
	"github.com/gridsociety/mythy/pkg/configio"
	"github.com/spf13/cobra"
)

// reconcileLocale picks the effective --locale for a command that
// consumes a YAML file (import / diff / validate). The decision is:
//
//   - no locale in YAML (legacy export): keep cf.locale unchanged.
//   - locale in YAML, user didn't pass --locale explicitly: adopt
//     the YAML's locale into cf.locale.
//   - locale in YAML, user passed --locale matching: keep cf.locale.
//   - locale in YAML, user passed --locale differing: return a typed
//     LocaleMismatchError unless --force-locale is set.
//
// The flag-was-passed check uses cobra's Changed(): the locale flag
// defaults to "en" (or MYTHY_LOCALE), and we want to distinguish "user
// typed --locale=en" from "user typed nothing and en is the default".
// "Changed" reports true only for explicit invocation.
//
// Reconciliation mutates cf.locale in place. Callers can rely on
// cf.load() / connFlags.build() using the resolved value transparently.
func reconcileLocale(cmd *cobra.Command, fileLocale string, cf *catalogFlags, forceLocale bool) error {
	if fileLocale == "" {
		return nil
	}
	explicit := cmd.Flags().Changed("locale")
	if !explicit {
		cf.locale = fileLocale
		return nil
	}
	if cf.locale == fileLocale {
		return nil
	}
	if forceLocale {
		return nil
	}
	return &configio.LocaleMismatchError{FromFile: fileLocale, FromCLI: cf.locale}
}
