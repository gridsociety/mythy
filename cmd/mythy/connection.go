package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/gridsociety/mythy/pkg/catalog"
	"github.com/gridsociety/mythy/pkg/session"
	"github.com/gridsociety/mythy/pkg/transport"
	"github.com/spf13/cobra"
)

// connFlags collects the connection flags every live command takes.
type connFlags struct {
	transportKind string // "tcp" / "rtu" / "" (auto from --host vs --serial)
	host          string
	port          uint16
	serialDev     string
	baud          uint
	parity        string
	stopBits      uint
	unitID        uint8
	timeout       time.Duration
	connTimeout   time.Duration
	retries       int
}

// bind registers the connection flags. Each flag's default is sourced
// from MYTHY_<NAME> if set (e.g. MYTHY_HOST, MYTHY_REQUEST_TIMEOUT),
// matching the precedent of MYTHY_TEMPLATES / MYTHY_LOCALE / MYTHY_FORMAT.
// CLI flags still win over env vars.
func (c *connFlags) bind(cmd *cobra.Command) {
	f := cmd.PersistentFlags()
	f.StringVar(&c.transportKind, "transport", envOrString("MYTHY_TRANSPORT", ""),
		"tcp or rtu (auto from --host or --serial if unset; or set MYTHY_TRANSPORT)")
	f.StringVar(&c.host, "host", envOrString("MYTHY_HOST", ""),
		"TCP host, e.g. 192.0.2.10 (or set MYTHY_HOST)")
	f.Uint16Var(&c.port, "port", envOrUint16("MYTHY_PORT", 502),
		"TCP port (default 502, IANA-registered Modbus TCP; or set MYTHY_PORT)")
	f.StringVar(&c.serialDev, "serial", envOrString("MYTHY_SERIAL", ""),
		"serial device path, e.g. /dev/ttyUSB0 (or set MYTHY_SERIAL)")
	f.UintVar(&c.baud, "baud", envOrUint("MYTHY_BAUD", 19200),
		"RTU baud rate (or set MYTHY_BAUD)")
	f.StringVar(&c.parity, "parity", envOrString("MYTHY_PARITY", "N"),
		"RTU parity: N, E, O (or set MYTHY_PARITY)")
	f.UintVar(&c.stopBits, "stopbits", envOrUint("MYTHY_STOPBITS", 1),
		"RTU stop bits: 1 or 2 (or set MYTHY_STOPBITS)")
	f.Uint8Var(&c.unitID, "unit-id", envOrUint8("MYTHY_UNIT_ID", 1),
		"Modbus unit ID (default 1; or set MYTHY_UNIT_ID)")
	f.DurationVar(&c.timeout, "request-timeout", envOrDuration("MYTHY_REQUEST_TIMEOUT", 10*time.Second),
		"per-request timeout (or set MYTHY_REQUEST_TIMEOUT)")
	f.DurationVar(&c.connTimeout, "connect-timeout", envOrDuration("MYTHY_CONNECT_TIMEOUT", 5*time.Second),
		"TCP connect timeout (or set MYTHY_CONNECT_TIMEOUT)")
	f.IntVar(&c.retries, "retries", envOrInt("MYTHY_RETRIES", 2),
		"transient-error retries on reads, never on writes (or set MYTHY_RETRIES)")
}

// build composes a ready-to-use Session: opens the transport, runs the
// discovery handshake, loads the matching catalog template.
func (c *connFlags) build(ctx context.Context, cf *catalogFlags) (*session.Session, error) {
	if c.host == "" && c.serialDev == "" {
		return nil, fmt.Errorf("one of --host or --serial is required for live commands")
	}
	if cf.templatesRoot == "" {
		return nil, fmt.Errorf("--templates is required (or set MYTHY_TEMPLATES)")
	}

	var t transport.Transport
	opts := transport.Options{
		UnitID:         c.unitID,
		RequestTimeout: c.timeout,
		ConnectTimeout: c.connTimeout,
		Host:           c.host,
		Port:           c.port,
		SerialDevice:   c.serialDev,
		Baud:           c.baud,
		Parity:         c.parity,
		StopBits:       c.stopBits,
		DataBits:       8,
	}
	switch {
	case c.host != "":
		t = transport.NewTCPClient(opts)
	case c.serialDev != "":
		t = transport.NewRTUClient(opts)
	}
	if err := t.Open(ctx); err != nil {
		_ = t.Close()
		return nil, err
	}

	codifica, err := catalog.LoadCodifica(cf.templatesRoot + "/Codifica.xml")
	if err != nil {
		_ = t.Close()
		return nil, err
	}

	// Audit I2 / SPEC § 5.4: when --device is given, skip the discovery
	// read and use the explicit product directly. Audit I1: when --device
	// is NOT given, do exactly one discovery read; the Session's later
	// Identify() call will reuse it via SeedIdentification().
	var (
		entry catalog.DeviceEntry
		ident *session.Identification
	)
	if cf.device != "" {
		for _, d := range codifica.Devices {
			if d.Product == cf.device {
				entry = d
				break
			}
		}
		if entry.Product == "" {
			_ = t.Close()
			return nil, fmt.Errorf("--device=%q not found in Codifica.xml", cf.device)
		}
	} else {
		// Look up IDENTIFICATION's wire address from the catalog instead
		// of hardcoding 0x143E (audit I11). We need a template for that
		// lookup, which we don't have yet — but every Thytronic relay
		// template that declares IDENTIFICATION puts it at num=5183
		// (empirically verified across the full vendor catalog), so
		// hardcoding 0x143E here is structurally fine; this comment
		// records the deliberate choice.
		regs, err := t.ReadInputRegisters(ctx, 0x143E, 5)
		if err != nil {
			_ = t.Close()
			return nil, fmt.Errorf("discovery: %w", err)
		}
		entry, err = codifica.ByIdentification(int(regs[0]))
		if err != nil {
			_ = t.Close()
			return nil, err
		}
		ident = &session.Identification{
			Identification:  regs[0],
			SerialNumber:    (uint32(regs[2]) << 16) | uint32(regs[1]),
			FwRelease:       regs[3],
			ProtocolRelease: regs[4],
		}
	}

	tpl, _, err := catalog.Load(catalog.LoadOptions{
		Root:    cf.templatesRoot,
		Locale:  cf.locale,
		Product: entry.Product,
	})
	if err != nil {
		_ = t.Close()
		return nil, err
	}

	s, err := session.NewWithTransport(t, tpl, entry)
	if err != nil {
		_ = t.Close()
		return nil, err
	}
	// Audit C8 / SPEC § 3.0: thread the retry policy from CLI flags
	// into the session. The session's data-read path retries transient
	// errors; writes and trigger reads never retry.
	s.SetRetryPolicy(c.retries, 200*time.Millisecond)

	// Audit I1: if we already have an identification from discovery,
	// seed it instead of re-reading. SecMode probe still happens.
	if ident != nil {
		s.SeedIdentification(ident)
	}
	if _, err := s.Identify(ctx); err != nil {
		_ = s.Close()
		return nil, fmt.Errorf("identify: %w", err)
	}
	if cf.device != "" {
		live := s.Ident()
		if live != nil && entry.Identification != 0 && live.Identification != uint16(entry.Identification) {
			fmt.Fprintf(os.Stderr, "warning: --device=%q (id=%d) doesn't match live identification %d\n",
				cf.device, entry.Identification, live.Identification)
		}
	}
	return s, nil
}
