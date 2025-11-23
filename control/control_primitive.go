package control

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/fornellas/slogxt/log"

	"github.com/fornellas/cgs/gcode"
	grblMod "github.com/fornellas/cgs/grbl"
)

var defaultCommandTimeout = 1 * time.Second

type ControlPrimitive struct {
	*tview.Flex
	grbl                       *grblMod.Grbl
	pushMessageCh              chan grblMod.Message
	app                        *tview.Application
	statusPrimitive            *StatusPrimitive
	quietGcodeParserStateComms bool
	quietGcodeParamStateComms  bool
	quietStatusComms           bool
	sendCommandCh              chan string
	sendRealTimeCommandCh      chan grblMod.RealTimeCommand
	commandsTextView           *tview.TextView
	pushMessagesTextView       *tview.TextView
	gcodeParserTextView        *tview.TextView
	gcodeParamsTextView        *tview.TextView
	commandInputField          *tview.InputField
	commandInputHistory        []string
	commandInputHistoryIdx     int
	mu                         sync.Mutex
	disableCommandInput        bool
	machineState               *string
}

//gocyclo:ignore
func NewControlPrimitive(
	ctx context.Context,
	grbl *grblMod.Grbl,
	pushMessageCh chan grblMod.Message,
	app *tview.Application,
	statusPrimitive *StatusPrimitive,
	quietGcodeParserStateComms bool,
	quietGcodeParamStateComms bool,
	quietStatusComms bool,
) *ControlPrimitive {
	cp := &ControlPrimitive{
		grbl:                       grbl,
		pushMessageCh:              pushMessageCh,
		app:                        app,
		statusPrimitive:            statusPrimitive,
		quietGcodeParserStateComms: quietGcodeParserStateComms,
		quietGcodeParamStateComms:  quietGcodeParamStateComms,
		quietStatusComms:           quietStatusComms,
		sendCommandCh:              make(chan string, 10),
		sendRealTimeCommandCh:      make(chan grblMod.RealTimeCommand, 10),
		commandInputHistoryIdx:     -1,
	}
	ctx, _ = log.MustWithGroup(ctx, "ControlPrimitive")

	// Commands
	commandsTextView := tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true).
		SetWrap(true)
	commandsTextView.SetBorder(true).SetTitle("Commands")
	commandsTextView.SetChangedFunc(func() {
		cp.app.QueueUpdate(func() {
			text := commandsTextView.GetText(false)
			if len(text) > 0 && text[0] == '\n' {
				commandsTextView.SetText(text[1:])
			}
			commandsTextView.ScrollToEnd()
		})
	})
	cp.commandsTextView = commandsTextView

	// Push Messages
	pushMessagesTextView := tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true).
		SetWrap(true)
	pushMessagesTextView.SetBorder(true).SetTitle("Push Messages")
	pushMessagesTextView.SetChangedFunc(func() {
		cp.app.QueueUpdate(func() {
			text := pushMessagesTextView.GetText(false)
			if len(text) > 0 && text[0] == '\n' {
				pushMessagesTextView.SetText(text[1:])
			}
			pushMessagesTextView.ScrollToEnd()
		})
	})
	cp.pushMessagesTextView = pushMessagesTextView

	// G-Code Parser
	gcodeParserTextView := tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true).
		SetWrap(true)
	gcodeParserTextView.SetBorder(true).SetTitle("G-Code Parser")
	gcodeParserTextView.SetChangedFunc(func() {
		cp.app.QueueUpdate(func() {
			text := gcodeParserTextView.GetText(false)
			if len(text) > 0 && text[len(text)-1] == '\n' {
				gcodeParserTextView.SetText(text[:len(text)-1])
			}
		})
	})
	cp.gcodeParserTextView = gcodeParserTextView

	// G-Code Parameters
	gcodeParamsTextView := tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true).
		SetWrap(true)
	gcodeParamsTextView.SetBorder(true).SetTitle("G-Code Parameters")
	gcodeParamsTextView.SetChangedFunc(func() {
		cp.app.QueueUpdate(func() {
			text := gcodeParamsTextView.GetText(false)
			if len(text) > 0 && text[len(text)-1] == '\n' {
				gcodeParamsTextView.SetText(text[:len(text)-1])
			}
		})
	})
	cp.gcodeParamsTextView = gcodeParamsTextView

	// Command
	commandInputField := tview.NewInputField()
	commandInputField.SetLabel("Command: ")
	commandInputField.SetDoneFunc(func(key tcell.Key) {
		_, logger := log.MustWithGroup(ctx, "commandInputField")
		logger.Debug("SetDoneFunc")
		switch key {
		case tcell.KeyEscape:
			commandInputField.SetText("")
		case tcell.KeyEnter:
			command := commandInputField.GetText()
			if command == "" {
				return
			}
			cp.QueueCommand(command)
			cp.commandInputHistory = append([]string{command}, cp.commandInputHistory...)
			cp.commandInputHistoryIdx = -1
			commandInputField.SetText("")
		}
	})
	commandInputField.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyUp:
			if cp.commandInputHistoryIdx == len(cp.commandInputHistory)-1 {
				return nil
			}
			cp.commandInputHistoryIdx++
			cp.commandInputField.SetText(cp.commandInputHistory[cp.commandInputHistoryIdx])
			return nil
		case tcell.KeyDown:
			if cp.commandInputHistoryIdx == -1 {
				return nil
			}
			cp.commandInputHistoryIdx--
			if cp.commandInputHistoryIdx < 0 {
				return nil
			}
			cp.commandInputField.SetText(cp.commandInputHistory[cp.commandInputHistoryIdx])
			return nil
		}
		return event
	})
	cp.commandInputField = commandInputField

	gcodeFlex := tview.NewFlex()
	gcodeFlex.SetDirection(tview.FlexColumn)
	gcodeFlex.AddItem(gcodeParserTextView, 0, 1, false)
	gcodeFlex.AddItem(gcodeParamsTextView, 0, 1, false)

	commsFlex := tview.NewFlex()
	commsFlex.SetDirection(tview.FlexColumn)
	commsFlex.AddItem(commandsTextView, 0, 1, false)
	commsFlex.AddItem(pushMessagesTextView, 0, 1, false)

	// Control
	controlFlex := tview.NewFlex()
	controlFlex.SetBorder(true)
	controlFlex.SetTitle("Contrtol")
	controlFlex.SetDirection(tview.FlexRow)
	controlFlex.AddItem(gcodeFlex, 18, 0, false)
	controlFlex.AddItem(commsFlex, 0, 1, false)
	controlFlex.AddItem(commandInputField, 1, 0, true)
	cp.Flex = controlFlex

	// Sending $G enables tracking of G-Code parsing state
	cp.QueueCommand("$G")
	// Sending $G enables tracking of G-Code parameters
	cp.QueueCommand("$#")

	return cp
}

