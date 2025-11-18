package control

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/fornellas/slogxt/log"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/fornellas/cgs/gcode"
	grblMod "github.com/fornellas/cgs/grbl"
)

type ControlOptions struct {
	DisplayStatusComms           bool
	DisplayGcodeParserStateComms bool
	DisplayGcodeParamStateComms  bool
}

type Control struct {
	grbl       *grblMod.Grbl
	options    *ControlOptions
	AppManager *AppManager
}

func NewControl(grbl *grblMod.Grbl, options *ControlOptions) *Control {
	if options == nil {
		options = &ControlOptions{}
	}
	return &Control{
		grbl:    grbl,
		options: options,
	}
}

func getMachineStateColor(state string) tcell.Color {
	switch state {
	case "Idle":
		return tcell.ColorGreen
	case "Run":
		return tcell.ColorLightCyan
	case "Hold":
		return tcell.ColorYellow
	case "Jog":
		return tcell.ColorBlue
	case "Alarm":
		return tcell.ColorRed
	case "Door":
		return tcell.ColorOrange
	case "Check":
		return tcell.ColorBlue
	case "Home":
		return tcell.ColorLime
	case "Sleep":
		return tcell.ColorSilver
	default:
		return tcell.ColorWhite
	}
}

//gocyclo:ignore
func (s *Control) sendCommand(
	ctx context.Context,
	command string,
) {
	// Extract and send real-time commands
	var buf bytes.Buffer
	for _, c := range []byte(command) {
		rtc, err := grblMod.NewRealTimeCommand(c)
		if err != nil {
			if !errors.Is(err, grblMod.ErrNotRealTimeCommand) {
				fmt.Fprintf(s.AppManager.CommandsTextView, "[%s]Real time command parsing fail: %s[-]\n", tcell.ColorRed, tview.Escape(err.Error()))
				return
			}
			buf.WriteByte(c)
		} else {
			s.sendRealTimeCommand(rtc)
		}
	}
	command = buf.String()

	if len(command) == 0 {
		return
	}

	// Verbosity & timeout
	var quiet bool
	timeout := 1 * time.Second
	parser := gcode.NewParser(strings.NewReader(command))
	for {
		block, err := parser.Next()
		if err != nil {
			fmt.Fprintf(s.AppManager.CommandsTextView, "[%s]Failed to parse: %s[-]\n", tcell.ColorRed, tview.Escape(err.Error()))
			return
		}
		if block == nil {
			break
		}
		if block.IsSystem() {
			switch block.String() {
			case "$G":
				if !s.options.DisplayGcodeParserStateComms {
					quiet = true
				}
			case "$#":
				if !s.options.DisplayGcodeParamStateComms {
					quiet = true
				}
			case "$H":
				timeout = 120 * time.Second
			}
		}
	}

	// send command
	if !quiet {
		fmt.Fprintf(s.AppManager.CommandsTextView, "[%s]%s[-]\n", tcell.ColorWhite, tview.Escape(command))
	}
	ctx, cancel := context.WithDeadline(ctx, time.Now().Add(timeout))
	defer cancel()
	messageResponse, err := s.grbl.SendCommand(ctx, command)
	if err != nil {
		fmt.Fprintf(s.AppManager.CommandsTextView, "[%s]Send command failed: %s[-]\n", tcell.ColorRed, tview.Escape(err.Error()))
		return
	}
	if quiet {
		return
	}
	if messageResponse.Error() == nil {
		fmt.Fprintf(s.AppManager.CommandsTextView, "[%s]%s[-]\n", tcell.ColorGreen, tview.Escape(messageResponse.String()))
	} else {
		fmt.Fprintf(s.AppManager.CommandsTextView, "[%s]%s[-]\n", tcell.ColorRed, tview.Escape(messageResponse.String()))
		fmt.Fprintf(s.AppManager.CommandsTextView, "[%s]%s[-]\n", tcell.ColorRed, tview.Escape(messageResponse.Error().Error()))
	}
}

func (s *Control) sendCommandWorker(
	ctx context.Context,
	sendCommandCh chan string,
) error {
	for {
		select {
		case <-ctx.Done():
			err := ctx.Err()
			if errors.Is(err, context.Canceled) {
				err = nil
			}
			return err
		case command := <-sendCommandCh:
			s.AppManager.CommandInputField.SetDisabled(true)
			s.AppManager.HomingButton.SetDisabled(true)
			s.AppManager.UnlockButton.SetDisabled(true)
			// s.shellApp.joggingButton.SetDisabled(true)
			s.AppManager.CheckButton.SetDisabled(true)
			s.AppManager.SleepButton.SetDisabled(true)
			// s.shellApp.settingsButton.SetDisabled(true)
			s.sendCommand(ctx, command)
			// Sending $G enables tracking of G-Code parsing state
			s.sendCommand(ctx, "$G")
			// Sending $G enables tracking of G-Code parameters
			s.sendCommand(ctx, "$#")
			s.AppManager.CommandInputField.SetText("")
			s.AppManager.CommandInputField.SetDisabled(false)
			s.AppManager.HomingButton.SetDisabled(false)
			s.AppManager.UnlockButton.SetDisabled(false)
			// s.shellApp.joggingButton.SetDisabled(false)
			s.AppManager.CheckButton.SetDisabled(false)
			s.AppManager.SleepButton.SetDisabled(false)
			// s.shellApp.settingsButton.SetDisabled(false)
		}
	}
}

