package grbl

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.bug.st/serial"
)

type Grbl struct {
	portName string
	port     serial.Port
}

func NewGrbl(portName string) *Grbl {
	g := &Grbl{
		portName: portName,
	}
	return g
}

func (g *Grbl) Open() error {
	mode := &serial.Mode{
		BaudRate: 115200,
	}
	port, err := serial.Open(g.portName, mode)
	if err != nil {
		return fmt.Errorf("grbl: serial port open error: %s: %w", g.portName, err)
	}

	// TODO there's a small race condition: after open Grbl resets, and the messages MAY be erased here.
	if err := port.ResetInputBuffer(); err != nil {
		closeErr := port.Close()
		if closeErr != nil {
			closeErr = fmt.Errorf("grbl: serial port close error: %s: %w", g.portName, closeErr)
		}
		return errors.Join(err, closeErr)
	}

	if err := port.ResetOutputBuffer(); err != nil {
		closeErr := port.Close()
		if closeErr != nil {
			closeErr = fmt.Errorf("grbl: serial port close error: %s: %w", g.portName, closeErr)
		}
		return errors.Join(err, closeErr)
	}

	g.port = port

	// TODO wait for reset message
	// │< "Grbl 1.1f ['$' for help]\r"                                                              │

	return nil
}

func (g *Grbl) Close() (err error) {
	if g.port != nil {
		err = g.port.Close()
		g.port = nil
		return
	}
	return
}

func (g *Grbl) Send(block string) error {
	line := append([]byte(block), '\n')
	n, err := g.port.Write(line)
	if err != nil {
		return fmt.Errorf("grbl: write to serial port error: %s: %w", g.portName, err)
	}
	if n != len(line) {
		return fmt.Errorf("grbl: write to serial port error: %s: wrote %d bytes, expected %d", g.portName, n, len(block))
	}
	if err := g.port.Drain(); err != nil {
		return fmt.Errorf("grbl: serial port drain error: %s: %w", g.portName, err)
	}

	return nil
}

func (g *Grbl) Receive(ctx context.Context) (Message, error) {
	deadline, ok := ctx.Deadline()
	var timeout time.Duration = serial.NoTimeout
	if ok {
		timeout = time.Until(deadline)
	}
	if err := g.port.SetReadTimeout(timeout); err != nil {
		return nil, fmt.Errorf("grbl: error setting serial port read timeout: %s: %w", g.portName, err)
	}

	line := []byte{}
	for {
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("grbl: context error: %s: %w", g.portName, err)
		}
		b := make([]byte, 1)
		n, err := g.port.Read(b)
		if err != nil {
			return nil, err
		}
		if n == 0 {
			return nil, fmt.Errorf("grbl: serial port read error: %s: 0 bytes read", g.portName)
		}
		if b[0] == '\n' {
			break
		}
		line = append(line, b[0])
	}

	if len(line) >= 1 && line[len(line)-1] == '\r' {
		line = line[:len(line)-1]
	}

	message := NewMessage(string(line))

	return message, nil
}
