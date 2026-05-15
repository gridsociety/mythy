package main

import (
	"errors"
	"testing"

	"github.com/gridsociety/mythy/pkg/configio"
	"github.com/spf13/cobra"
)

// mkLocaleCmd sets up a minimal cobra.Command + catalogFlags with a
// --locale flag wired exactly as the real subcommands do (default
// "en", env MYTHY_LOCALE). args provides the CLI tokens after the
// command name; pass nothing for "user did not specify --locale".
func mkLocaleCmd(t *testing.T, args ...string) (*cobra.Command, *catalogFlags) {
	t.Helper()
	cf := &catalogFlags{}
	cmd := &cobra.Command{Use: "x", RunE: func(*cobra.Command, []string) error { return nil }}
	cmd.PersistentFlags().StringVar(&cf.locale, "locale", "en", "")
	cmd.SetArgs(args)
	if err := cmd.ParseFlags(args); err != nil {
		t.Fatalf("ParseFlags: %v", err)
	}
	return cmd, cf
}

func TestReconcileLocaleLegacyFile(t *testing.T) {
	cmd, cf := mkLocaleCmd(t)
	if err := reconcileLocale(cmd, "", cf, false); err != nil {
		t.Errorf("legacy export (no locale in file): err = %v, want nil", err)
	}
	if cf.locale != "en" {
		t.Errorf("legacy export: cf.locale = %q, want unchanged 'en'", cf.locale)
	}
}

func TestReconcileLocaleAdoptsFileLocaleWhenCLIImplicit(t *testing.T) {
	cmd, cf := mkLocaleCmd(t)
	if err := reconcileLocale(cmd, "it", cf, false); err != nil {
		t.Errorf("err = %v, want nil", err)
	}
	if cf.locale != "it" {
		t.Errorf("cf.locale = %q, want 'it' (adopted from file)", cf.locale)
	}
}

func TestReconcileLocaleKeepsCLIWhenMatching(t *testing.T) {
	cmd, cf := mkLocaleCmd(t, "--locale", "it")
	if err := reconcileLocale(cmd, "it", cf, false); err != nil {
		t.Errorf("err = %v, want nil", err)
	}
	if cf.locale != "it" {
		t.Errorf("cf.locale = %q, want 'it'", cf.locale)
	}
}

func TestReconcileLocaleErrorsOnExplicitMismatch(t *testing.T) {
	cmd, cf := mkLocaleCmd(t, "--locale", "us")
	err := reconcileLocale(cmd, "it", cf, false)
	var lm *configio.LocaleMismatchError
	if !errors.As(err, &lm) {
		t.Fatalf("err = %v, want LocaleMismatchError", err)
	}
	if lm.FromFile != "it" || lm.FromCLI != "us" {
		t.Errorf("error fields = %+v, want FromFile=it FromCLI=us", lm)
	}
}

func TestReconcileLocaleForceLocaleBypassesMismatch(t *testing.T) {
	cmd, cf := mkLocaleCmd(t, "--locale", "us")
	if err := reconcileLocale(cmd, "it", cf, true); err != nil {
		t.Errorf("--force-locale should bypass mismatch: err = %v", err)
	}
	if cf.locale != "us" {
		t.Errorf("--force-locale must keep CLI value: cf.locale = %q, want 'us'", cf.locale)
	}
}
