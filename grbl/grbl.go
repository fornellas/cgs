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

type Grbl struct {
	dataMu                     sync.Mutex
	commandMu                  sync.Mutex
	openPortFn                 func(context.Context, *serial.Mode) (serial.Port, error)
	port                       serial.Port
	workCoordinateOffset       *StatusReportWorkCoordinateOffset
	overrideValues             *StatusReportOverrideValues
	gcodeParameters            *GcodeParameters
	accessoryState             *StatusReportAccessoryState
	receiveCtxCancel           context.CancelFunc
	pushMessageCh              chan Message
	responseMessageCh          chan Message
	messageReceiverWorkerErrCh chan error
}

func NewGrbl(openPortFn func(context.Context, *serial.Mode) (serial.Port, error)) *Grbl {
	g := &Grbl{
		openPortFn: openPortFn,
	}
	return g
}

//gocyclo:ignore
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
		g.dataMu.Lock()
		g.workCoordinateOffset = nil
		g.overrideValues = nil
		g.gcodeParameters = &GcodeParameters{}
		g.accessoryState = nil
		g.dataMu.Unlock()
	}

	if messagePushStatusReport, ok := message.(*MessagePushStatusReport); ok {
		g.dataMu.Lock()
		if messagePushStatusReport.WorkCoordinateOffset != nil {
			g.workCoordinateOffset = messagePushStatusReport.WorkCoordinateOffset
		}
		if messagePushStatusReport.OverrideValues != nil {
			g.overrideValues = messagePushStatusReport.OverrideValues
		}
		if messagePushStatusReport.AccessoryState != nil {
			g.accessoryState = messagePushStatusReport.AccessoryState
		}
		g.dataMu.Unlock()
	}

	if messagePushGcodeParam, ok := message.(*MessagePushGcodeParam); ok {
		g.dataMu.Lock()
		g.gcodeParameters.Update(messagePushGcodeParam)
		g.dataMu.Unlock()
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
			g.dataMu.Lock()
			close(g.pushMessageCh)
			g.pushMessageCh = nil
			g.dataMu.Unlock()
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
			g.dataMu.Lock()
			close(g.pushMessageCh)
			g.pushMessageCh = nil
			g.dataMu.Unlock()
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
	mode := &serial.Mode{
		BaudRate: 115200,
		DataBits: 8,
		Parity:   serial.NoParity,
		StopBits: serial.OneStopBit,
	}

	port, err := g.openPortFn(ctx, mode)
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

	g.dataMu.Lock()

	g.port = port

	g.workCoordinateOffset = nil
	g.overrideValues = nil
	g.gcodeParameters = &GcodeParameters{}
	g.accessoryState = nil

	var receiveCtx context.Context
	receiveCtx, g.receiveCtxCancel = context.WithCancel(ctx)
	g.pushMessageCh = make(chan Message, 50)
	g.responseMessageCh = make(chan Message, 50)
	g.messageReceiverWorkerErrCh = make(chan error, 1)
	go g.messageReceiverWorker(receiveCtx)

	g.dataMu.Unlock()

	if err := g.waitForWelcomeMessage(ctx); err != nil {
		return nil, errors.Join(err, g.Disconnect(ctx))
	}

	return g.pushMessageCh, nil
}

// GetLastGetWorkCoordinateOffset returns the newest value received via a push message status report.
// Returns nil if no previous message was received.
func (g *Grbl) GetLastGetWorkCoordinateOffset() *StatusReportWorkCoordinateOffset {
	g.dataMu.Lock()
	defer g.dataMu.Unlock()
	return g.workCoordinateOffset
}

// GetLastGetOverrideValues returns the newest value received via a push message status report.
// Returns nil if no previous message was received.
func (g *Grbl) GetLastGetOverrideValues() *StatusReportOverrideValues {
	g.dataMu.Lock()
	defer g.dataMu.Unlock()
	return g.overrideValues
}

// GetLastGetGcodeParameters returns the newest value received via a push message gcode parameters.
// Returns nil if no previous message was received.
func (g *Grbl) GetLastGetGcodeParameters() *GcodeParameters {
	g.dataMu.Lock()
	defer g.dataMu.Unlock()
	return g.gcodeParameters
}

// GetLastAccessoryState returns the newest value received via a push message status report.
// Returns nil if no previous message was received.
func (g *Grbl) GetLastAccessoryState() *StatusReportAccessoryState {
	g.dataMu.Lock()
	defer g.dataMu.Unlock()
	return g.accessoryState
}

// SendRealTimeCommand issues a real time command to Grbl.
func (g *Grbl) SendRealTimeCommand(cmd RealTimeCommand) error {
	g.dataMu.Lock()
	defer g.dataMu.Unlock()
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

// Send a command / system command to Grbl synchronously.
// It waits for the response message and returns it.
func (g *Grbl) SendCommand(ctx context.Context, command string) (*MessageResponse, error) {
	if strings.Contains(command, "\n") {
		return nil, fmt.Errorf("command must be single line string: %#v", command)
	}

	g.commandMu.Lock()
	defer g.commandMu.Unlock()

	g.dataMu.Lock()
	if g.port == nil {
		g.dataMu.Unlock()
		return nil, fmt.Errorf("grbl: disconnected")
	}
	line := append([]byte(command), '\n')
	n, err := g.port.Write(line)
	g.dataMu.Unlock()
	if err != nil {
		return nil, fmt.Errorf("grbl: write to serial port error: %w", err)
	}
	if n != len(line) {
		return nil, fmt.Errorf("grbl: write to serial port error: wrote %d bytes, expected %d", n, len(command))
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

	return message.(*MessageResponse), nil
}

// Disconnect will stop all goroutines and close the serial port.
func (g *Grbl) Disconnect(ctx context.Context) (err error) {
	g.dataMu.Lock()
	if g.port == nil {
		g.dataMu.Unlock()
		return
	}
	g.receiveCtxCancel()
	g.dataMu.Unlock()

	err = <-g.messageReceiverWorkerErrCh

	g.dataMu.Lock()
	defer g.dataMu.Unlock()
	close(g.responseMessageCh)
	close(g.messageReceiverWorkerErrCh)
	err = errors.Join(err, g.port.Close())
	g.port = nil
	g.workCoordinateOffset = nil
	g.overrideValues = nil
	g.gcodeParameters = &GcodeParameters{}
	g.accessoryState = nil
	g.receiveCtxCancel = nil
	g.pushMessageCh = nil
	g.responseMessageCh = nil
	g.messageReceiverWorkerErrCh = nil
	return
}
