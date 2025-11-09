package grbl

import (
	"errors"
	"fmt"

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
		return fmt.Errorf("%s: %w", g.portName, err)
	}

	// TODO there's a small race condition: after open Grbl resets, and the messages MAY be erased here.
	if err := port.ResetInputBuffer(); err != nil {
		return errors.Join(err, port.Close())
	}

	if err := port.ResetOutputBuffer(); err != nil {
		return errors.Join(err, port.Close())
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
		return err
	}
	if n != len(line) {
		return fmt.Errorf("sent %d bytes, expected %d", n, len(block))
	}
	if err := g.port.Drain(); err != nil {
		return err
	}

	return nil
}

func (g *Grbl) Receive() (Message, error) {
	line := []byte{}
	for {
		b := make([]byte, 1)
		n, err := g.port.Read(b)
		if err != nil {
			return nil, err
		}
		if n == 0 {
			return nil, errors.New("0 bytes read")
		}
		if b[0] == '\n' {
			break
		}
		line = append(line, b[0])
	}

	if len(line) >= 1 && line[len(line)-1] == '\r' {
		line = line[:len(line)-1]
	}

	message, err := NewMessage(string(line))
	if err != nil {
		return nil, err
	}

	return message, nil
}
