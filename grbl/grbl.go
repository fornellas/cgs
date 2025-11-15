package grbl

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

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
	mu                         sync.Mutex
	openPortFn                 func(*serial.Mode) (serial.Port, error)
	port                       serial.Port
	workCoordinateOffset       *StatusReportWorkCoordinateOffset
	overrideValues             *StatusReportOverrideValues
	receiveCtxCancel           context.CancelFunc
	pushMessageCh              chan Message
	responseMessageCh          chan Message
	messageReceiverWorkerErrCh chan error
}

func NewGrbl(openPortFn func(*serial.Mode) (serial.Port, error)) *Grbl {
	g := &Grbl{
		openPortFn: openPortFn,
	}
	return g
}

func (g *Grbl) receiveMessage(ctx context.Context) (Message, error) {
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

	message, err := NewMessage(string(line))
	if err != nil {
		return nil, fmt.Errorf("grbl: receive message: bad message: %w", err)
	}

	if _, ok := message.(*MessagePushWelcome); ok {
		g.workCoordinateOffset = nil
		g.overrideValues = nil
	}

	if messagePushStatusReport, ok := message.(*MessagePushStatusReport); ok {
		if messagePushStatusReport.WorkCoordinateOffset != nil {
			g.workCoordinateOffset = messagePushStatusReport.WorkCoordinateOffset
		}
		if messagePushStatusReport.OverrideValues != nil {
			g.overrideValues = messagePushStatusReport.OverrideValues
		}
	}

	return message, nil
}

func (g *Grbl) messageReceiverWorker(ctx context.Context) {
	for {
		message, err := g.receiveMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				err = nil
			}
			g.mu.Lock()
			close(g.pushMessageCh)
			g.pushMessageCh = nil
			g.mu.Unlock()
			g.messageReceiverWorkerErrCh <- err
			return
		}

		var messageCh chan Message
		switch message.Type() {
		case MessageTypePush:
			messageCh = g.pushMessageCh
		case MessageTypeResponse:
			messageCh = g.responseMessageCh
		default:
			panic(fmt.Sprintf("bug: unexpected message type: %#v", message.Type()))
		}

		select {
		case messageCh <- message:
		case <-ctx.Done():
			g.mu.Lock()
			close(g.pushMessageCh)
			g.pushMessageCh = nil
			g.mu.Unlock()
			g.messageReceiverWorkerErrCh <- nil
			return
		}
	}
}

func (g *Grbl) waitForWelcomeMessage(ctx context.Context) error {
	welcomeCtx, welcomeCtxCancel := context.WithDeadline(ctx, time.Now().Add(5*time.Second))
	defer welcomeCtxCancel()
	for {
		select {
		case message, ok := <-g.pushMessageCh:
			if !ok {
				return errors.New("grbl: push message channel closed before welcome message received")
			}
			if _, ok := message.(*MessagePushWelcome); ok {
				return nil
			}
		case <-welcomeCtx.Done():
			return welcomeCtx.Err()
		}
	}
}

// Connect opens the serial connection and waits for Grbl welcome push message before returning.
// On success, it returns a channel where push messages received from Grbl are sent to: this channel
// must be read from in a loop to process the push messages. On read errors, the push messages
// channel will be closed, Disconnect() must be called in this case, and it'll return the error.
// Disconnect() must be called when the connection isn't needed anymore.
//
//gocyclo:ignore
func (g *Grbl) Connect(ctx context.Context) (chan Message, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	mode := &serial.Mode{
		BaudRate: 115200,
		DataBits: 8,
		Parity:   serial.NoParity,
		StopBits: serial.OneStopBit,
	}
	port, err := g.openPortFn(mode)
	if err != nil {
		return nil, fmt.Errorf("grbl: serial port open error: %w", err)
	}

	// we need to set this to allow polling reads to support context cancellation / timeout
	if err := port.SetReadTimeout(100 * time.Millisecond); err != nil {
		closeErr := port.Close()
		if closeErr != nil {
			closeErr = fmt.Errorf("grbl: serial port close error: %w", closeErr)
		}
		return nil, errors.Join(fmt.Errorf("grbl: error setting read timeout: %w", err), closeErr)
	}

	g.port = port

	g.workCoordinateOffset = nil
	g.overrideValues = nil

	var receiveCtx context.Context
	receiveCtx, g.receiveCtxCancel = context.WithCancel(ctx)
	g.pushMessageCh = make(chan Message, 50)
	g.responseMessageCh = make(chan Message, 50)
	g.messageReceiverWorkerErrCh = make(chan error, 1)
	go g.messageReceiverWorker(receiveCtx)

	if err := g.waitForWelcomeMessage(ctx); err != nil {
		return nil, errors.Join(err, g.Disconnect(ctx))
	}

	return g.pushMessageCh, nil
}

