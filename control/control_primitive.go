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
	mu                         sync.Mutex
	disableCommandInput        bool
	machineState               *string
}

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
		}
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
	controlFlex.AddItem(gcodeFlex, 0, 1, false)
	controlFlex.AddItem(commsFlex, 0, 1, false)
	controlFlex.AddItem(commandInputField, 1, 0, true)
	cp.Flex = controlFlex

	return cp
}

func (cp *ControlPrimitive) setDisabledState(ctx context.Context) {
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

func (cp *ControlPrimitive) setMachineState(ctx context.Context, machineState string) {
	cp.mu.Lock()
	cp.machineState = &machineState
	cp.mu.Unlock()
	cp.setDisabledState(ctx)
}

func (cp *ControlPrimitive) DisableCommandInput(ctx context.Context, disabled bool) {
	cp.mu.Lock()
	cp.disableCommandInput = disabled
	cp.mu.Unlock()
	cp.setDisabledState(ctx)
}

//gocyclo:ignore
func (cp *ControlPrimitive) sendCommand(ctx context.Context, command string) {
	textView := cp.commandsTextView

	// Extract and send real-time commands
	var buf bytes.Buffer
	for _, c := range []byte(command) {
		rtc, err := grblMod.NewRealTimeCommand(c)
		if err != nil {
			if !errors.Is(err, grblMod.ErrNotRealTimeCommand) {
				fmt.Fprintf(textView, "\n[%s]Real time command parsing fail: %s[-]", tcell.ColorRed, tview.Escape(err.Error()))
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
	timeout := 1 * time.Second
	parser := gcode.NewParser(strings.NewReader(command))
	for {
		block, err := parser.Next()
		if err != nil {
			fmt.Fprintf(textView, "\n[%s]Failed to parse: %s[-]", tcell.ColorRed, tview.Escape(err.Error()))
			return
		}
		if block == nil {
			break
		}
		if block.IsSystem() {
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
					Message: "(virtual status report: Home)",
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

	// send command
	if !quiet {
		fmt.Fprintf(textView, "\n[%s]%s[-]", tcell.ColorWhite, tview.Escape(command))
	}
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithDeadline(ctx, time.Now().Add(timeout))
		defer cancel()
	}
	messageResponse, err := cp.grbl.SendCommand(ctx, command)
	if err != nil {
		fmt.Fprintf(textView, "\n[%s]Send command failed: %s[-]", tcell.ColorRed, tview.Escape(err.Error()))
		return
	}
	if quiet {
		return
	}
	if messageResponse.Error() == nil {
		fmt.Fprintf(textView, "\n[%s]%s[-]", tcell.ColorGreen, tview.Escape(messageResponse.String()))
	} else {
		fmt.Fprintf(textView, "\n[%s]%s[-]", tcell.ColorRed, tview.Escape(messageResponse.String()))
		fmt.Fprintf(textView, "\n[%s]%s[-]", tcell.ColorRed, tview.Escape(messageResponse.Error().Error()))
	}
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
			cp.DisableCommandInput(ctx, true)
			// s.shellApp.settingsButton.SetDisabled(true)
			cp.sendCommand(ctx, command)
			// Sending $G enables tracking of G-Code parsing state
			cp.sendCommand(ctx, "$G")
			// Sending $G enables tracking of G-Code parameters
			cp.sendCommand(ctx, "$#")
			cp.DisableCommandInput(ctx, false)
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
	ctx context.Context,
	messagePushGcodeState *grblMod.MessagePushGcodeState,
) tcell.Color {
	var buf bytes.Buffer

	if modalGroup := messagePushGcodeState.ModalGroup; modalGroup != nil {
		if modalGroup.Motion != nil {
			fmt.Fprintf(&buf, "%s:%s\n", modalGroup.Motion.NormalizedString(), modalGroup.Motion.Name())
		}
		if modalGroup.PlaneSelection != nil {
			fmt.Fprintf(&buf, "%s:%s\n", modalGroup.PlaneSelection.NormalizedString(), modalGroup.PlaneSelection.Name())
		}
		if modalGroup.DistanceMode != nil {
			fmt.Fprintf(&buf, "%s:%s\n", modalGroup.DistanceMode.NormalizedString(), modalGroup.DistanceMode.Name())
		}
		if modalGroup.FeedRateMode != nil {
			fmt.Fprintf(&buf, "%s:%s\n", modalGroup.FeedRateMode.NormalizedString(), modalGroup.FeedRateMode.Name())
		}
		if modalGroup.Units != nil {
			fmt.Fprintf(&buf, "%s:%s\n", messagePushGcodeState.ModalGroup.Units.NormalizedString(), messagePushGcodeState.ModalGroup.Units.Name())
		}
		if modalGroup.CutterRadiusCompensation != nil {
			fmt.Fprintf(&buf, "%s:%s\n", modalGroup.CutterRadiusCompensation.NormalizedString(), modalGroup.CutterRadiusCompensation.Name())
		}
		if modalGroup.ToolLengthOffset != nil {
			fmt.Fprintf(&buf, "%s:%s\n", modalGroup.ToolLengthOffset.NormalizedString(), modalGroup.ToolLengthOffset.Name())
		}
		if modalGroup.CoordinateSystemSelection != nil {
			fmt.Fprintf(&buf, "%s:%s\n", modalGroup.CoordinateSystemSelection.NormalizedString(), modalGroup.CoordinateSystemSelection.Name())
		}
		if modalGroup.Stopping != nil {
			fmt.Fprintf(&buf, "%s:%s\n", modalGroup.Stopping.NormalizedString(), modalGroup.Stopping.Name())
		}
		if modalGroup.SpindleTurning != nil {
			fmt.Fprintf(&buf, "%s:%s\n", modalGroup.SpindleTurning.NormalizedString(), modalGroup.SpindleTurning.Name())
		}
		for _, word := range modalGroup.Coolant {
			fmt.Fprintf(&buf, "%s:%s\n", word.NormalizedString(), word.Name())
		}
	}

	if messagePushGcodeState.Tool != nil {
		fmt.Fprintf(&buf, "Tool: %.0f\n", *messagePushGcodeState.Tool)
	}
	if messagePushGcodeState.FeedRate != nil {
		fmt.Fprintf(&buf, "Feed Rate: %.0f\n", *messagePushGcodeState.FeedRate)
	}
	if messagePushGcodeState.SpindleSpeed != nil {
		fmt.Fprintf(&buf, "Speed: %.0f\n", *messagePushGcodeState.SpindleSpeed)
	}

	cp.app.QueueUpdate(func() {
		cp.gcodeParserTextView.Clear()
	})

	fmt.Fprint(cp.gcodeParserTextView, tview.Escape(buf.String()))

	return tcell.ColorGreen
}

//gocyclo:ignore
func (cp *ControlPrimitive) processMessagePushGcodeParam(ctx context.Context) tcell.Color {
	color := tcell.ColorGreen

	params := cp.grbl.GetGcodeParameters()
	if params == nil {
		return color
	}

	var buf bytes.Buffer

	if params.CoordinateSystem1 != nil {
		fmt.Fprintf(&buf, "G54:Coordinate System 1\n")
		fmt.Fprintf(&buf, "X:%.4f Y:%.4f Z:%.4f", params.CoordinateSystem1.X, params.CoordinateSystem1.Y, params.CoordinateSystem1.Z)
		if params.CoordinateSystem1.A != nil {
			fmt.Fprintf(&buf, " A:%.4f", *params.CoordinateSystem1.A)
		}
		fmt.Fprintf(&buf, "\n")
	}
	if params.CoordinateSystem2 != nil {
		fmt.Fprintf(&buf, "G55:Coordinate System 2\n")
		fmt.Fprintf(&buf, "X:%.4f Y:%.4f Z:%.4f", params.CoordinateSystem2.X, params.CoordinateSystem2.Y, params.CoordinateSystem2.Z)
		if params.CoordinateSystem2.A != nil {
			fmt.Fprintf(&buf, " A:%.4f", *params.CoordinateSystem2.A)
		}
		fmt.Fprintf(&buf, "\n")
	}
	if params.CoordinateSystem3 != nil {
		fmt.Fprintf(&buf, "G56:Coordinate System 3\n")
		fmt.Fprintf(&buf, "X:%.4f Y:%.4f Z:%.4f", params.CoordinateSystem3.X, params.CoordinateSystem3.Y, params.CoordinateSystem3.Z)
		if params.CoordinateSystem3.A != nil {
			fmt.Fprintf(&buf, " A:%.4f", *params.CoordinateSystem3.A)
		}
		fmt.Fprintf(&buf, "\n")
	}
	if params.CoordinateSystem4 != nil {
		fmt.Fprintf(&buf, "G57:Coordinate System 4\n")
		fmt.Fprintf(&buf, "X:%.4f Y:%.4f Z:%.4f", params.CoordinateSystem4.X, params.CoordinateSystem4.Y, params.CoordinateSystem4.Z)
		if params.CoordinateSystem4.A != nil {
			fmt.Fprintf(&buf, " A:%.4f", *params.CoordinateSystem4.A)
		}
		fmt.Fprintf(&buf, "\n")
	}
	if params.CoordinateSystem5 != nil {
		fmt.Fprintf(&buf, "G58:Coordinate System 5\n")
		fmt.Fprintf(&buf, "X:%.4f Y:%.4f Z:%.4f", params.CoordinateSystem5.X, params.CoordinateSystem5.Y, params.CoordinateSystem5.Z)
		if params.CoordinateSystem5.A != nil {
			fmt.Fprintf(&buf, " A:%.4f", *params.CoordinateSystem5.A)
		}
		fmt.Fprintf(&buf, "\n")
	}
	if params.CoordinateSystem6 != nil {
		fmt.Fprintf(&buf, "G59:Coordinate System 6\n")
		fmt.Fprintf(&buf, "X:%.4f Y:%.4f Z:%.4f", params.CoordinateSystem6.X, params.CoordinateSystem6.Y, params.CoordinateSystem6.Z)
		if params.CoordinateSystem6.A != nil {
			fmt.Fprintf(&buf, " A:%.4f", *params.CoordinateSystem6.A)
		}
		fmt.Fprintf(&buf, "\n")
	}
	if params.PrimaryPreDefinedPosition != nil {
		fmt.Fprintf(&buf, "G28:Primary Pre-Defined Position\n")
		fmt.Fprintf(&buf, "X:%.4f Y:%.4f Z:%.4f", params.PrimaryPreDefinedPosition.X, params.PrimaryPreDefinedPosition.Y, params.PrimaryPreDefinedPosition.Z)
		if params.PrimaryPreDefinedPosition.A != nil {
			fmt.Fprintf(&buf, " A:%.4f", *params.PrimaryPreDefinedPosition.A)
		}
		fmt.Fprintf(&buf, "\n")
	}
	if params.SecondaryPreDefinedPosition != nil {
		fmt.Fprintf(&buf, "G30:Secondary Pre-Defined Position\n")
		fmt.Fprintf(&buf, "X:%.4f Y:%.4f Z:%.4f", params.SecondaryPreDefinedPosition.X, params.SecondaryPreDefinedPosition.Y, params.SecondaryPreDefinedPosition.Z)
		if params.SecondaryPreDefinedPosition.A != nil {
			fmt.Fprintf(&buf, " A:%.4f", *params.SecondaryPreDefinedPosition.A)
		}
		fmt.Fprintf(&buf, "\n")
	}
	if params.CoordinateOffset != nil {
		fmt.Fprintf(&buf, "G92:Coordinate Offset\n")
		fmt.Fprintf(&buf, "X:%.4f Y:%.4f Z:%.4f", params.CoordinateOffset.X, params.CoordinateOffset.Y, params.CoordinateOffset.Z)
		if params.CoordinateOffset.A != nil {
			fmt.Fprintf(&buf, " A:%.4f", *params.CoordinateOffset.A)
		}
		fmt.Fprintf(&buf, "\n")
	}
	if params.ToolLengthOffset != nil {
		fmt.Fprintf(&buf, "Tool Length Offset\n")
		fmt.Fprintf(&buf, "Z:%.4f\n", *params.ToolLengthOffset)
	}
	if params.Probe != nil {
		fmt.Fprintf(&buf, "Last Probing Cycle\n")
		fmt.Fprintf(&buf, "X:%.4f Y:%.4f Z:%.4f", params.Probe.Coordinates.X, params.Probe.Coordinates.Y, params.Probe.Coordinates.Z)
		if params.Probe.Coordinates.A != nil {
			fmt.Fprintf(&buf, " A:%.4f", *params.Probe.Coordinates.A)
		}
		fmt.Fprintf(&buf, "\n")
		fmt.Fprintf(&buf, "Successful: %v\n", params.Probe.Successful)
	}

	cp.app.QueueUpdate(func() {
		cp.gcodeParamsTextView.Clear()
	})

	fmt.Fprint(cp.gcodeParamsTextView, tview.Escape(buf.String()))

	return color
}

func (cp *ControlPrimitive) processMessagePushWelcome(ctx context.Context) {
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
	ctx context.Context,
	statusReport *grblMod.MessagePushStatusReport,
) tcell.Color {
	color := getMachineStateColor(statusReport.MachineState.State)
	if color == tcell.ColorBlack {
		color = tcell.ColorWhite
	}
	cp.setMachineState(ctx, statusReport.MachineState.State)
	return color
}

func (cp *ControlPrimitive) ProcessMessage(ctx context.Context, message grblMod.Message) {
	var color = tcell.ColorGreen
	var extraInfo string

	if messagePushGcodeState, ok := message.(*grblMod.MessagePushGcodeState); ok {
		color = cp.processMessagePushGcodeState(ctx, messagePushGcodeState)
		if cp.quietGcodeParserStateComms {
			return
		}
	}

	if _, ok := message.(*grblMod.MessagePushGcodeParam); ok {
		color = cp.processMessagePushGcodeParam(ctx)
		if cp.quietGcodeParamStateComms {
			return
		}
	}

	if _, ok := message.(*grblMod.MessagePushWelcome); ok {
		cp.processMessagePushWelcome(ctx)
	}

	if messagePushAlarm, ok := message.(*grblMod.MessagePushAlarm); ok {
		extraInfo, color = cp.processMessagePushAlarm(messagePushAlarm)
	}

	if messagePushStatusReport, ok := message.(*grblMod.MessagePushStatusReport); ok {
		color = cp.processMessagePushStatusReport(ctx, messagePushStatusReport)
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
