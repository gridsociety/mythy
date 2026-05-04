package transport

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

func TestFakeReadInputRegisters(t *testing.T) {
	f := NewFake()
	f.OnReadInput(0x143E, 5, []uint16{0x1234, 0x86A0, 0x0001, 0x0100, 0x0000})

	got, err := f.ReadInputRegisters(context.Background(), 0x143E, 5)
	if err != nil {
		t.Fatalf("ReadInputRegisters: %v", err)
	}
	want := []uint16{0x1234, 0x86A0, 0x0001, 0x0100, 0x0000}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestFakeReadInputUnscriptedPanics(t *testing.T) {
	f := NewFake()
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic on unscripted read")
		}
	}()
	_, _ = f.ReadInputRegisters(context.Background(), 100, 1)
}

func TestFakeException(t *testing.T) {
	f := NewFake()
	f.OnWriteSingleException(0x3C2F, &ModbusException{FC: 6, Code: 0x04, Message: "no session"})

	err := f.WriteSingleRegister(context.Background(), 0x3C2F, 2)
	var me *ModbusException
	if !errors.As(err, &me) {
		t.Fatalf("expected ModbusException, got %T %v", err, err)
	}
	if me.Code != 0x04 {
		t.Errorf("Code = 0x%02X", me.Code)
	}
}

func TestFakeRecordsWrites(t *testing.T) {
	f := NewFake()
	f.OnWriteSingleOK(0x3C2F)
	_ = f.WriteSingleRegister(context.Background(), 0x3C2F, 2)
	if got, want := f.Writes, []FakeWrite{{FC: 6, Addr: 0x3C2F, Values: []uint16{2}}}; !reflect.DeepEqual(got, want) {
		t.Errorf("Writes = %+v, want %+v", got, want)
	}
}
