package grbl

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"go.bug.st/serial"
)

var ErrInvalidMessage = errors.New("invalid Grbl message")

type Grbl struct {
	grblMu                     sync.Mutex
	portWriteMu                sync.Mutex
	openPortFn                 func(context.Context, *serial.Mode) (serial.Port, error)
	port                       serial.Port
	workCoordinateOffset       *WorkCoordinateOffset
	overrideValues             *OverrideValues
	gcodeParameters            *GcodeParameters
	accessoryState             *AccessoryState
	receiveCtxCancel           context.CancelFunc
	pushMessageCh              chan PushMessage
	responseMessageCh          chan *ResponseMessage
	messageReceiverWorkerErrCh chan error
}

func NewGrbl(openPortFn func(context.Context, *serial.Mode) (serial.Port, error)) *Grbl {
	g := &Grbl{
		openPortFn: openPortFn,
	}
	return g
}

//gocyclo:ignore
func (g *Grbl) receiveMessage(ctx context.Context) (PushMessage, *ResponseMessage, error) {
	messageBytes := []byte{}
	for {
		if err := ctx.Err(); err != nil {
			return nil, nil, fmt.Errorf("receive message: context error: %w", err)
		}
		b := make([]byte, 1)

		n, err := g.port.Read(b)
		if err != nil && !errors.Is(err, os.ErrDeadlineExceeded) {
			return nil, nil, fmt.Errorf("receive message: read error: %w", err)
		}
		if n == 0 {
			continue
		}
		if b[0] == '\n' {
			break
		}
		messageBytes = append(messageBytes, b[0])
	}

	if len(messageBytes) >= 1 && messageBytes[len(messageBytes)-1] == '\r' {
		messageBytes = messageBytes[:len(messageBytes)-1]
	}

	pushMessage, err := NewPushMessage(string(messageBytes))
	if err != nil {
		if err != ErrInvalidMessage {
			return nil, nil, fmt.Errorf("receive message: bad message: %w", err)
		}
	} else {
		if _, ok := pushMessage.(*WelcomePushMessage); ok {
			g.grblMu.Lock()
			g.workCoordinateOffset = nil
			g.overrideValues = nil
			g.gcodeParameters = &GcodeParameters{}
			g.accessoryState = nil
			g.grblMu.Unlock()
		}

		if statusReportPushMessage, ok := pushMessage.(*StatusReportPushMessage); ok {
			g.grblMu.Lock()
			if statusReportPushMessage.WorkCoordinateOffset != nil {
				g.workCoordinateOffset = statusReportPushMessage.WorkCoordinateOffset
			}
			if statusReportPushMessage.OverrideValues != nil {
				g.overrideValues = statusReportPushMessage.OverrideValues
			}
			if statusReportPushMessage.AccessoryState != nil {
				g.accessoryState = statusReportPushMessage.AccessoryState
			}
			g.grblMu.Unlock()
		}

		if gcodeParamPushMessage, ok := pushMessage.(*GcodeParamPushMessage); ok {
			g.grblMu.Lock()
			g.gcodeParameters.Update(gcodeParamPushMessage)
			g.grblMu.Unlock()
		}
		return pushMessage, nil, nil
	}

	responseMessage, err := NewMessageResponse(string(messageBytes))
	if err != nil {
		if err != ErrInvalidMessage {
			return nil, nil, fmt.Errorf("receive message: bad message: %w", err)
		}
	} else {
		return nil, responseMessage, nil
	}

	panic(fmt.Sprintf("bug: unknown message: %#v", string(messageBytes)))
}