func (cp *ControlPrimitive) setDisabledState() {
	cp.app.QueueUpdate(func() {
		cp.mu.Lock()
		defer cp.mu.Unlock()
		if cp.disableCommandInput || cp.machineState == nil {
			cp.commandInputField.SetDisabled(true)
			return
		}
		switch *cp.machineState {
		case "Idle":
			cp.commandInputField.SetDisabled(false)
		case "Run":
			cp.commandInputField.SetDisabled(true)
		case "Hold":
			cp.commandInputField.SetDisabled(true)
		case "Jog":
			cp.commandInputField.SetDisabled(true)
		case "Alarm":
			cp.commandInputField.SetDisabled(true)
		case "Door":
			cp.commandInputField.SetDisabled(true)
		case "Check":
			cp.commandInputField.SetDisabled(false)
		case "Home":
			cp.commandInputField.SetDisabled(true)
		case "Sleep":
			cp.commandInputField.SetDisabled(true)
		default:
			panic(fmt.Errorf("unknown state: %s", *cp.machineState))
		}
	})
}

func (cp *ControlPrimitive) setMachineState(machineState string) {
	cp.mu.Lock()
	cp.machineState = &machineState
	cp.mu.Unlock()
	cp.setDisabledState()
}

func (cp *ControlPrimitive) DisableCommandInput(ctx context.Context, disabled bool) {
	cp.mu.Lock()
	cp.disableCommandInput = disabled
	cp.mu.Unlock()
	cp.setDisabledState()
}

