package transport

import (
	"context"
	"strings"
	"testing"
)

func TestNewRTUClientDefaults(t *testing.T) {
	c := NewRTUClient(Options{SerialDevice: "/dev/null"})
	if c.opts.Baud == 0 {
		t.Errorf("default Baud must be set, got 0")
	}
	if c.opts.UnitID == 0 {
		t.Errorf("default UnitID must be 1, got 0")
	}
	if c.opts.Parity == "" {
		t.Errorf("default Parity must be set")
	}
}

func TestNewRTUClientNoDevice(t *testing.T) {
	c := NewRTUClient(Options{})
	if err := c.Open(context.TODO()); err == nil || !strings.Contains(err.Error(), "SerialDevice") {
		t.Errorf("Open with no SerialDevice should error, got %v", err)
	}
}
