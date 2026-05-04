// Package transport implements Modbus TCP and Modbus RTU client transports
// for Thytronic devices. The Transport interface is what pkg/session/
// consumes; production code uses TCP / RTU, tests use the fake.
package transport

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// Transport is the abstract Modbus client interface used by pkg/session/.
// Implementations: tcp.Client, rtu.Client (production); fake.Transport (tests).
type Transport interface {
	// Open establishes the underlying connection.
	Open(ctx context.Context) error
	// Close tears down the connection. Must be safe to call multiple times.
	Close() error

	// ReadInputRegisters issues FC04. Returns qty 16-bit registers, big-endian.
	ReadInputRegisters(ctx context.Context, addr, qty uint16) ([]uint16, error)
	// ReadHoldingRegisters issues FC03.
	ReadHoldingRegisters(ctx context.Context, addr, qty uint16) ([]uint16, error)
	// WriteSingleRegister issues FC06.
	WriteSingleRegister(ctx context.Context, addr, value uint16) error
	// WriteMultipleRegisters issues FC16.
	WriteMultipleRegisters(ctx context.Context, addr uint16, values []uint16) error
}

// Options is the union of TCP and RTU connection settings; only the fields
// matching the chosen Transport implementation are read.
type Options struct {
	// Common
	UnitID         uint8         // default 1
	RequestTimeout time.Duration // default 2 s
	ConnectTimeout time.Duration // default 5 s

	// TCP only
	Host string
	Port uint16 // default 502 (IANA-registered Modbus TCP port)

	// RTU only
	SerialDevice string // e.g. "/dev/ttyUSB0"
	Baud         uint
	DataBits     uint
	Parity       string // "N" / "E" / "O"
	StopBits     uint
}

// ModbusException wraps an exception response from the device.
type ModbusException struct {
	FC      uint8 // the request FC (without the 0x80 error bit)
	Code    uint8 // exception code (0x01..0x06 typically)
	Message string
}

func (e *ModbusException) Error() string {
	return fmt.Sprintf("modbus exception: FC=0x%02X code=0x%02X (%s)", e.FC, e.Code, e.Message)
}

// IsTransient reports whether err is something a read should retry on.
// Used by pkg/session/ to implement the §3.0 retry policy.
//
// Modbus exceptions are NOT transient — they're device decisions. Any
// other non-nil error (timeout, EOF, connection-reset, …) is transient.
func IsTransient(err error) bool {
	if err == nil {
		return false
	}
	var me *ModbusException
	if errors.As(err, &me) {
		return false
	}
	return true
}

// RangePlan is one batched Modbus read request, produced by MergeRanges.
type RangePlan struct {
	FC        uint8 // 3 (holding) or 4 (input)
	StartAddr uint16
	Count     uint16
}

// End returns the exclusive end address (StartAddr + Count).
func (r RangePlan) End() int { return int(r.StartAddr) + int(r.Count) }

// Contains reports whether addr falls inside [StartAddr, End()).
func (r RangePlan) Contains(addr int) bool {
	return addr >= int(r.StartAddr) && addr < r.End()
}
