package grbl

import (
	"errors"
	"fmt"

	"go.bug.st/serial"
)

type Grbl struct {
	port serial.Port
}

func NewGrbl(port serial.Port) *Grbl {
	g := &Grbl{
		port: port,
	}
	return g
}

func (g *Grbl) Connect() error {
	if err := g.port.ResetInputBuffer(); err != nil {
		return err
	}
	if err := g.port.ResetOutputBuffer(); err != nil {
		return err
	}

	// TODO wait for reset to complete:
	// │< "\r"                                                                                      │
	// │< "Grbl 1.1f ['$' for help]\r"                                                              │
	// │< "[MSG:'$H'|'$X' to unlock]\r"

	return nil
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
	var line []byte
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
