// Package session is the high-level mythy API for live device operations.
package session

import (
	"errors"
	"fmt"

	"github.com/gridsociety/mythy/pkg/transport"
)

// ErrTransactionNotOpen means an FC06/FC16 write returned exception 0x04
// while no edit transaction was open. The fix is to call BeginEdit first.
type ErrTransactionNotOpen struct{ Underlying error }

func (e *ErrTransactionNotOpen) Error() string {
	return "edit transaction not open: writes to *_PARAM registers require START_CHANGE_DB; the device returned exception 0x04"
}
func (e *ErrTransactionNotOpen) Unwrap() error { return e.Underlying }

// ErrAccessLevelRequired means a write was rejected because secure mode
// is active and the user has not authenticated. mythy v1 doesn't yet
// drive the auth flow; the operator should escalate via ThyVisor first.
type ErrAccessLevelRequired struct{ Underlying error }

func (e *ErrAccessLevelRequired) Error() string {
	return "device rejected the write — secure mode is ON and mythy v1 does not yet implement the auth flow; escalate access level via ThyVisor first"
}
func (e *ErrAccessLevelRequired) Unwrap() error { return e.Underlying }

// ErrDeviceFailure is the catch-all for Modbus exception codes that
// don't map to a more specific error.
type ErrDeviceFailure struct{ Underlying *transport.ModbusException }

func (e *ErrDeviceFailure) Error() string {
	return fmt.Sprintf("device failure: %s", e.Underlying.Error())
}
func (e *ErrDeviceFailure) Unwrap() error { return e.Underlying }

// MapException translates a transport-layer Modbus exception into the
// most specific typed error mythy can produce, given runtime context
// (whether an edit transaction is open, whether secure mode is on).
//
// Non-exception errors pass through untouched.
func MapException(err error, txnOpen, secureModeOn bool) error {
	if err == nil {
		return nil
	}
	var exc *transport.ModbusException
	if !errors.As(err, &exc) {
		return err
	}
	if exc.Code == 0x04 {
		// Two distinct meanings for the same wire code:
		// - txn-not-open if no transaction was open
		// - auth-required if secure mode is on and txn was open
		if !txnOpen {
			return &ErrTransactionNotOpen{Underlying: exc}
		}
		if secureModeOn {
			return &ErrAccessLevelRequired{Underlying: exc}
		}
	}
	return &ErrDeviceFailure{Underlying: exc}
}