func (cp *ControlPrimitive) sendCommand(
	ctx context.Context,
	command string,
	timeout time.Duration,
	quiet, skipGcodeStatusCmds bool,
) {
	cp.DisableCommandInput(ctx, true)
	defer cp.DisableCommandInput(ctx, false)

	commands := []string{command}
	if !skipGcodeStatusCmds {
		// Sending $G enables tracking of G-Code parsing state
		commands = append(commands, "$G")
		// Sending $G enables tracking of G-Code parameters
		commands = append(commands, "$#")
	}

	for i, command := range commands {
		quietCmd := quiet
		if !quietCmd && i > 0 {
			if command == "$G" && cp.quietGcodeParserStateComms {
				quietCmd = true
			}
			if command == "$#" && cp.quietGcodeParamStateComms {
				quietCmd = true
			}
		}

		if !quietCmd {
			fmt.Fprintf(cp.commandsTextView, "\n[%s]%s[-]", tcell.ColorWhite, tview.Escape(command))
		}

		cmdCtx := ctx
		if i == 0 && timeout > 0 {
			var cancel context.CancelFunc
			cmdCtx, cancel = context.WithDeadline(ctx, time.Now().Add(timeout))
			defer cancel()
		}

		messageResponse, err := cp.grbl.SendCommand(cmdCtx, command)
		if err != nil {
			fmt.Fprintf(cp.commandsTextView, "\n[%s]Send command failed: %s[-]", tcell.ColorRed, tview.Escape(err.Error()))
			return
		}

		if quietCmd {
			continue
		}

		if messageResponse.Error() == nil {
			fmt.Fprintf(cp.commandsTextView, "\n[%s]%s[-]", tcell.ColorGreen, tview.Escape(messageResponse.String()))
		} else {
			fmt.Fprintf(cp.commandsTextView, "\n[%s]%s[-]", tcell.ColorRed, tview.Escape(messageResponse.String()))
			fmt.Fprintf(cp.commandsTextView, "\n[%s]%s[-]", tcell.ColorRed, tview.Escape(messageResponse.Error().Error()))
		}
	}
}

func (cp *ControlPrimitive) processCommand(ctx context.Context, command string) {
	// Extract and send real-time commands
	var buf bytes.Buffer
	for _, c := range []byte(command) {
		rtc, err := grblMod.NewRealTimeCommand(c)
		if err != nil {
			if !errors.Is(err, grblMod.ErrNotRealTimeCommand) {
				fmt.Fprintf(cp.commandsTextView, "\n[%s]Real time command parsing fail: %s[-]", tcell.ColorRed, tview.Escape(err.Error()))
				return
			}
			buf.WriteByte(c)
		} else {
			cp.sendRealTimeCommand(rtc)
		}
	}
	command = buf.String()

	if len(command) == 0 {
		return
	}

	// Verbosity & timeout
	var quiet bool
	var skipGcodeStatusCmds bool
	timeout := defaultCommandTimeout
	parser := gcode.NewParser(strings.NewReader(command))
	for {
		block, err := parser.Next()
		if err != nil {
			fmt.Fprintf(cp.commandsTextView, "\n[%s]Failed to parse: %s[-]", tcell.ColorRed, tview.Escape(err.Error()))
			return
		}
		if block == nil {
			break
		}
		if block.IsSystem() {
			skipGcodeStatusCmds = true
			switch block.String() {
			case "$G":
				if cp.quietGcodeParserStateComms {
					quiet = true
				}
			case "$#":
				if cp.quietGcodeParamStateComms {
					quiet = true
				}
			case "$H":
				timeout = 120 * time.Second
				// Grbl stops responding to status report queries while homing. Generating this
				// virtual status report enables subscribers to process the otherwise unreported
				//  state.
				cp.pushMessageCh <- &grblMod.MessagePushStatusReport{
					Message: "(virtual push message: status report: Home)",
					MachineState: grblMod.StatusReportMachineState{
						State: "Home",
					},
				}
			}
		} else {
			switch block.String() {
			case "M0":
				timeout = 0
			}
		}
	}

	// Send command
	cp.sendCommand(ctx, command, timeout, quiet, skipGcodeStatusCmds)
}

