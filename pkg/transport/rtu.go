package transport

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/simonvetter/modbus"
)

// RTUClient implements Transport over Modbus RTU (RS-485 / USB serial).
type RTUClient struct {
	opts   Options
	mu     sync.Mutex
	client *modbus.ModbusClient
}

// NewRTUClient builds an RTU client with sensible defaults.
func NewRTUClient(opts Options) *RTUClient {
	if opts.UnitID == 0 {
		opts.UnitID = 1
	}
	if opts.Baud == 0 {
		opts.Baud = 19200
	}
	if opts.DataBits == 0 {
		opts.DataBits = 8
	}
	if opts.Parity == "" {
		opts.Parity = "N"
	}
	if opts.StopBits == 0 {
		opts.StopBits = 1
	}
	if opts.RequestTimeout == 0 {
		opts.RequestTimeout = 2 * time.Second
	}
	return &RTUClient{opts: opts}
}

// Open opens the serial port and the Modbus RTU framing.
func (c *RTUClient) Open(_ context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.client != nil {
		return nil
	}
	if c.opts.SerialDevice == "" {
		return fmt.Errorf("transport/rtu: SerialDevice is required")
	}
	parity := mapParity(c.opts.Parity)
	cfg := &modbus.ClientConfiguration{
		URL:      fmt.Sprintf("rtu://%s", c.opts.SerialDevice),
		Speed:    c.opts.Baud,
		DataBits: c.opts.DataBits,
		Parity:   parity,
		StopBits: c.opts.StopBits,
		Timeout:  c.opts.RequestTimeout,
	}
	cli, err := modbus.NewClient(cfg)
	if err != nil {
		return fmt.Errorf("modbus.NewClient (rtu): %w", err)
	}
	if err := cli.Open(); err != nil {
		return fmt.Errorf("modbus open %s: %w", c.opts.SerialDevice, err)
	}
	if err := cli.SetUnitId(c.opts.UnitID); err != nil {
		_ = cli.Close()
		return fmt.Errorf("SetUnitId(%d): %w", c.opts.UnitID, err)
	}
	c.client = cli
	return nil
}

func (c *RTUClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.client == nil {
		return nil
	}
	err := c.client.Close()
	c.client = nil
	return err
}

func (c *RTUClient) requireOpen() (*modbus.ModbusClient, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.client == nil {
		return nil, fmt.Errorf("transport: not open")
	}
	return c.client, nil
}

func (c *RTUClient) ReadInputRegisters(_ context.Context, addr, qty uint16) ([]uint16, error) {
	cli, err := c.requireOpen()
	if err != nil {
		return nil, err
	}
	regs, err := cli.ReadRegisters(addr, qty, modbus.INPUT_REGISTER)
	return regs, mapErr(err, 4)
}

func (c *RTUClient) ReadHoldingRegisters(_ context.Context, addr, qty uint16) ([]uint16, error) {
	cli, err := c.requireOpen()
	if err != nil {
		return nil, err
	}
	regs, err := cli.ReadRegisters(addr, qty, modbus.HOLDING_REGISTER)
	return regs, mapErr(err, 3)
}

func (c *RTUClient) WriteSingleRegister(_ context.Context, addr, value uint16) error {
	cli, err := c.requireOpen()
	if err != nil {
		return err
	}
	return mapErr(cli.WriteRegister(addr, value), 6)
}

func (c *RTUClient) WriteMultipleRegisters(_ context.Context, addr uint16, values []uint16) error {
	cli, err := c.requireOpen()
	if err != nil {
		return err
	}
	return mapErr(cli.WriteRegisters(addr, values), 16)
}

// mapParity translates the user-facing single-letter parity to the
// simonvetter/modbus integer constant.
func mapParity(p string) uint {
	switch p {
	case "N", "n":
		return modbus.PARITY_NONE
	case "E", "e":
		return modbus.PARITY_EVEN
	case "O", "o":
		return modbus.PARITY_ODD
	}
	return modbus.PARITY_NONE
}
