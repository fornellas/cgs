package grbl

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/fornellas/slogxt/log"
	"go.bug.st/serial"
)

type RealTimeCommand byte

var realTimeCommandStringsMap = map[RealTimeCommand]string{
	RealTimeCommandSoftReset:                                          "Soft-Reset",
	RealTimeCommandStatusReportQuery:                                  "Status Report Query",
	RealTimeCommandCycleStartResume:                                   "Cycle Start / Resume",
	RealTimeCommandFeedHold:                                           "Feed Hold",
	RealTimeCommandSafetyDoor:                                         "Safety Door",
	RealTimeCommandJogCancel:                                          "Jog Cancel",
	RealTimeCommandFeedOverrideSet100OfProgrammedRate:                 "Feed Override: Set 100% of programmed rate.",
	RealTimeCommandFeedOverrideIncrease10:                             "Feed Override: Increase 10%",
	RealTimeCommandFeedOverrideDecrease10:                             "Feed Override: Decrease 10%",
	RealTimeCommandFeedOverrideIncrease1:                              "Feed Override: Increase 1%",
	RealTimeCommandFeedOverrideDecrease1:                              "Feed Override: Decrease 1%",
	RealTimeCommandRapidOverrideSetTo100FullRapidRate:                 "Rapid Override: Set to 100% full rapid rate.",
	RealTimeCommandRapidOverrideSetTo50OfRapidRate:                    "Rapid Override: Set to 50% of rapid rate.",
	RealTimeCommandRapidOverrideSetTo25OfRapidRate:                    "Rapid Override: Set to 25% of rapid rate.",
	RealTimeCommandSpindleSpeedOverrideSet100OfProgrammedSpindleSpeed: "Spindle Speed Override: Set 100% of programmed spindle speed",
	RealTimeCommandSpindleSpeedOverrideIncrease10:                     "Spindle Speed Override: Increase 10%",
	RealTimeCommandSpindleSpeedOverrideDecrease10:                     "Spindle Speed Override: Decrease 10%",
	RealTimeCommandSpindleSpeedOverrideIncrease1:                      "Spindle Speed Override: Increase 1%",
	RealTimeCommandSpindleSpeedOverrideDecrease1:                      "Spindle Speed Override: Decrease 1%",
	RealTimeCommandToggleSpindleStop:                                  "Toggle Spindle Stop",
	RealTimeCommandToggleFloodCoolant:                                 "Toggle Flood Coolant",
	RealTimeCommandToggleMistCoolant:                                  "Toggle Mist Coolant",
}

func (c RealTimeCommand) String() string {
	if str, ok := realTimeCommandStringsMap[c]; ok {
		return str
	}
	return fmt.Sprintf("Unknown (%#v)", c)
}

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
	// WorkCoordinateOffset holds the newest value received via a status report.
	WorkCoordinateOffset *StatusReportWorkCoordinateOffset
	// OverrideValues holds the newest value received via a status report.
	OverrideValues *StatusReportOverrideValues
}

func NewGrbl(portName string) *Grbl {
	g := &Grbl{
		portName: portName,
	}
	return g
}

func (g *Grbl) mustLogger(ctx context.Context) (context.Context, *slog.Logger) {
	return log.MustWithGroupAttrs(ctx, "Grbl", "portName", g.portName)
}

func (g *Grbl) Open(ctx context.Context) error {
	ctx, logger := g.mustLogger(ctx)

	logger.Info("Connecting")

	mode := &serial.Mode{
		BaudRate: 115200,
	}
	logger.Debug("Opening")
	port, err := serial.Open(g.portName, mode)
	if err != nil {
		return fmt.Errorf("grbl: serial port open error: %s: %w", g.portName, err)
	}

	g.port = port
	g.WorkCoordinateOffset = nil
	g.OverrideValues = nil

	logger.Debug("Waiting for welcome message")
	receiveCtx, receiveCancel := context.WithDeadline(ctx, time.Now().Add(5*time.Second))
	defer receiveCancel()
	for {
		message, err := g.Receive(receiveCtx)
		if err != nil {
			g.port = nil
			closeErr := port.Close()
			if closeErr != nil {
				closeErr = fmt.Errorf("grbl: serial port close error: %s: %w", g.portName, closeErr)
			}
			return errors.Join(err, closeErr)
		}
		_, ok := message.(*MessagePushWelcome)
		if ok {
			break
		}
		logger.Debug("Ignoring", "message", message)
	}
	logger.Debug("Welcome message received")

	// we need to set this to allow context cancellation to work
	logger.Debug("Setting read timeout")
	if err := g.port.SetReadTimeout(50 * time.Millisecond); err != nil {
		g.port = nil
		closeErr := port.Close()
		if closeErr != nil {
			closeErr = fmt.Errorf("grbl: serial port close error: %s: %w", g.portName, closeErr)
		}
		return errors.Join(fmt.Errorf("grbl: error setting read timeout: %w", err), closeErr)
	}

	return nil
}

func (g *Grbl) Close(ctx context.Context) (err error) {
	_, logger := g.mustLogger(ctx)
	logger.Debug("Closing")
	if g.port != nil {
		err = g.port.Close()
		g.port = nil
		return
	}
	return
}

// SendRealTimeCommand issues a real time command to Grbl.
func (g *Grbl) SendRealTimeCommand(ctx context.Context, cmd RealTimeCommand) error {
	_, logger := g.mustLogger(ctx)
	logger.Debug("Sending real time command", "command", cmd)
	data := []byte{byte(cmd)}
	n, err := g.port.Write(data)
	if err != nil {
		return fmt.Errorf("grbl: write to serial port error: %s: %w", g.portName, err)
	}
	if n != len(data) {
		return fmt.Errorf("grbl: write to serial port error: %s: wrote %d bytes, expected %d", g.portName, n, len(data))
	}
	return nil
}

// TODO Jogging

// TODO Synchronization

func (g *Grbl) SendBlock(ctx context.Context, block string) error {
	_, logger := g.mustLogger(ctx)
	logger.Debug("Sending block", "block", block)
	line := append([]byte(block), '\n')
	n, err := g.port.Write(line)
	if err != nil {
		return fmt.Errorf("grbl: write to serial port error: %s: %w", g.portName, err)
	}
	if n != len(line) {
		return fmt.Errorf("grbl: write to serial port error: %s: wrote %d bytes, expected %d", g.portName, n, len(block))
	}
	return nil
}

// Receive message from Grbl.
func (g *Grbl) Receive(ctx context.Context) (Message, error) {
	ctx, logger := g.mustLogger(ctx)
	logger.Debug("Receiving message")
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
			continue
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

	if _, ok := message.(*MessagePushWelcome); ok {
		g.WorkCoordinateOffset = nil
		g.OverrideValues = nil
	}

	if messagePushStatusReport, ok := message.(*MessagePushStatusReport); ok {
		if messagePushStatusReport.WorkCoordinateOffset != nil {
			g.WorkCoordinateOffset = messagePushStatusReport.WorkCoordinateOffset
		}
		if messagePushStatusReport.OverrideValues != nil {
			g.OverrideValues = messagePushStatusReport.OverrideValues
		}
	}

	return message, nil
}
