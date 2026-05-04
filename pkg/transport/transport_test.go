package transport

import (
	"errors"
	"testing"
)

func TestExceptionError(t *testing.T) {
	err := &ModbusException{Code: 0x04, Message: "slave device failure"}
	if err.Error() == "" {
		t.Error("Error() must produce a string")
	}

	var me *ModbusException
	if !errors.As(err, &me) {
		t.Error("errors.As must extract ModbusException")
	}
	if me.Code != 0x04 {
		t.Errorf("Code = 0x%02X", me.Code)
	}
}

func TestRangePlanRequest(t *testing.T) {
	plan := RangePlan{FC: 4, StartAddr: 100, Count: 5}
	if got, want := plan.End(), 105; got != want {
		t.Errorf("End() = %d, want %d", got, want)
	}
	if !plan.Contains(102) {
		t.Error("RangePlan should contain its midpoint")
	}
	if plan.Contains(105) {
		t.Error("RangePlan.End() is exclusive")
	}
}