// GetWorkCoordinateOffset returns the newest value received via a push message status report.
// Returns nil if no previous message was received.
func (g *Grbl) GetWorkCoordinateOffset() *StatusReportWorkCoordinateOffset {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.workCoordinateOffset
}

// GetOverrideValues returns the newest value received via a push message status report.
// Returns nil if no previous message was received.
func (g *Grbl) GetOverrideValues() *StatusReportOverrideValues {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.overrideValues
}

// SendRealTimeCommand issues a real time command to Grbl.
func (g *Grbl) SendRealTimeCommand(ctx context.Context, cmd RealTimeCommand) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.port == nil {
		return fmt.Errorf("grbl: disconnected")
	}
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
	if strings.Contains(command, "\n") {
		return nil, fmt.Errorf("command must be single line string: %#v", command)
	}

	{
		g.mu.Lock()
		defer g.mu.Unlock()
		if g.port == nil {
			return nil, fmt.Errorf("grbl: disconnected")
		}
		line := append([]byte(command), '\n')
		n, err := g.port.Write(line)
		if err != nil {
			return nil, fmt.Errorf("grbl: write to serial port error: %w", err)
		}
		if n != len(line) {
			return nil, fmt.Errorf("grbl: write to serial port error: wrote %d bytes, expected %d", n, len(command))
		}
	}

	var ok bool

	// If a previous command context is cancelled / deadline was exceeded before the response
	// message is processed, it'll still be in the buffer. This ensures the buffer is empty before
	// we send the next command, ensuring the response message we get, is related to this command,
	// not the previous.
	for {
		if len(g.responseMessageCh) == 0 {
			break
		}
		select {
		case _, ok = <-g.responseMessageCh:
			if !ok {
				return nil, fmt.Errorf("grbl: command failed: response message channel is closed")
			}
		case <-ctx.Done():
			return nil, fmt.Errorf("grbl: command failed: %w", ctx.Err())
		}
	}

	var message Message
	select {
	case message, ok = <-g.responseMessageCh:
		if !ok {
			return nil, fmt.Errorf("grbl: command failed: response message channel is closed")
		}
	case <-ctx.Done():
		return nil, fmt.Errorf("grbl: command failed: %w", ctx.Err())
	}
	messageResponse := message.(*MessageResponse)
	return messageResponse, nil
}

// DRAFT generated by Claude
// // StreamProgram streams a G-code program to Grbl using the Character-Counting protocol.
// // It parses each line and fails if EEPROM commands are found.
// func (g *Grbl) StreamProgram(ctx context.Context, r io.Reader) error {
// 	// EEPROM commands that cannot be used with character-counting protocol
// 	eepromCommands := map[string]bool{
// 		// Write commands
// 		"G10":   true, // Will check L parameter
// 		"G28.1": true,
// 		"G30.1": true,
// 		// System commands starting with $
// 		"$": true,
// 		// Read commands
// 		"G54": true, "G55": true, "G56": true, "G57": true, "G58": true, "G59": true,
// 		"G28": true, "G30": true,
// 	}

// 	// Parse and collect all lines first, checking for EEPROM commands
// 	var lines []string
// 	scanner := bufio.NewScanner(r)
// 	lineNum := 0

// 	for scanner.Scan() {
// 		lineNum++
// 		line := strings.TrimSpace(scanner.Text())

// 		// Skip empty lines and comments
// 		if line == "" || strings.HasPrefix(line, ";") || strings.HasPrefix(line, "(") {
// 			continue
// 		}

// 		// Check for system commands (start with $)
// 		if strings.HasPrefix(line, "$") {
// 			return fmt.Errorf("grbl: line %d: EEPROM command not allowed with character-counting: %s", lineNum, line)
// 		}

// 		// Parse the line to check for G-code EEPROM commands
// 		parser := gcode.NewParser(strings.NewReader(line))
// 		block, err := parser.Next()
// 		if err != nil {
// 			return fmt.Errorf("grbl: line %d: parse error: %w", lineNum, err)
// 		}

// 		if block != nil && block.IsCommand() {
// 			for _, word := range block.Commands() {
// 				wordStr := word.NormalizedString()