func (cp *ControlPrimitive) RunSendCommandWorker(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			err := ctx.Err()
			if errors.Is(err, context.Canceled) {
				err = nil
			}
			return err
		case command := <-cp.sendCommandCh:
			cp.processCommand(ctx, command)
		}
	}
}

func (cp *ControlPrimitive) QueueCommand(
	command string,
) {
	cp.sendCommandCh <- command
}

func (cp *ControlPrimitive) sendRealTimeCommand(
	cmd grblMod.RealTimeCommand,
) {
	textView := cp.commandsTextView

	if cp.quietStatusComms || cmd != grblMod.RealTimeCommandStatusReportQuery {
		fmt.Fprintf(textView, "\n[%s]%s[-]", tcell.ColorBlue, tview.Escape(cmd.String()))
	}
	if err := cp.grbl.SendRealTimeCommand(cmd); err != nil {
		fmt.Fprintf(textView, "\n[%s]Failed to send real-time command: %s[-]", tcell.ColorRed, err)
	}
}

func (cp *ControlPrimitive) QueueRealTimeCommand(rtc grblMod.RealTimeCommand) {
	cp.sendRealTimeCommandCh <- rtc
}

func (cp *ControlPrimitive) RunSendRealTimeCommandWorker(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			err := ctx.Err()
			if errors.Is(err, context.Canceled) {
				err = nil
			}
			return err
		case realTimeCommand := <-cp.sendRealTimeCommandCh:
			cp.sendRealTimeCommand(realTimeCommand)
		}
	}
}

//gocyclo:ignore
func (cp *ControlPrimitive) processMessagePushGcodeState(
	messagePushGcodeState *grblMod.MessagePushGcodeState,
) tcell.Color {
	var buf bytes.Buffer

	if modalGroup := messagePushGcodeState.ModalGroup; modalGroup != nil {
		if modalGroup.Motion != nil {
			fmt.Fprintf(
				&buf, "%s:%s\n",
				sprintGcodeWord(modalGroup.Motion.NormalizedString()), modalGroup.Motion.Name(),
			)
		}
		if modalGroup.PlaneSelection != nil {
			fmt.Fprintf(
				&buf, "%s:%s\n",
				sprintGcodeWord(modalGroup.PlaneSelection.NormalizedString()), modalGroup.PlaneSelection.Name(),
			)
		}
		if modalGroup.DistanceMode != nil {
			fmt.Fprintf(&buf, "%s:%s\n", sprintGcodeWord(modalGroup.DistanceMode.NormalizedString()), modalGroup.DistanceMode.Name())
		}
		if modalGroup.FeedRateMode != nil {
			fmt.Fprintf(&buf, "%s:%s\n", sprintGcodeWord(modalGroup.FeedRateMode.NormalizedString()), modalGroup.FeedRateMode.Name())
		}
		if modalGroup.Units != nil {
			fmt.Fprintf(&buf, "%s:%s\n", sprintGcodeWord(messagePushGcodeState.ModalGroup.Units.NormalizedString()), messagePushGcodeState.ModalGroup.Units.Name())
		}
		if modalGroup.CutterRadiusCompensation != nil {
			fmt.Fprintf(&buf, "%s:%s\n", sprintGcodeWord(modalGroup.CutterRadiusCompensation.NormalizedString()), modalGroup.CutterRadiusCompensation.Name())
		}
		if modalGroup.ToolLengthOffset != nil {
			fmt.Fprintf(&buf, "%s:%s\n", sprintGcodeWord(modalGroup.ToolLengthOffset.NormalizedString()), modalGroup.ToolLengthOffset.Name())
		}
		if modalGroup.CoordinateSystemSelection != nil {
			fmt.Fprintf(&buf, "%s:%s\n", sprintGcodeWord(modalGroup.CoordinateSystemSelection.NormalizedString()), modalGroup.CoordinateSystemSelection.Name())
		}
		if modalGroup.Stopping != nil {
			fmt.Fprintf(&buf, "%s:%s\n", sprintGcodeWord(modalGroup.Stopping.NormalizedString()), modalGroup.Stopping.Name())
		}
		if modalGroup.SpindleTurning != nil {
			fmt.Fprintf(&buf, "%s:%s\n", sprintGcodeWord(modalGroup.SpindleTurning.NormalizedString()), modalGroup.SpindleTurning.Name())
		}
		for _, word := range modalGroup.Coolant {
			fmt.Fprintf(&buf, "%s:%s\n", sprintGcodeWord(word.NormalizedString()), word.Name())
		}
	}

	if messagePushGcodeState.Tool != nil {
		fmt.Fprintf(&buf, "Tool: %s\n", sprintTool(*messagePushGcodeState.Tool))
	}
	if messagePushGcodeState.FeedRate != nil {
		fmt.Fprintf(&buf, "Feed Rate: %s\n", sprintFeed(*messagePushGcodeState.FeedRate))
	}
	if messagePushGcodeState.SpindleSpeed != nil {
		fmt.Fprintf(&buf, "Speed: %s\n", sprintSpeed(*messagePushGcodeState.SpindleSpeed))
	}

	cp.app.QueueUpdate(func() {
		if buf.String() == cp.gcodeParserTextView.GetText(false) {
			return
		}
		cp.gcodeParserTextView.SetText(buf.String())
	})

	return tcell.ColorGreen
}

