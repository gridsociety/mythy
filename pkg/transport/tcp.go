package transport

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/simonvetter/modbus"
)

// TCPClient implements Transport over Modbus TCP.
type TCPClient struct {
	opts   Options
	mu     sync.Mutex
	client *modbus.ModbusClient
}

// NewTCPClient builds a TCP client with sensible defaults filled in.
func NewTCPClient(opts Options) *TCPClient {
	if opts.UnitID == 0 {
		opts.UnitID = 1
	}
	if opts.Port == 0 {
		opts.Port = 502
	}
	if opts.RequestTimeout == 0 {
		opts.RequestTimeout = 10 * time.Second
	}
	if opts.ConnectTimeout == 0 {
		opts.ConnectTimeout = 5 * time.Second
	}
	return &TCPClient{opts: opts}
}

// Open establishes the TCP connection.
func (c *TCPClient) Open(_ context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.client != nil {
		return nil
	}
	cfg := &modbus.ClientConfiguration{
		URL:     fmt.Sprintf("tcp://%s:%d", c.opts.Host, c.opts.Port),
		Timeout: c.opts.RequestTimeout,
	}
	cli, err := modbus.NewClient(cfg)
	if err != nil {
		return fmt.Errorf("modbus.NewClient: %w", err)
	}
	if err := cli.Open(); err != nil {
		return fmt.Errorf("modbus open %s:%d: %w", c.opts.Host, c.opts.Port, err)
	}
	if err := cli.SetUnitId(c.opts.UnitID); err != nil {
		_ = cli.Close()
		return fmt.Errorf("SetUnitId(%d): %w", c.opts.UnitID, err)
	}
	c.client = cli
	return nil
}

// Close tears down the connection. Safe to call repeatedly.
func (c *TCPClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.client == nil {
		return nil
	}
	err := c.client.Close()
	c.client = nil
	return err
}

func (c *TCPClient) requireOpen() (*modbus.ModbusClient, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.client == nil {
		return nil, fmt.Errorf("transport: not open")
	}
	return c.client, nil
}

// ReadInputRegisters issues FC04.
func (c *TCPClient) ReadInputRegisters(_ context.Context, addr, qty uint16) ([]uint16, error) {
	cli, err := c.requireOpen()
	if err != nil {
		return nil, err
	}
	regs, err := cli.ReadRegisters(addr, qty, modbus.INPUT_REGISTER)
	return regs, mapErr(err, 4)
}

// ReadHoldingRegisters issues FC03.
func (c *TCPClient) ReadHoldingRegisters(_ context.Context, addr, qty uint16) ([]uint16, error) {
	cli, err := c.requireOpen()
	if err != nil {
		return nil, err
	}
	regs, err := cli.ReadRegisters(addr, qty, modbus.HOLDING_REGISTER)
	return regs, mapErr(err, 3)
}

// WriteSingleRegister issues FC06.
func (c *TCPClient) WriteSingleRegister(_ context.Context, addr, value uint16) error {
	cli, err := c.requireOpen()
	if err != nil {
		return err
	}
	return mapErr(cli.WriteRegister(addr, value), 6)
}

// WriteMultipleRegisters issues FC16.
func (c *TCPClient) WriteMultipleRegisters(_ context.Context, addr uint16, values []uint16) error {
	cli, err := c.requireOpen()
	if err != nil {
		return err
	}
	return mapErr(cli.WriteRegisters(addr, values), 16)
}

// mapErr translates modbus library errors into our typed exception form.
func mapErr(err error, fc uint8) error {
	if err == nil {
		return nil
	}
	// simonvetter/modbus returns sentinel errors for exception codes.
	switch err {
	case modbus.ErrIllegalFunction:
		return &ModbusException{FC: fc, Code: 0x01, Message: "illegal function"}
	case modbus.ErrIllegalDataAddress:
		return &ModbusException{FC: fc, Code: 0x02, Message: "illegal data address"}
	case modbus.ErrIllegalDataValue:
		return &ModbusException{FC: fc, Code: 0x03, Message: "illegal data value"}
	case modbus.ErrServerDeviceFailure:
		return &ModbusException{FC: fc, Code: 0x04, Message: "slave device failure"}
	case modbus.ErrAcknowledge:
		return &ModbusException{FC: fc, Code: 0x05, Message: "acknowledge"}
	case modbus.ErrServerDeviceBusy:
		return &ModbusException{FC: fc, Code: 0x06, Message: "device busy"}
	}
	return err
}
