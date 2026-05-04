package session

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// RebootResult describes a reboot outcome.
type RebootResult struct {
	TriggerOK bool          // trigger returned 1 cleanly
	Outage    time.Duration // measured TCP outage if waited; 0 with --no-wait
}

// Reboot triggers RestartDevice via the G61850 parser. If wait is true,
// it polls for reconnection (up to 60 s) and reports the measured outage.
//
// SPEC § 2.8.2: trigger acknowledgement returns immediately; the actual
// reboot starts ~7 s later; outage is ~2 s on the Sample BESS.
func (s *Session) Reboot(ctx context.Context, wait bool) (*RebootResult, error) {
	if _, err := s.G61850(ctx, "RestartDevice", "", ""); err != nil {
		return nil, err
	}
	if !wait {
		return &RebootResult{TriggerOK: true}, nil
	}

	t0 := time.Now()
	deadline := t0.Add(60 * time.Second)
	var dropAt, recoverAt time.Time
	for time.Now().Before(deadline) {
		_ = s.t.Close()
		err := s.t.Open(ctx)
		if err != nil {
			if dropAt.IsZero() {
				dropAt = time.Now()
			}
			time.Sleep(500 * time.Millisecond)
			continue
		}
		// Probe with the discovery handshake.
		_, perr := s.t.ReadInputRegisters(ctx, addrIdentification, 5)
		if perr == nil {
			recoverAt = time.Now()
			break
		}
		if dropAt.IsZero() && errors.Is(perr, context.Canceled) {
			break
		}
		if dropAt.IsZero() {
			dropAt = time.Now()
		}
		time.Sleep(500 * time.Millisecond)
	}
	if recoverAt.IsZero() {
		return &RebootResult{TriggerOK: true}, fmt.Errorf("device did not return within 60 s")
	}
	if dropAt.IsZero() {
		dropAt = recoverAt
	}
	return &RebootResult{TriggerOK: true, Outage: recoverAt.Sub(dropAt)}, nil
}