//gocyclo:ignore
func (cp *ControlPrimitive) processMessagePushGcodeParam() tcell.Color {
	color := tcell.ColorGreen

	params := cp.grbl.GetGcodeParameters()
	if params == nil {
		return color
	}

	var buf bytes.Buffer

	if params.HasCoordinateSystem() {
		fmt.Fprint(&buf, "Coordinate System\n")
		if params.CoordinateSystem1 != nil {
			fmt.Fprintf(&buf, "  %s:", sprintGcodeWord("G54"))
			fmt.Fprintf(&buf, "%s\n", sprintCoordinatesSingleLine(params.CoordinateSystem1, " "))
		}
		if params.CoordinateSystem2 != nil {
			fmt.Fprintf(&buf, "  %s:", sprintGcodeWord("G55"))
			fmt.Fprintf(&buf, "%s\n", sprintCoordinatesSingleLine(params.CoordinateSystem2, " "))
		}
		if params.CoordinateSystem3 != nil {
			fmt.Fprintf(&buf, "  %s:", sprintGcodeWord("G56"))
			fmt.Fprintf(&buf, "%s\n", sprintCoordinatesSingleLine(params.CoordinateSystem3, " "))
		}
		if params.CoordinateSystem4 != nil {
			fmt.Fprintf(&buf, "  %s:", sprintGcodeWord("G57"))
			fmt.Fprintf(&buf, "%s\n", sprintCoordinatesSingleLine(params.CoordinateSystem4, " "))
		}
		if params.CoordinateSystem5 != nil {
			fmt.Fprintf(&buf, "  %s:", sprintGcodeWord("G58"))
			fmt.Fprintf(&buf, "%s\n", sprintCoordinatesSingleLine(params.CoordinateSystem5, " "))
		}
		if params.CoordinateSystem6 != nil {
			fmt.Fprintf(&buf, "  %s:", sprintGcodeWord("G59"))
			fmt.Fprintf(&buf, "%s\n", sprintCoordinatesSingleLine(params.CoordinateSystem6, " "))
		}
	}
	if params.HasPreDefinedPosition() {
		fmt.Fprintf(&buf, "Pre-Defined Position\n")
		if params.PrimaryPreDefinedPosition != nil {
			fmt.Fprintf(&buf, "  %s:%s\n", sprintGcodeWord("G28"), sprintCoordinatesSingleLine(params.PrimaryPreDefinedPosition, " "))
		}
		if params.SecondaryPreDefinedPosition != nil {
			fmt.Fprintf(&buf, "  %s:%s\n", sprintGcodeWord("G30"), sprintCoordinatesSingleLine(params.SecondaryPreDefinedPosition, " "))
		}
	}
	if params.CoordinateOffset != nil {
		fmt.Fprintf(&buf, "Coordinate Offset\n")
		fmt.Fprintf(&buf, "  %s:%s\n", sprintGcodeWord("G92"), sprintCoordinatesSingleLine(params.CoordinateOffset, " "))
	}
	if params.ToolLengthOffset != nil {
		fmt.Fprintf(&buf, "Tool Length Offset:%s\n", sprintCoordinate(*params.ToolLengthOffset))
	}
	if params.Probe != nil {
		fmt.Fprintf(&buf, "Last Probing Cycle\n")
		fmt.Fprintf(&buf, "  %s\n", sprintCoordinatesSingleLine(&params.Probe.Coordinates, " "))
		fmt.Fprintf(&buf, "  Successful: %s\n", sprintBool(params.Probe.Successful))
	}

	cp.app.QueueUpdate(func() {
		if buf.String() == cp.gcodeParamsTextView.GetText(false) {
			return
		}
		cp.gcodeParamsTextView.SetText(buf.String())
	})

	return color
}

