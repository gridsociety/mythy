package session

import (
	"context"
	"fmt"
	"time"
)

// RebootResult describes a reboot outcome.
type RebootResult struct {
	TriggerOK bool          // trigger returned 1 cleanly
	Outage    time.Duration // measured TCP outage if waited; 0 with --no-wait
}

// rebootDropPollInterval is how often Phase 1 polls the existing
// connection waiting for the device to actually drop. SPEC § 2.8.2
// says the trigger acknowledges immediately and the device tears
// down the TCP socket ~7 s later; 200 ms gives us the drop instant
// to within a quarter of the SPEC-documented ~2 s outage.
const rebootDropPollInterval = 200 * time.Millisecond

// rebootRecoverPollInterval is how often Phase 2 retries reconnect
// while the device is rebooting. ~500 ms balances responsiveness
// against not hammering the boot loader.
const rebootRecoverPollInterval = 500 * time.Millisecond

// Reboot triggers RestartDevice via the G61850 parser. If wait is true,
// it polls for the actual TCP drop and the subsequent reconnect (up to
// 60 s total) and reports the measured outage.
//
// SPEC § 2.8.2: trigger acknowledgement returns immediately; the
// actual reboot starts ~7 s later; outage is ~2 s on the Sample BESS.
//
// The polling has two phases:
//
//  1. **Wait for drop.** Re-use the existing connection — read the
//     IDENTIFICATION register every 200 ms until a read fails. The
//     first failure marks dropAt. Closing-and-reopening the connection
//     in this phase would mask the drop because the socket is still
//     usable for several seconds after the trigger acknowledges.
//  2. **Wait for recovery.** Now that the device is gone, close the
//     stale socket and try to reopen. Once Open + IDENTIFICATION read
//     both succeed, the device is back. Mark recoverAt.
//
// Outage = recoverAt - dropAt.
func (s *Session) Reboot(ctx context.Context, wait bool) (*RebootResult, error) {
	if _, err := s.G61850(ctx, "RestartDevice", "", ""); err != nil {
		return nil, err
	}
	if !wait {
		return &RebootResult{TriggerOK: true}, nil
	}

	deadline := time.Now().Add(60 * time.Second)

	dropAt, err := s.waitForRebootDrop(ctx, deadline)
	if err != nil {
		return &RebootResult{TriggerOK: true}, err
	}
	recoverAt, err := s.waitForRebootRecovery(ctx, deadline)
	if err != nil {
		return &RebootResult{TriggerOK: true}, err
	}
	return &RebootResult{TriggerOK: true, Outage: recoverAt.Sub(dropAt)}, nil
}

// waitForRebootDrop polls the open connection at rebootDropPollInterval
// until a read fails. Returns the time of the first failure.
func (s *Session) waitForRebootDrop(ctx context.Context, deadline time.Time) (time.Time, error) {
	for time.Now().Before(deadline) {
		if _, err := s.t.ReadInputRegisters(ctx, addrIdentification, 5); err != nil {
			return time.Now(), nil
		}
		select {
		case <-ctx.Done():
			return time.Time{}, ctx.Err()
		case <-time.After(rebootDropPollInterval):
		}
	}
	return time.Time{}, fmt.Errorf("device did not drop within %s", time.Until(deadline))
}

// waitForRebootRecovery closes the stale socket, then attempts
// Open+probe at rebootRecoverPollInterval until both succeed.
// Returns the time of the first successful probe.
func (s *Session) waitForRebootRecovery(ctx context.Context, deadline time.Time) (time.Time, error) {
	for time.Now().Before(deadline) {
		_ = s.t.Close()
		if err := s.t.Open(ctx); err == nil {
			if _, perr := s.t.ReadInputRegisters(ctx, addrIdentification, 5); perr == nil {
				return time.Now(), nil
			}
		}
		select {
		case <-ctx.Done():
			return time.Time{}, ctx.Err()
		case <-time.After(rebootRecoverPollInterval):
		}
	}
	return time.Time{}, fmt.Errorf("device did not return within timeout")
}
