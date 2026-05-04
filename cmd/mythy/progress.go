package main

import (
	"fmt"
	"io"
	"os"
	"time"

	"golang.org/x/term"
)

// makeReadProgress returns a TTY-aware progress reporter for the
// mythy commands that loop one Modbus read per catalog leaf
// (export, import, diff). The reporter overwrites a single line on
// `w` using \r, throttled to ~10 Hz so terminal redraws don't
// dominate the export/diff cost. The first call always renders so
// the user gets immediate feedback; the final sentinel call (with
// name == "" and done >= total) clears the line.
//
// When stderr isn't a TTY the returned function is nil so callers
// don't even invoke it.
func makeReadProgress(w io.Writer, force bool) func(done, total int, name string) {
	if !force && !term.IsTerminal(int(os.Stderr.Fd())) {
		return nil
	}
	const (
		throttle = 100 * time.Millisecond
		nameW    = 40
		lineW    = 60
	)
	var lastUpdate time.Time
	return func(done, total int, name string) {
		if name == "" && done >= total {
			fmt.Fprintf(w, "\r%-*s\r", lineW, "")
			return
		}
		if !lastUpdate.IsZero() && time.Since(lastUpdate) < throttle && done > 0 {
			return
		}
		lastUpdate = time.Now()
		shown := name
		if len(shown) > nameW {
			shown = shown[:nameW-3] + "..."
		}
		fmt.Fprintf(w, "\r[%5d/%5d] %-*s", done+1, total, nameW, shown)
	}
}