func (cp *ControlPrimitive) processMessagePushWelcome() {
	cp.app.QueueUpdate(func() {
		cp.gcodeParserTextView.Clear()
		cp.gcodeParamsTextView.Clear()
	})
	fmt.Fprintf(cp.pushMessagesTextView, "\n[%s]Soft-Reset detected[-]", tcell.ColorOrange)
	// Sending $G enables tracking of G-Code parsing state
	cp.QueueCommand("$G")
	// Sending $G enables tracking of G-Code parameters
	cp.QueueCommand("$#")
}

func (cp *ControlPrimitive) processMessagePushAlarm(
	messagePushAlarm *grblMod.MessagePushAlarm,
) (string, tcell.Color) {
	return tview.Escape(messagePushAlarm.Error().Error()), tcell.ColorRed
}

func (cp *ControlPrimitive) processMessagePushStatusReport(
	statusReport *grblMod.MessagePushStatusReport,
) tcell.Color {
	color := getMachineStateColor(statusReport.MachineState.State)
	if color == tcell.ColorBlack {
		color = tcell.ColorWhite
	}
	cp.setMachineState(statusReport.MachineState.State)
	return color
}

func (cp *ControlPrimitive) ProcessMessage(ctx context.Context, message grblMod.Message) {
	var color = tcell.ColorGreen
	var extraInfo string

	if messagePushGcodeState, ok := message.(*grblMod.MessagePushGcodeState); ok {
		color = cp.processMessagePushGcodeState(messagePushGcodeState)
		if cp.quietGcodeParserStateComms {
			return
		}
	}

	if _, ok := message.(*grblMod.MessagePushGcodeParam); ok {
		color = cp.processMessagePushGcodeParam()
		if cp.quietGcodeParamStateComms {
			return
		}
	}

	if _, ok := message.(*grblMod.MessagePushWelcome); ok {
		cp.processMessagePushWelcome()
	}

	if messagePushAlarm, ok := message.(*grblMod.MessagePushAlarm); ok {
		extraInfo, color = cp.processMessagePushAlarm(messagePushAlarm)
	}

	if messagePushStatusReport, ok := message.(*grblMod.MessagePushStatusReport); ok {
		color = cp.processMessagePushStatusReport(messagePushStatusReport)
		if cp.quietStatusComms {
			return
		}
	}

	text := message.String()
	if len(text) == 0 {
		fmt.Fprintf(cp.pushMessagesTextView, "\n[%s](%#v)[-]", color, tview.Escape(reflect.TypeOf(message).String()))
	} else {
		fmt.Fprintf(cp.pushMessagesTextView, "\n[%s]%s[-]", color, tview.Escape(text))
	}
	if len(extraInfo) > 0 {
		fmt.Fprintf(cp.pushMessagesTextView, "\n[%s]%s[-]", tcell.ColorWhite, tview.Escape(extraInfo))
	}
}
