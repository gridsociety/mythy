package main

import (
	"bytes"
	"testing"
	"time"

	"github.com/spf13/cobra"
)

func TestEnvHelpers(t *testing.T) {
	t.Run("unset returns default", func(t *testing.T) {
		t.Setenv("MYTHY_TEST_UNSET", "")
		if got := envOrUint16("MYTHY_TEST_UNSET", 502); got != 502 {
			t.Errorf("unset uint16: got %d, want 502", got)
		}
		if got := envOrDuration("MYTHY_TEST_UNSET", 2*time.Second); got != 2*time.Second {
			t.Errorf("unset duration: got %v, want 2s", got)
		}
	})

	t.Run("set parses", func(t *testing.T) {
		t.Setenv("MYTHY_TEST_PORT", "1502")
		if got := envOrUint16("MYTHY_TEST_PORT", 502); got != 1502 {
			t.Errorf("uint16: got %d, want 1502", got)
		}
		t.Setenv("MYTHY_TEST_TIMEOUT", "5s")
		if got := envOrDuration("MYTHY_TEST_TIMEOUT", 2*time.Second); got != 5*time.Second {
			t.Errorf("duration: got %v, want 5s", got)
		}
	})

	t.Run("malformed warns once across repeated calls", func(t *testing.T) {
		var warns bytes.Buffer
		envWarn = &warns
		resetEnvWarned()
		t.Cleanup(func() { envWarn = nil; resetEnvWarned() })
		t.Setenv("MYTHY_TEST_BAD", "abc")
		// connFlags.bind runs once per subcommand — repeated lookups must
		// not produce duplicate warnings.
		for range 5 {
			if got := envOrUint16("MYTHY_TEST_BAD", 502); got != 502 {
				t.Errorf("malformed uint16: got %d, want 502 fallback", got)
			}
		}
		if !bytes.Contains(warns.Bytes(), []byte("MYTHY_TEST_BAD")) {
			t.Errorf("expected warning about MYTHY_TEST_BAD; got %q", warns.String())
		}
		if n := bytes.Count(warns.Bytes(), []byte("warning:")); n != 1 {
			t.Errorf("expected exactly 1 warning across 5 calls, got %d:\n%s", n, warns.String())
		}
	})
}

// TestConnFlagsHonorEnv verifies that connection-flag defaults pick up
// MYTHY_<NAME> env vars at bind time, and that explicit CLI flags still win.
func TestConnFlagsHonorEnv(t *testing.T) {
	t.Run("env supplies defaults", func(t *testing.T) {
		t.Setenv("MYTHY_HOST", "10.0.0.42")
		t.Setenv("MYTHY_PORT", "1502")
		t.Setenv("MYTHY_REQUEST_TIMEOUT", "7s")
		t.Setenv("MYTHY_RETRIES", "0")

		var c connFlags
		cmd := &cobra.Command{Use: "x"}
		c.bind(cmd)
		// Cobra applies default-from-bind without parsing args.
		if err := cmd.ParseFlags(nil); err != nil {
			t.Fatalf("ParseFlags: %v", err)
		}
		if c.host != "10.0.0.42" {
			t.Errorf("host: got %q, want 10.0.0.42", c.host)
		}
		if c.port != 1502 {
			t.Errorf("port: got %d, want 1502", c.port)
		}
		if c.timeout != 7*time.Second {
			t.Errorf("request-timeout: got %v, want 7s", c.timeout)
		}
		if c.retries != 0 {
			t.Errorf("retries: got %d, want 0", c.retries)
		}
	})

	t.Run("CLI flag overrides env", func(t *testing.T) {
		t.Setenv("MYTHY_PORT", "1502")

		var c connFlags
		cmd := &cobra.Command{Use: "x"}
		c.bind(cmd)
		if err := cmd.ParseFlags([]string{"--port", "9000"}); err != nil {
			t.Fatalf("ParseFlags: %v", err)
		}
		if c.port != 9000 {
			t.Errorf("port: got %d, want 9000 (CLI wins over env)", c.port)
		}
	})
}
