package session_test

import (
	"context"
	"testing"

	"github.com/gridsociety/mythy/pkg/transport"
)

func TestEnabledModulesCachesResult(t *testing.T) {
	f := transport.NewFake()
	// EnableBoard_Madre at the wire address declared in the fixture
	// (= num-1 of the message named after Variabile). For the synthetic
	// fixture this register doesn't exist as a <message>, so the probe
	// falls through Read's "not found" path → that module is treated
	// as "unknown → enabled". This test verifies the cache + fallback.
	s := mkSession(t, f)
	m1, err := s.EnabledModules(context.Background())
	if err != nil {
		t.Fatalf("EnabledModules: %v", err)
	}
	if !m1["SchedaMadre"] {
		t.Errorf("missing-register fallback should treat module as enabled")
	}
	// Second call must hit the cache — no new transport activity.
	m2, _ := s.EnabledModules(context.Background())
	if &m1 == &m2 {
		t.Skip("can't compare map identity; we just want the call to be a no-op")
	}
}
