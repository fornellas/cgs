package control

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/fornellas/cgs/gcode"
	grblMod "github.com/fornellas/cgs/grbl"
)

type CommandDispatcher struct {
	grbl                       *grblMod.Grbl
	appManager                 *AppManager
	quietGcodeParserStateComms bool
	quietGcodeParamStateComms  bool
	quietStatusComms           bool
	sendCommandCh              chan string
	sendRealTimeCommandCh      chan grblMod.RealTimeCommand
}

func NewCommandDispatcher(
	grbl *grblMod.Grbl,
	appManager *AppManager,
	quietGcodeParserStateComms bool,
	quietGcodeParamStateComms bool,
	quietStatusComms bool,
) *CommandDispatcher {
	return &CommandDispatcher{
		grbl:                       grbl,
		appManager:                 appManager,
		quietGcodeParserStateComms: quietGcodeParserStateComms,
		quietGcodeParamStateComms:  quietGcodeParamStateComms,
		quietStatusComms:           quietStatusComms,
		sendCommandCh:              make(chan string, 10),
		sendRealTimeCommandCh:      make(chan grblMod.RealTimeCommand, 10),
	}
}

//gocyclo:ignore
func (cd *CommandDispatcher) sendCommand(
	ctx context.Context,
	command string,
) {
	// Extract and send real-time commands
	var buf bytes.Buffer
	for _, c := range []byte(command) {
		rtc, err := grblMod.NewRealTimeCommand(c)
		if err != nil {
			if !errors.Is(err, grblMod.ErrNotRealTimeCommand) {
				fmt.Fprintf(cd.appManager.CommandsTextView, "[%s]Real time command parsing fail: %s[-]\n", tcell.ColorRed, tview.Escape(err.Error()))
				return
			}
			buf.WriteByte(c)
		} else {
			cd.sendRealTimeCommand(rtc)
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
			fmt.Fprintf(cd.appManager.CommandsTextView, "[%s]Failed to parse: %s[-]\n", tcell.ColorRed, tview.Escape(err.Error()))
			return
		}
		if block == nil {
			break
		}
		if block.IsSystem() {
			switch block.String() {
			case "$G":
				if cd.quietGcodeParserStateComms {
					quiet = true
				}
			case "$#":
				if cd.quietGcodeParamStateComms {
					quiet = true
				}
			case "$H":
				timeout = 120 * time.Second
			}
		}
	}

	// send command
	if !quiet {
		fmt.Fprintf(cd.appManager.CommandsTextView, "[%s]%s[-]\n", tcell.ColorWhite, tview.Escape(command))
	}
	ctx, cancel := context.WithDeadline(ctx, time.Now().Add(timeout))
	defer cancel()
	messageResponse, err := cd.grbl.SendCommand(ctx, command)
	if err != nil {
		fmt.Fprintf(cd.appManager.CommandsTextView, "[%s]Send command failed: %s[-]\n", tcell.ColorRed, tview.Escape(err.Error()))
		return
	}
	if quiet {
		return
	}
	if messageResponse.Error() == nil {
		fmt.Fprintf(cd.appManager.CommandsTextView, "[%s]%s[-]\n", tcell.ColorGreen, tview.Escape(messageResponse.String()))
	} else {
		fmt.Fprintf(cd.appManager.CommandsTextView, "[%s]%s[-]\n", tcell.ColorRed, tview.Escape(messageResponse.String()))
		fmt.Fprintf(cd.appManager.CommandsTextView, "[%s]%s[-]\n", tcell.ColorRed, tview.Escape(messageResponse.Error().Error()))
	}
}

func (cd *CommandDispatcher) RunSendCommandWorker(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			err := ctx.Err()
			if errors.Is(err, context.Canceled) {
				err = nil
			}
			return err
		case command := <-cd.sendCommandCh:
			cd.appManager.CommandInputField.SetDisabled(true)
			cd.appManager.HomingButton.SetDisabled(true)
			cd.appManager.UnlockButton.SetDisabled(true)
			// s.shellApp.joggingButton.SetDisabled(true)
			cd.appManager.CheckButton.SetDisabled(true)
			cd.appManager.SleepButton.SetDisabled(true)
			// s.shellApp.settingsButton.SetDisabled(true)
			cd.sendCommand(ctx, command)
			// Sending $G enables tracking of G-Code parsing state
			cd.sendCommand(ctx, "$G")
			// Sending $G enables tracking of G-Code parameters
			cd.sendCommand(ctx, "$#")
			cd.appManager.CommandInputField.SetText("")
			cd.appManager.CommandInputField.SetDisabled(false)
			cd.appManager.HomingButton.SetDisabled(false)
			cd.appManager.UnlockButton.SetDisabled(false)
			// s.shellApp.joggingButton.SetDisabled(false)
			cd.appManager.CheckButton.SetDisabled(false)
			cd.appManager.SleepButton.SetDisabled(false)
			// s.shellApp.settingsButton.SetDisabled(false)
		}
	}
}

func (cd *CommandDispatcher) QueueCommand(
	command string,
) {
	cd.sendCommandCh <- command
}

func (cd *CommandDispatcher) sendRealTimeCommand(
	cmd grblMod.RealTimeCommand,
) {
	if cd.quietStatusComms || cmd != grblMod.RealTimeCommandStatusReportQuery {
		fmt.Fprintf(cd.appManager.CommandsTextView, "[%s]%s[-]\n", tcell.ColorWhite, tview.Escape(cmd.String()))
	}
	if err := cd.grbl.SendRealTimeCommand(cmd); err != nil {
		fmt.Fprintf(cd.appManager.CommandsTextView, "[%s]Failed to send soft reset: %s[-]\n", tcell.ColorRed, err)
	}
}

func (cd *CommandDispatcher) QueueRealTimeCommand(rtc grblMod.RealTimeCommand) {
	cd.sendRealTimeCommandCh <- rtc
}

func (cd *CommandDispatcher) RunSendRealTimeCommandWorker(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			err := ctx.Err()
			if errors.Is(err, context.Canceled) {
				err = nil
			}
			return err
		case realTimeCommand := <-cd.sendRealTimeCommandCh:
			cd.sendRealTimeCommand(realTimeCommand)
		}
	}
}