func (s *Control) sendRealTimeCommand(
	cmd grblMod.RealTimeCommand,
) {
	if s.options.DisplayStatusComms || cmd != grblMod.RealTimeCommandStatusReportQuery {
		fmt.Fprintf(s.AppManager.CommandsTextView, "[%s]%s[-]\n", tcell.ColorWhite, tview.Escape(cmd.String()))
	}
	if err := s.grbl.SendRealTimeCommand(cmd); err != nil {
		fmt.Fprintf(s.AppManager.CommandsTextView, "[%s]Failed to send soft reset: %s[-]\n", tcell.ColorRed, err)
	}
}

func (s *Control) sendRealTimeCommandWorker(
	ctx context.Context,
	sendRealTimeCommandCh chan grblMod.RealTimeCommand,
) error {
	for {
		select {
		case <-ctx.Done():
			err := ctx.Err()
			if errors.Is(err, context.Canceled) {
				err = nil
			}
			return err
		case realTimeCommand := <-sendRealTimeCommandCh:
			s.sendRealTimeCommand(realTimeCommand)
		}
	}
}

func (s *Control) statusQueryWorker(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			err := ctx.Err()
			if errors.Is(err, context.Canceled) {
				err = nil
			}
			return err
		case <-time.After(200 * time.Millisecond):
			if err := s.grbl.SendRealTimeCommand(grblMod.RealTimeCommandStatusReportQuery); err != nil {
				return fmt.Errorf("failed to send periodic status query real-time command: %w", err)
			}
		}
	}
}

func (s *Control) Run(ctx context.Context) (err error) {
	logger := log.MustLogger(ctx)
	logger.Info("Connecting")

	pushMessageCh, err := s.grbl.Connect(ctx)
	if err != nil {
		return err
	}

	ctx, cancelFn := context.WithCancel(ctx)

	sendCommandCh := make(chan string, 10)
	sendCommandWorkerErrCh := make(chan error, 1)

	sendRealTimeCommandCh := make(chan grblMod.RealTimeCommand, 10)
	sendRealTimeCommandWorkerErrCh := make(chan error, 1)

	pushMessageErrCh := make(chan error, 1)

	statusQueryErrCh := make(chan error, 1)

	s.AppManager = NewAppManager(sendCommandCh, sendRealTimeCommandCh)
	defer func() { s.AppManager = nil }()

	logger = slog.New(NewViewLogHandler(
		logger.Handler(),
		s.AppManager.PushMessagesLogsTextView,
	))
	ctx = log.WithLogger(ctx, logger)

	go func() {
		defer cancelFn()
		defer s.AppManager.App.Stop()
		sendCommandWorkerErrCh <- s.sendCommandWorker(ctx, sendCommandCh)
	}()
	go func() {
		defer cancelFn()
		defer s.AppManager.App.Stop()
		sendRealTimeCommandWorkerErrCh <- s.sendRealTimeCommandWorker(
			ctx, sendRealTimeCommandCh,
		)
	}()
	go func() {
		defer cancelFn()
		defer s.AppManager.App.Stop()
		// Sending $G enables tracking of G-Code parsing state
		s.sendCommand(ctx, "$G")
		// Sending $G enables tracking of G-Code parameters
		s.sendCommand(ctx, "$#")
		messageProcessor := NewMessageProcessor(
			s.grbl,
			s.AppManager,
			pushMessageCh,
			s.sendCommand,
			!s.options.DisplayGcodeParserStateComms,
			!s.options.DisplayGcodeParamStateComms,
			!s.options.DisplayStatusComms,
		)
		pushMessageErrCh <- messageProcessor.Run(ctx)
	}()
	go func() {
		defer cancelFn()
		defer s.AppManager.App.Stop()
		statusQueryErrCh <- s.statusQueryWorker(ctx)
	}()

	defer func() {
		cancelFn()
		err = errors.Join(err, <-sendCommandWorkerErrCh)
		err = errors.Join(err, <-sendRealTimeCommandWorkerErrCh)
		err = errors.Join(err, <-pushMessageErrCh)
		err = errors.Join(err, <-statusQueryErrCh)
		err = errors.Join(err, s.grbl.Disconnect())
	}()
	return s.AppManager.App.Run()
}
