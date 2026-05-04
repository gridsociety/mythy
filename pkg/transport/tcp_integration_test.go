//go:build integration

package transport

import (
	"context"
	"fmt"
	"net"
	"reflect"
	"testing"
	"time"

	"github.com/simonvetter/modbus"
)

// fakeServer is a tiny in-process Modbus TCP server backed by a fixed
// register snapshot. We use it to exercise the real client serialization
// path without needing a device.
type fakeServer struct {
	input   map[uint16]uint16 // FC04 source
	holding map[uint16]uint16 // FC03 source
	written map[uint16]uint16 // FC06 / FC16 sink
}

func (s *fakeServer) HandleCoils(_ *modbus.CoilsRequest) ([]bool, error) {
	return nil, modbus.ErrIllegalFunction
}
func (s *fakeServer) HandleDiscreteInputs(_ *modbus.DiscreteInputsRequest) ([]bool, error) {
	return nil, modbus.ErrIllegalFunction
}
func (s *fakeServer) HandleHoldingRegisters(req *modbus.HoldingRegistersRequest) ([]uint16, error) {
	if req.IsWrite {
		for i, v := range req.Args {
			s.written[req.Addr+uint16(i)] = v
		}
		return req.Args, nil
	}
	out := make([]uint16, req.Quantity)
	for i := range out {
		out[i] = s.holding[req.Addr+uint16(i)]
	}
	return out, nil
}
func (s *fakeServer) HandleInputRegisters(req *modbus.InputRegistersRequest) ([]uint16, error) {
	out := make([]uint16, req.Quantity)
	for i := range out {
		out[i] = s.input[req.Addr+uint16(i)]
	}
	return out, nil
}

// reserveEphemeralPort grabs a free 127.0.0.1 port by opening and immediately
// closing a listener. simonvetter/modbus's server doesn't expose its listener
// address, so we pre-pick a port and pass it in via the URL. There's a small
// race window between Close and the server's Listen, but it's good enough for
// integration tests.
func reserveEphemeralPort(t *testing.T) (string, uint16) {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserveEphemeralPort: %v", err)
	}
	addr := l.Addr().(*net.TCPAddr)
	host := addr.IP.String()
	port := uint16(addr.Port)
	if err := l.Close(); err != nil {
		t.Fatalf("reserveEphemeralPort close: %v", err)
	}
	return host, port
}

func TestTCPClientReadWrite(t *testing.T) {
	srv := &fakeServer{
		input:   map[uint16]uint16{0x143E: 0x1234, 0x143F: 0x86A0, 0x1440: 0x0001, 0x1441: 0x0100, 0x1442: 0x0000},
		holding: map[uint16]uint16{0x1800: 0x0001},
		written: map[uint16]uint16{},
	}
	host, port := reserveEphemeralPort(t)
	mbsrv, err := modbus.NewServer(&modbus.ServerConfiguration{
		URL:        fmt.Sprintf("tcp://%s:%d", host, port),
		Timeout:    2 * time.Second,
		MaxClients: 4,
	}, srv)
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	if err := mbsrv.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer mbsrv.Stop()

	// Give the listener a moment to come up after Start (it spawns a goroutine).
	time.Sleep(50 * time.Millisecond)

	client := NewTCPClient(Options{Host: host, Port: port, UnitID: 1, RequestTimeout: 2 * time.Second})
	if err := client.Open(context.Background()); err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer client.Close()

	regs, err := client.ReadInputRegisters(context.Background(), 0x143E, 5)
	if err != nil {
		t.Fatalf("ReadInputRegisters: %v", err)
	}
	want := []uint16{0x1234, 0x86A0, 0x0001, 0x0100, 0x0000}
	if !reflect.DeepEqual(regs, want) {
		t.Errorf("got %v, want %v", regs, want)
	}

	if err := client.WriteSingleRegister(context.Background(), 0x1800, 5); err != nil {
		t.Fatalf("WriteSingleRegister: %v", err)
	}
	if got := srv.written[0x1800]; got != 5 {
		t.Errorf("server saw %d, want 5", got)
	}
}