// 				// Check for EEPROM commands
// 				if eepromCommands[wordStr] {
// 					// Special case for G10 - check L parameter
// 					if wordStr == "G10" {
// 						lParam, err := block.GetArgumentNumber('L')
// 						if err == nil && lParam != nil {
// 							// G10 with L2 or L20 is EEPROM command
// 							if *lParam == 2 || *lParam == 20 {
// 								return fmt.Errorf("grbl: line %d: EEPROM command not allowed with character-counting: %s", lineNum, line)
// 							}
// 						} else {
// 							// G10 without L parameter is also EEPROM command
// 							return fmt.Errorf("grbl: line %d: EEPROM command not allowed with character-counting: %s", lineNum, line)
// 						}
// 					} else {
// 						return fmt.Errorf("grbl: line %d: EEPROM command not allowed with character-counting: %s", lineNum, line)
// 					}
// 				}
// 			}
// 		}

// 		lines = append(lines, line)
// 	}

// 	if err := scanner.Err(); err != nil {
// 		return fmt.Errorf("grbl: read error: %w", err)
// 	}

// 	// Check connection after EEPROM validation
// 	if g.port == nil {
// 		return fmt.Errorf("grbl: not connected")
// 	}

// 	// Stream using character-counting protocol
// 	const maxBufferSize = 128
// 	var pendingLines []string
// 	var pendingBytes int

// 	for _, line := range lines {
// 		lineWithNewline := line + "\n"
// 		lineBytes := len(lineWithNewline)

// 		// If this line would overflow the buffer, wait for responses
// 		for pendingBytes+lineBytes > maxBufferSize {
// 			if len(pendingLines) == 0 {
// 				return fmt.Errorf("grbl: line too long for buffer: %s", line)
// 			}

// 			// Wait for a response
// 			select {
// 			case message, ok := <-g.responseMessageCh:
// 				if !ok {
// 					return fmt.Errorf("grbl: response message channel is closed")
// 				}

// 				// Check for errors
// 				if responseMsg, ok := message.(*MessageResponse); ok {
// 					if strings.HasPrefix(responseMsg.Message, "error:") {
// 						return fmt.Errorf("grbl: %s", responseMsg.Message)
// 					}
// 				}

// 				// Remove the first pending line and subtract its byte count
// 				if len(pendingLines) > 0 {
// 					executedLine := pendingLines[0] + "\n"
// 					pendingLines = pendingLines[1:]
// 					pendingBytes -= len(executedLine)
// 				}

// 			case <-ctx.Done():
// 				return ctx.Err()

// 			case <-time.After(5 * time.Second):
// 				return fmt.Errorf("grbl: timeout waiting for response")
// 			}
// 		}

// 		// Send the line
// 		n, err := g.port.Write([]byte(lineWithNewline))
// 		if err != nil {
// 			return fmt.Errorf("grbl: write to serial port error: %w", err)
// 		}
// 		if n != len(lineWithNewline) {
// 			return fmt.Errorf("grbl: write to serial port error: wrote %d bytes, expected %d", n, len(lineWithNewline))
// 		}

// 		// Track the sent line
// 		pendingLines = append(pendingLines, line)
// 		pendingBytes += lineBytes
// 	}

// 	// Wait for all remaining responses
// 	for len(pendingLines) > 0 {
// 		select {
// 		case message, ok := <-g.responseMessageCh:
// 			if !ok {
// 				return fmt.Errorf("grbl: response message channel is closed")
// 			}

// 			// Check for errors
// 			if responseMsg, ok := message.(*MessageResponse); ok {
// 				if strings.HasPrefix(responseMsg.Message, "error:") {
// 					return fmt.Errorf("grbl: %s", responseMsg.Message)
// 				}
// 			}

// 			// Remove the first pending line
// 			if len(pendingLines) > 0 {
// 				pendingLines = pendingLines[1:]
// 			}

// 		case <-ctx.Done():
// 			return ctx.Err()

// 		case <-time.After(5 * time.Second):
// 			return fmt.Errorf("grbl: timeout waiting for response")
// 		}
// 	}

// 	return nil
// }

// Disconnect will stop all goroutines and close the serial port.
func (g *Grbl) Disconnect(ctx context.Context) (err error) {
	g.mu.Lock()
	if g.port == nil {
		g.mu.Unlock()
		return
	}
	g.receiveCtxCancel()
	g.mu.Unlock()

	err = <-g.messageReceiverWorkerErrCh

	g.mu.Lock()
	defer g.mu.Unlock()
	close(g.responseMessageCh)
	close(g.messageReceiverWorkerErrCh)
	err = errors.Join(err, g.port.Close())
	g.port = nil
	g.workCoordinateOffset = nil
	g.overrideValues = nil
	g.receiveCtxCancel = nil
	g.pushMessageCh = nil
	g.responseMessageCh = nil
	g.messageReceiverWorkerErrCh = nil
	return
}
