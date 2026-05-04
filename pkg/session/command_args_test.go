package session_test

import (
	"context"
	"testing"

	"github.com/gridsociety/mythy/pkg/transport"
)

func TestCommandWithStructuredArgs(t *testing.T) {
	// SET_RTC is num=5169 (wire 5168), dim=6, with parts day/month/year/hour/minute/second.
	f := transport.NewFake()
	f.OnWriteMultiOK(5168)
	s := mkSession(t, f)

	args := map[string]any{
		"RTCDay": uint64(30), "RTCMonth": uint64(4), "RTCYear": uint64(26),
		"RTCHour": uint64(12), "RTCMinute": uint64(0), "RTCSecond": uint64(0),
	}
	if err := s.Command(context.Background(), "SET_RTC", args); err != nil {
		t.Fatalf("Command: %v", err)
	}
	if len(f.Writes) != 1 || f.Writes[0].Addr != 5168 || len(f.Writes[0].Values) != 6 {
		t.Fatalf("expected one FC16 write of 6 regs to 5168; got %+v", f.Writes)
	}
	if got := f.Writes[0].Values[0]; got != 30 {
		t.Errorf("RTCDay regs[0] = %d, want 30", got)
	}
}
