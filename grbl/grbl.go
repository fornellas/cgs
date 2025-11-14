package grbl

import (
	"context"
	"errors"
	"fmt"
	"os"
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
	openPortFn func(*serial.Mode) (serial.Port, error)
	port       serial.Port
	// WorkCoordinateOffset holds the newest value received via a status report.
	WorkCoordinateOffset *StatusReportWorkCoordinateOffset
	// OverrideValues holds the newest value received via a status report.
	OverrideValues    *StatusReportOverrideValues
	receiveCtxCancel  context.CancelFunc
	pushMessageCh     chan Message
	responseMessageCh chan Message
	receiveDoneCh     chan error
}

func NewGrbl(openPortFn func(*serial.Mode) (serial.Port, error)) *Grbl {
	g := &Grbl{
		openPortFn: openPortFn,
	}
	return g
}

func (g *Grbl) receiveMessage(ctx context.Context) (Message, error) {
	logger := log.MustLogger(ctx)
	logger.Debug("Receiving message")
	line := []byte{}
	for {
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("grbl: receive message: context error: %w", err)
		}
		b := make([]byte, 1)

		n, err := g.port.Read(b)
		if err != nil && !errors.Is(err, os.ErrDeadlineExceeded) {
			return nil, fmt.Errorf("grbl: receive message: read error: %w", err)
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

	logger.Debug("Received message", "line", string(line))

	message, err := NewMessage(string(line))
	if err != nil {
		return nil, fmt.Errorf("grbl: receive message: bad message: %w", err)
	}

	if _, ok := message.(*MessagePushWelcome); ok {
		g.WorkCoordinateOffset = nil
		g.OverrideValues = nil
	}

	if messagePushStatusReport, ok := message.(*MessagePushStatusReport); ok {
		g.WorkCoordinateOffset = messagePushStatusReport.WorkCoordinateOffset
		g.OverrideValues = messagePushStatusReport.OverrideValues
	}

	return message, nil
}

// Connect opens the serial connection and waits for Grbl reset.
// On success, it returns a channel where push messages received from Grbl are sent to: this channel
// must be read from in a loop to process the push messages. On read errors, this channel will be
// closed.
// Disconnect() must be called when the connection won't be used anymore or when the message channel
// is closed.
func (g *Grbl) Connect(ctx context.Context) (chan Message, error) {
	logger := log.MustLogger(ctx)

	logger.Info("Connecting")

	mode := &serial.Mode{
		BaudRate: 115200,
		DataBits: 8,
		Parity:   serial.NoParity,
		StopBits: serial.OneStopBit,
	}
	logger.Debug("Opening")
	port, err := g.openPortFn(mode)
	if err != nil {
		return nil, fmt.Errorf("grbl: serial port open error: %w", err)
	}

	// we need to set this to allow polling reads to support context cancellation / timeout
	logger.Debug("Setting read timeout")
	if err := port.SetReadTimeout(100 * time.Millisecond); err != nil {
		closeErr := port.Close()
		if closeErr != nil {
			closeErr = fmt.Errorf("grbl: serial port close error: %w", closeErr)
		}
		return nil, errors.Join(fmt.Errorf("grbl: error setting read timeout: %w", err), closeErr)
	}

	g.port = port

	g.WorkCoordinateOffset = nil
	g.OverrideValues = nil

	var receiveCtx context.Context
	receiveCtx, g.receiveCtxCancel = context.WithCancel(ctx)
	g.pushMessageCh = make(chan Message)
	g.responseMessageCh = make(chan Message)
	g.receiveDoneCh = make(chan error, 1)

	go func() {
		for {
			message, err := g.receiveMessage(receiveCtx)
			if err != nil {
				logger.Debug("Receive message failed", "err", err)
				if errors.Is(err, context.Canceled) {
					err = nil
				}
				g.receiveDoneCh <- err
				close(g.pushMessageCh)
				close(g.responseMessageCh)
				return
			}
			switch message.Type() {
			case MessageTypePush:
				logger.Debug("Push message")
				g.pushMessageCh <- message
			case MessageTypeResponse:
				logger.Debug("Response message")
				g.responseMessageCh <- message
			default:
				panic(fmt.Sprintf("bug: unexpected message type: %#v", message.Type()))
			}
		}
	}()

	logger.Debug("Waiting for welcome message")
	welcomeCtx, welcomeCtxCancel := context.WithDeadline(ctx, time.Now().Add(5*time.Second))
	defer welcomeCtxCancel()
	for {
		select {
		case message, ok := <-g.pushMessageCh:
			if !ok {
				logger.Debug("Push message channel closed, disconnecting")
				return nil, g.Disconnect(ctx)
			}
			if _, ok := message.(*MessagePushWelcome); ok {
				logger.Debug("Connected")
				return g.pushMessageCh, nil
			}
			logger.Debug("Ignoring", "message", message)
		case <-welcomeCtx.Done():
			logger.Debug("Context done")
			return nil, errors.Join(welcomeCtx.Err(), g.Disconnect(ctx))
		}
	}
}

func (g *Grbl) Disconnect(ctx context.Context) (err error) {
	if g.port != nil {
		logger := log.MustLogger(ctx)
		logger.Debug("Cancelling receive context")
		g.receiveCtxCancel()
		err = <-g.receiveDoneCh
		logger.Debug("Closing serial port")
		err = errors.Join(err, g.port.Close())
		g.port = nil
	}
	return
}

// SendRealTimeCommand issues a real time command to Grbl.
func (g *Grbl) SendRealTimeCommand(ctx context.Context, cmd RealTimeCommand) error {
	logger := log.MustLogger(ctx)
	logger.Debug("Sending real time command", "command", cmd)
	data := []byte{byte(cmd)}
	n, err := g.port.Write(data)
	if err != nil {
		return fmt.Errorf("grbl: write to serial port error: %w", err)
	}
	if n != len(data) {
		return fmt.Errorf("grbl: write to serial port error: wrote %d bytes, expected %d", n, len(data))
	}
	return nil
}

// TODO Jogging

// TODO Synchronization

// Send a command / system command to Grbl synchronously.
// It waits for the response message and returns it.
func (g *Grbl) SendCommand(ctx context.Context, command string) (Message, error) {
	logger := log.MustLogger(ctx)
	logger.Debug("Sending block", "block", command)
	line := append([]byte(command), '\n')
	n, err := g.port.Write(line)
	if err != nil {
		return nil, fmt.Errorf("grbl: write to serial port error: %w", err)
	}
	if n != len(line) {
		return nil, fmt.Errorf("grbl: write to serial port error: wrote %d bytes, expected %d", n, len(command))
	}
	message, ok := <-g.responseMessageCh
	if !ok {
		return nil, fmt.Errorf("grbl: send command failed: message channel is closed")
	}
	messageResponse := message.(*MessageResponse)
	return messageResponse, nil
}

// TODO Streaming Protocol: Character-Counting
//   EEPROM Issues: can't stream with character counting for some commands:
//     write commands: G10 L2, G10 L20, G28.1, G30.1, $x=, $I=, $Nx=, $RST=
//     read commands: G54-G59, G28, G30, $$, $I, $N, $#
//   Check g-code with $C before sending
// func (g *Grbl)StreamProgram(r io.Reader) error {
// }