func (g *Grbl) messageReceiverWorker(ctx context.Context) {
	for {
		pushMessage, responseMessage, err := g.receiveMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				err = nil
			}
			g.grblMu.Lock()
			close(g.pushMessageCh)
			g.pushMessageCh = nil
			g.grblMu.Unlock()
			g.messageReceiverWorkerErrCh <- err
			return
		}

		if pushMessage != nil {
			select {
			case g.pushMessageCh <- pushMessage:
			case <-ctx.Done():
				g.grblMu.Lock()
				close(g.pushMessageCh)
				g.pushMessageCh = nil
				g.grblMu.Unlock()
				g.messageReceiverWorkerErrCh <- nil
				return
			}
		}

		if responseMessage != nil {
			select {
			case g.responseMessageCh <- responseMessage:
			case <-ctx.Done():
				g.grblMu.Lock()
				close(g.pushMessageCh)
				g.pushMessageCh = nil
				g.grblMu.Unlock()
				g.messageReceiverWorkerErrCh <- nil
				return
			}
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
			if _, ok := message.(*WelcomePushMessage); ok {
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
func (g *Grbl) Connect(ctx context.Context) (chan PushMessage, error) {
	mode := &serial.Mode{
		BaudRate: 115200,
		DataBits: 8,
		Parity:   serial.NoParity,
		StopBits: serial.OneStopBit,
	}

	port, err := g.openPortFn(ctx, mode)
	if err != nil {
		return nil, fmt.Errorf("serial port open error: %w", err)
	}

	// we need to set this to allow polling reads to support context cancellation / timeout
	if err := port.SetReadTimeout(100 * time.Millisecond); err != nil {
		closeErr := port.Close()
		if closeErr != nil {
			closeErr = fmt.Errorf("serial port close error: %w", closeErr)
		}
		return nil, errors.Join(fmt.Errorf("error setting read timeout: %w", err), closeErr)
	}

	g.grblMu.Lock()

	g.port = port

	g.workCoordinateOffset = nil
	g.overrideValues = nil
	g.gcodeParameters = &GcodeParameters{}
	g.accessoryState = nil

	var receiveCtx context.Context
	receiveCtx, g.receiveCtxCancel = context.WithCancel(ctx)
	g.pushMessageCh = make(chan PushMessage, 50)
	g.responseMessageCh = make(chan *ResponseMessage, 50)
	g.messageReceiverWorkerErrCh = make(chan error, 1)
	go g.messageReceiverWorker(receiveCtx)

	g.grblMu.Unlock()

	if err := g.waitForWelcomeMessage(ctx); err != nil {
		return nil, errors.Join(err, g.Disconnect(ctx))
	}

	return g.pushMessageCh, nil
}

// GetLastWorkCoordinateOffset returns the newest value received via a push message status report.
// Returns nil if no previous message was received.
func (g *Grbl) GetLastWorkCoordinateOffset() *WorkCoordinateOffset {
	g.grblMu.Lock()
	defer g.grblMu.Unlock()
	return g.workCoordinateOffset
}

// GetLastOverrideValues returns the newest value received via a push message status report.
// Returns nil if no previous message was received.
func (g *Grbl) GetLastOverrideValues() *OverrideValues {
	g.grblMu.Lock()
	defer g.grblMu.Unlock()
	return g.overrideValues
}

// GetLastGcodeParameters returns the newest value received via a push message gcode parameters.
// Returns nil if no previous message was received.
func (g *Grbl) GetLastGcodeParameters() *GcodeParameters {
	g.grblMu.Lock()
	defer g.grblMu.Unlock()
	return g.gcodeParameters
}

// GetLastAccessoryState returns the newest value received via a push message status report.
// Returns nil if no previous message was received.
func (g *Grbl) GetLastAccessoryState() *AccessoryState {
	g.grblMu.Lock()
	defer g.grblMu.Unlock()
	return g.accessoryState
}

// SendRealTimeCommand issues a real time command to Grbl.
func (g *Grbl) SendRealTimeCommand(cmd RealTimeCommand) error {
	g.grblMu.Lock()
	defer g.grblMu.Unlock()
	if g.port == nil {
		return fmt.Errorf("disconnected")
	}
	data := []byte{byte(cmd)}
	n, err := g.port.Write(data)
	if err != nil {
		return fmt.Errorf("write to serial port error: %w", err)
	}
	if n != len(data) {
		return fmt.Errorf("write to serial port error: wrote %d bytes, expected %d", n, len(data))
	}
	return nil
}

// If a previous command context is cancelled / deadline was exceeded before the response
// message is processed, it'll still be in the buffer. This ensures the buffer is empty before
// we send the next command, ensuring the response message we get, is related to this command,
// not the previous.
func (g *Grbl) emptyResponseMessageCh(ctx context.Context) error {
	for {
		if len(g.responseMessageCh) == 0 {
			break
		}
		select {
		case _, ok := <-g.responseMessageCh:
			if !ok {
				return fmt.Errorf("command failed: response message channel is closed")
			}
		case <-ctx.Done():
			return fmt.Errorf("command failed: %w", ctx.Err())
		}
	}

	return nil
}

// Send a command / system command to Grbl synchronously.
// It waits for the response message.
func (g *Grbl) SendCommand(ctx context.Context, command string) error {
	if strings.Contains(command, "\n") {
		return fmt.Errorf("command must be single line string: %#v", command)
	}

	g.portWriteMu.Lock()
	defer g.portWriteMu.Unlock()

	g.grblMu.Lock()
	if g.port == nil {
		g.grblMu.Unlock()
		return fmt.Errorf("disconnected")
	}
	line := append([]byte(command), '\n')
	n, err := g.port.Write(line)
	g.grblMu.Unlock()
	if err != nil {
		return fmt.Errorf("write to serial port error: %w", err)
	}
	if n != len(line) {
		return fmt.Errorf("write to serial port error: wrote %d bytes, expected %d", n, len(command))
	}

	var ok bool

	if err := g.emptyResponseMessageCh(ctx); err != nil {
		return err
	}

	var responseMessage *ResponseMessage
	select {
	case responseMessage, ok = <-g.responseMessageCh:
		if !ok {
			return fmt.Errorf("command failed: response message channel is closed")
		}
	case <-ctx.Done():
		return fmt.Errorf("command failed: %w", ctx.Err())
	}

	return responseMessage.Error()
}

func (g *Grbl) StreamProgram(ctx context.Context, programReader io.Reader) error {
	g.portWriteMu.Lock()
	defer g.portWriteMu.Unlock()

	if err := g.emptyResponseMessageCh(ctx); err != nil {
		return err
	}

	// TODO call $I to check [OPT: response to fetch serial RX buffer bytes
	const maxSerialRxBufferBytes = 128
	return NewProgramStreamer(g.port, g.responseMessageCh, maxSerialRxBufferBytes).Run(ctx, programReader)
}

// Disconnect will stop all goroutines and close the serial port.
func (g *Grbl) Disconnect(ctx context.Context) (err error) {
	g.grblMu.Lock()
	if g.port == nil {
		g.grblMu.Unlock()
		return
	}
	g.receiveCtxCancel()
	g.grblMu.Unlock()

	err = <-g.messageReceiverWorkerErrCh

	g.grblMu.Lock()
	defer g.grblMu.Unlock()
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
