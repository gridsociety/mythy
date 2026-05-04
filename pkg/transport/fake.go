package transport

import (
	"context"
	"fmt"
)

// Fake is a scripted in-memory Transport for tests. Calls that are not
// pre-scripted panic, so a test cannot silently rely on a default value.
//
// Usage:
//
//	f := NewFake()
//	f.OnReadInput(0x143E, 5, []uint16{...})
//	f.OnWriteSingleOK(0x3C2F)
//	...
//	for _, w := range f.Writes { /* assert */ }
type Fake struct {
	readInput   map[fakeKey][]uint16
	readHolding map[fakeKey][]uint16
	writeSingle map[uint16]error // addr → error (nil = OK)
	writeMulti  map[uint16]error // addr → error (nil = OK)
	excs        map[fakeKey]*ModbusException

	Writes []FakeWrite // recorded in call order
}

type fakeKey struct {
	addr, qty uint16
}

// FakeWrite is a recorded write; one per FC06 (Values has 1 element) or
// FC16 (Values has Count elements) call.
type FakeWrite struct {
	FC     uint8
	Addr   uint16
	Values []uint16
}

// NewFake returns an empty fake transport.
func NewFake() *Fake {
	return &Fake{
		readInput:   make(map[fakeKey][]uint16),
		readHolding: make(map[fakeKey][]uint16),
		writeSingle: make(map[uint16]error),
		writeMulti:  make(map[uint16]error),
		excs:        make(map[fakeKey]*ModbusException),
	}
}

// OnReadInput pre-scripts FC04(addr, qty) to return the given regs.
func (f *Fake) OnReadInput(addr, qty uint16, regs []uint16) {
	f.readInput[fakeKey{addr, qty}] = regs
}

// OnReadHolding pre-scripts FC03(addr, qty).
func (f *Fake) OnReadHolding(addr, qty uint16, regs []uint16) {
	f.readHolding[fakeKey{addr, qty}] = regs
}

// OnReadInputException pre-scripts FC04 to return an exception.
func (f *Fake) OnReadInputException(addr, qty uint16, exc *ModbusException) {
	f.excs[fakeKey{addr, qty}] = exc
}

// OnWriteSingleOK / OnWriteSingleException pre-script FC06.
func (f *Fake) OnWriteSingleOK(addr uint16)                              { f.writeSingle[addr] = nil }
func (f *Fake) OnWriteSingleException(addr uint16, exc *ModbusException) { f.writeSingle[addr] = exc }

// OnWriteMultiOK / OnWriteMultiException pre-script FC16.
func (f *Fake) OnWriteMultiOK(addr uint16)                              { f.writeMulti[addr] = nil }
func (f *Fake) OnWriteMultiException(addr uint16, exc *ModbusException) { f.writeMulti[addr] = exc }

func (f *Fake) Open(_ context.Context) error { return nil }
func (f *Fake) Close() error                 { return nil }

func (f *Fake) ReadInputRegisters(_ context.Context, addr, qty uint16) ([]uint16, error) {
	k := fakeKey{addr, qty}
	if exc, ok := f.excs[k]; ok && exc != nil {
		return nil, exc
	}
	regs, ok := f.readInput[k]
	if !ok {
		panic(fmt.Sprintf("fake: unscripted FC04 read addr=0x%04X qty=%d", addr, qty))
	}
	out := make([]uint16, len(regs))
	copy(out, regs)
	return out, nil
}

func (f *Fake) ReadHoldingRegisters(_ context.Context, addr, qty uint16) ([]uint16, error) {
	regs, ok := f.readHolding[fakeKey{addr, qty}]
	if !ok {
		panic(fmt.Sprintf("fake: unscripted FC03 read addr=0x%04X qty=%d", addr, qty))
	}
	out := make([]uint16, len(regs))
	copy(out, regs)
	return out, nil
}

func (f *Fake) WriteSingleRegister(_ context.Context, addr, value uint16) error {
	err, ok := f.writeSingle[addr]
	if !ok {
		panic(fmt.Sprintf("fake: unscripted FC06 addr=0x%04X", addr))
	}
	f.Writes = append(f.Writes, FakeWrite{FC: 6, Addr: addr, Values: []uint16{value}})
	return err
}

func (f *Fake) WriteMultipleRegisters(_ context.Context, addr uint16, values []uint16) error {
	err, ok := f.writeMulti[addr]
	if !ok {
		panic(fmt.Sprintf("fake: unscripted FC16 addr=0x%04X", addr))
	}
	cp := make([]uint16, len(values))
	copy(cp, values)
	f.Writes = append(f.Writes, FakeWrite{FC: 16, Addr: addr, Values: cp})
	return err
}
