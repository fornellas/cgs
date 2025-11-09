package grbl

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.bug.st/serial"
)

type RealTimeCommand byte

var (
	// Soft-Reset
	RealTimeCommandSoftReset RealTimeCommand = 0x18
	// Status Report Query
	RealTimeCommandStatusReportQuery RealTimeCommand = '?'
	// Cycle Start / Resume
	RealTimeCommandCycleStartResume RealTimeCommand = '~'
	// Feed Hold
	RealTimeCommandFeedHold RealTimeCommand = '!'
	// Safety Door
	RealTimeCommandSafetyDoor RealTimeCommand = 0x84
	// Jog Cancel
	RealTimeCommandJogCancel RealTimeCommand = 0x85
	// Feed Override: Set 100% of programmed rate.
	RealTimeCommandFeedOverrideSet100OfProgrammedRate RealTimeCommand = 0x90
	// Feed Override: Increase 10%
	RealTimeCommandFeedOverrideIncrease10 RealTimeCommand = 0x91
	// Feed Override: Decrease 10%
	RealTimeCommandFeedOverrideDecrease10 RealTimeCommand = 0x92
	// Feed Override: Increase 1%
	RealTimeCommandFeedOverrideIncrease1 RealTimeCommand = 0x93
	// Feed Override: Decrease 1%
	RealTimeCommandFeedOverrideDecrease1 RealTimeCommand = 0x94
	// Rapid Override: Set to 100% full rapid rate.
	RealTimeCommandRapidOverrideSetTo100FullRapidRate RealTimeCommand = 0x95
	// Rapid Override: Set to 50% of rapid rate.
	RealTimeCommandRapidOverrideSetTo50OfRapidRate RealTimeCommand = 0x96
	// Rapid Override: Set to 25% of rapid rate.
	RealTimeCommandRapidOverrideSetTo25OfRapidRate RealTimeCommand = 0x97
	// Spindle Speed Override: Set 100% of programmed spindle speed
	RealTimeCommandSpindleSpeedOverrideSet100OfProgrammedSpindleSpeed RealTimeCommand = 0x99
	// Spindle Speed Override: Increase 10%
	RealTimeCommandSpindleSpeedOverrideIncrease10 RealTimeCommand = 0x9A
	// Spindle Speed Override: Decrease 10%
	RealTimeCommandSpindleSpeedOverrideDecrease10 RealTimeCommand = 0x9B
	// Spindle Speed Override: Increase 1%
	RealTimeCommandSpindleSpeedOverrideIncrease1 RealTimeCommand = 0x9C
	// Spindle Speed Override: Decrease 1%
	RealTimeCommandSpindleSpeedOverrideDecrease1 RealTimeCommand = 0x9D
	// Toggle Spindle Stop
	RealTimeCommandToggleSpindleStop RealTimeCommand = 0x9E
	// Toggle Flood Coolant
	RealTimeCommandToggleFloodCoolant RealTimeCommand = 0xA0
	// Toggle Mist Coolant
	RealTimeCommandToggleMistCoolant RealTimeCommand = 0xA1
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

// SendRealTimeCommand issues a real time command to Grbl.
func (g *Grbl) SendRealTimeCommand(cmd RealTimeCommand) error {
	data := []byte{byte(cmd)}
	n, err := g.port.Write(data)
	if err != nil {
		return fmt.Errorf("grbl: write to serial port error: %s: %w", g.portName, err)
	}
	if n != len(data) {
		return fmt.Errorf("grbl: write to serial port error: %s: wrote %d bytes, expected %d", g.portName, n, len(data))
	}
	if err := g.port.Drain(); err != nil {
		return fmt.Errorf("grbl: serial port drain error: %s: %w", g.portName, err)
	}
	return nil
}

// TODO Jogging

// TODO Synchronization

func (g *Grbl) SendBlock(block string) error {
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
