package main

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"sync"
	"time"
)

// envWarn writes warnings about malformed env vars. Patched in tests.
var envWarn io.Writer = os.Stderr

// envWarned dedupes warnings — connFlags.bind runs once per subcommand
// so without this a single malformed env var fires N warnings during
// command-tree setup. Reset between tests via resetEnvWarned.
var (
	envWarnedMu sync.Mutex
	envWarned   = map[string]bool{}
)

func resetEnvWarned() {
	envWarnedMu.Lock()
	defer envWarnedMu.Unlock()
	envWarned = map[string]bool{}
}

func warnOnce(name, format string, args ...any) {
	envWarnedMu.Lock()
	defer envWarnedMu.Unlock()
	if envWarned[name] {
		return
	}
	envWarned[name] = true
	fmt.Fprintf(envWarn, format, args...)
}

func envOrString(name, def string) string {
	if v, ok := os.LookupEnv(name); ok {
		return v
	}
	return def
}

func envOrUint16(name string, def uint16) uint16 {
	v, ok := os.LookupEnv(name)
	if !ok || v == "" {
		return def
	}
	n, err := strconv.ParseUint(v, 10, 16)
	if err != nil {
		warnOnce(name, "warning: %s=%q is not a valid uint16; using %d\n", name, v, def)
		return def
	}
	return uint16(n)
}

func envOrUint8(name string, def uint8) uint8 {
	v, ok := os.LookupEnv(name)
	if !ok || v == "" {
		return def
	}
	n, err := strconv.ParseUint(v, 10, 8)
	if err != nil {
		warnOnce(name, "warning: %s=%q is not a valid uint8; using %d\n", name, v, def)
		return def
	}
	return uint8(n)
}

func envOrUint(name string, def uint) uint {
	v, ok := os.LookupEnv(name)
	if !ok || v == "" {
		return def
	}
	n, err := strconv.ParseUint(v, 10, 64)
	if err != nil {
		warnOnce(name, "warning: %s=%q is not a valid uint; using %d\n", name, v, def)
		return def
	}
	return uint(n)
}

func envOrInt(name string, def int) int {
	v, ok := os.LookupEnv(name)
	if !ok || v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		warnOnce(name, "warning: %s=%q is not a valid int; using %d\n", name, v, def)
		return def
	}
	return n
}

func envOrDuration(name string, def time.Duration) time.Duration {
	v, ok := os.LookupEnv(name)
	if !ok || v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		warnOnce(name, "warning: %s=%q is not a valid duration; using %s\n", name, v, def)
		return def
	}
	return d
}
