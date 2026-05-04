package main

import (
	"strings"
	"testing"
)

func TestDescribe(t *testing.T) {
	out, err := runMythy(t,
		"--templates", testdataRoot(t),
		"--device", "TEST-VX0-a",
		"describe", "MB_address")
	if err != nil {
		t.Fatalf("describe: %v\n%s", err, out)
	}
	for _, want := range []string{
		"MB_address",
		"Modbus address",
		"FC=3", "addr=0x1801", "qty=1",
		"path: Set/Base",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q:\n%s", want, out)
		}
	}
}

func TestDescribeEnum(t *testing.T) {
	out, err := runMythy(t,
		"--templates", testdataRoot(t),
		"--device", "TEST-VX0-a",
		"describe", "MB_baudrate")
	if err != nil {
		t.Fatalf("describe: %v\n%s", err, out)
	}
	for _, want := range []string{"EnumBaudrate", "0=1200 baud", "4=19200 baud"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q:\n%s", want, out)
		}
	}
}

func TestDescribeUnknown(t *testing.T) {
	_, err := runMythy(t,
		"--templates", testdataRoot(t),
		"--device", "TEST-VX0-a",
		"describe", "NotARealName")
	if err == nil {
		t.Error("expected error for unknown name")
	}
}
