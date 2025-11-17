package shell

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"time"

	"github.com/fornellas/slogxt/log"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/fornellas/cgs/gcode"
	grblMod "github.com/fornellas/cgs/grbl"
)

type Options struct {
	DisplayStatusComms           bool
	DisplayGcodeParserStateComms bool
	DisplayGcodeParamStateComms  bool
}

type Shell struct {
	grbl    *grblMod.Grbl
	options *Options
}

func NewShell(grbl *grblMod.Grbl, options *Options) *Shell {
	if options == nil {
		options = &Options{}
	}
	return &Shell{
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

func (s *Shell) getCommandsTextView(app *tview.Application) *tview.TextView {
	textView := tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true).
		SetWrap(true)
	textView.SetBorder(true).SetTitle("Commands")
	textView.SetChangedFunc(func() {
		textView.ScrollToEnd()
		app.Draw()
	})
	textView.SetBackgroundColor(tcell.ColorDefault)
	return textView
}

func (s *Shell) getPushMessagesLogsTextView(app *tview.Application) *tview.TextView {
	textView := tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true).
		SetWrap(true)
	textView.SetBorder(true).SetTitle("Push Messages / Logs")
	textView.SetChangedFunc(func() {
		textView.ScrollToEnd()
		app.Draw()
	})
	textView.SetBackgroundColor(tcell.ColorDefault)
	return textView
}

func (s *Shell) getFeedbackTextView(app *tview.Application) *tview.TextView {
	textView := tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true).
		SetWrap(true)
	textView.SetTitle("Feedback Message")
	textView.SetChangedFunc(func() {
		textView.ScrollToEnd()
		app.Draw()
	})
	textView.SetBackgroundColor(tcell.ColorDefault)
	return textView
}

func (s *Shell) getGcodeParserTextView(app *tview.Application) *tview.TextView {
	textView := tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true).
		SetWrap(true)
	textView.SetBorder(true).SetTitle("G-Code Parser")
	textView.SetChangedFunc(func() {
		textView.ScrollToBeginning()
		app.Draw()
	})
	textView.SetBackgroundColor(tcell.ColorDefault)
	return textView
}

func (s *Shell) getGcodeParamsTextView(app *tview.Application) *tview.TextView {
	textView := tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true).
		SetWrap(true)
	textView.SetBorder(true).SetTitle("G-Code Parameters")
	textView.SetChangedFunc(func() {
		textView.ScrollToBeginning()
		app.Draw()
	})
	textView.SetBackgroundColor(tcell.ColorDefault)
	return textView
}

func (s *Shell) getStateTextView(app *tview.Application) *tview.TextView {
	textView := tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true).
		SetWrap(true)
	textView.SetBorder(true).SetTitle("State")
	textView.SetChangedFunc(func() {
		app.Draw()
	})
	textView.SetBackgroundColor(tcell.ColorDefault)
	return textView
}

func (s *Shell) getStatusTextView(app *tview.Application) *tview.TextView {
	textView := tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true).
		SetWrap(true)
	textView.SetBorder(true).SetTitle("Status")
	textView.SetChangedFunc(func() {
		textView.ScrollToBeginning()
		app.Draw()
	})
	textView.SetBackgroundColor(tcell.ColorDefault)
	return textView
}

func (s *Shell) getCommandInputField(commandCh chan string) *tview.InputField {
	inputField := tview.NewInputField().
		SetLabel("Command: ")
	inputField.SetDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEscape:
			inputField.SetText("")
		case tcell.KeyEnter:
			command := inputField.GetText()
			if command == "" {
				return
			}
			commandCh <- command
		}
	})
	inputField.SetBackgroundColor(tcell.ColorDefault)
	return inputField
}

func (s *Shell) getApp(
	sendCommandCh chan string,
	sendRealTimeCommandCh chan grblMod.RealTimeCommand,
) (
	*tview.Application,
	*tview.TextView,
	*tview.TextView,
	*tview.TextView,
	*tview.TextView,
	*tview.TextView,
	*tview.TextView,
	*tview.TextView,
	*tview.InputField,
) {
	app := tview.NewApplication()
	app.EnableMouse(true)
	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyCtrlX {
			sendRealTimeCommandCh <- grblMod.RealTimeCommandSoftReset
			return nil
		}
		return event
	})
	commandsTextView := s.getCommandsTextView(app)
	pushMessagesLogsTextView := s.getPushMessagesLogsTextView(app)
	feedbackTextView := s.getFeedbackTextView(app)
	gcodeParserTextView := s.getGcodeParserTextView(app)
	gcodeParamsTextView := s.getGcodeParamsTextView(app)
	stateTextView := s.getStateTextView(app)
	statusTextView := s.getStatusTextView(app)
	commandInputField := s.getCommandInputField(sendCommandCh)
	homingButton := tview.NewButton("Homing").
		SetSelectedFunc(func() { sendCommandCh <- "$H" })
	unlockButton := tview.NewButton("Unlock").
		SetSelectedFunc(func() { sendCommandCh <- "$X" })
	resetButton := tview.NewButton("Reset").
		SetSelectedFunc(func() { sendRealTimeCommandCh <- grblMod.RealTimeCommandSoftReset })
	// joggingButton := tview.NewButton("Jogging").
	// 	SetSelectedFunc(func() { sendCommandCh <- "TODO" })
	// overridesButton := tview.NewButton("Overrides").
	// 	SetSelectedFunc(func() { sendCommandCh <- "TODO" })
	checkButton := tview.NewButton("Check").
		SetSelectedFunc(func() { sendCommandCh <- "$C" })
	doorButton := tview.NewButton("Door").
		SetSelectedFunc(func() { sendRealTimeCommandCh <- grblMod.RealTimeCommandSafetyDoor })
	sleepButton := tview.NewButton("Sleep").
		SetSelectedFunc(func() { sendCommandCh <- "$SLP" })
	holdButton := tview.NewButton("Hold").
		SetSelectedFunc(func() { sendCommandCh <- "!" })
	resumeButton := tview.NewButton("Resume").
		SetSelectedFunc(func() { sendCommandCh <- "~" })
	// settingsButton := tview.NewButton("Settings").
	// 	SetSelectedFunc(func() { sendCommandCh <- "TODO" })
	spindleButton := tview.NewButton("Spindle").
		SetSelectedFunc(func() { sendRealTimeCommandCh <- grblMod.RealTimeCommandToggleSpindleStop })
	coolantButton := tview.NewButton("Coolant").
		SetSelectedFunc(func() { sendRealTimeCommandCh <- grblMod.RealTimeCommandToggleMistCoolant })
	rootFlex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(
			tview.NewFlex().SetDirection(tview.FlexRow).
				AddItem(
					tview.NewFlex().
						AddItem(
							tview.NewFlex().SetDirection(tview.FlexRow).
								AddItem(pushMessagesLogsTextView, 0, 1, false).
								AddItem(commandsTextView, 0, 1, false),
							0, 3, false,
						).
						AddItem(
							tview.NewFlex().SetDirection(tview.FlexRow).
								AddItem(gcodeParserTextView, 0, 1, false).
								AddItem(gcodeParamsTextView, 0, 1, false),
							0, 2, false,
						).
						AddItem(
							tview.NewFlex().SetDirection(tview.FlexRow).
								AddItem(stateTextView, 4, 0, false).
								AddItem(statusTextView, 0, 1, false),
							14, 0, false,
						),
					0, 1, false,
				).
				AddItem(feedbackTextView, 1, 0, false).
				AddItem(commandInputField, 1, 0, false),
			0, 1, false,
		).
		AddItem(
			tview.NewFlex().
				AddItem(homingButton, 0, 1, false).
				AddItem(unlockButton, 0, 1, false).
				AddItem(resetButton, 0, 1, false).
				// AddItem(joggingButton, 0, 1, false).
				// AddItem(overridesButton, 0, 1, false).
				AddItem(checkButton, 0, 1, false),
			1, 0, false,
		).
		AddItem(
			tview.NewFlex().
				AddItem(doorButton, 0, 1, false).
				AddItem(sleepButton, 0, 1, false).
				AddItem(holdButton, 0, 1, false).
				AddItem(resumeButton, 0, 1, false).
				// AddItem(settingsButton, 0, 1, false).
				AddItem(spindleButton, 0, 1, false).
				AddItem(coolantButton, 0, 1, false),
			1, 0, false,
		)
	app.SetRoot(rootFlex, true).SetFocus(commandInputField)
	return app,
		commandsTextView,
		pushMessagesLogsTextView,
		feedbackTextView,
		gcodeParserTextView,
		gcodeParamsTextView,
		stateTextView,
		statusTextView,
		commandInputField
}

//gocyclo:ignore
func (s *Shell) sendCommand(
	ctx context.Context,
	commandsTextView *tview.TextView,
	command string,
) {
	// Extract and send real-time commands
	var buf bytes.Buffer
	for _, c := range []byte(command) {
		rtc, err := grblMod.NewRealTimeCommand(c)
		if err != nil {
			if !errors.Is(err, grblMod.ErrNotRealTimeCommand) {
				fmt.Fprintf(commandsTextView, "[%s]Real time command parsing fail: %s[-]\n", tcell.ColorRed, tview.Escape(err.Error()))
				return
			}
			buf.WriteByte(c)
		} else {
			s.sendRealTimeCommand(commandsTextView, rtc)
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
			fmt.Fprintf(commandsTextView, "[%s]Failed to parse: %s[-]\n", tcell.ColorRed, tview.Escape(err.Error()))
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
		fmt.Fprintf(commandsTextView, "[%s]%s[-]\n", tcell.ColorWhite, tview.Escape(command))
	}
	ctx, cancel := context.WithDeadline(ctx, time.Now().Add(timeout))
	defer cancel()
	messageResponse, err := s.grbl.SendCommand(ctx, command)
	if err != nil {
		fmt.Fprintf(commandsTextView, "[%s]Send command failed: %s[-]\n", tcell.ColorRed, tview.Escape(err.Error()))
		return
	}
	if quiet {
		return
	}
	if messageResponse.Error() == nil {
		fmt.Fprintf(commandsTextView, "[%s]%s[-]\n", tcell.ColorGreen, tview.Escape(messageResponse.String()))
	} else {
		fmt.Fprintf(commandsTextView, "[%s]%s[-]\n", tcell.ColorRed, tview.Escape(messageResponse.String()))
		fmt.Fprintf(commandsTextView, "[%s]%s[-]\n", tcell.ColorRed, tview.Escape(messageResponse.Error().Error()))
	}
}

func (s *Shell) sendCommandWorker(
	ctx context.Context,
	commandsTextView *tview.TextView,
	commandInputField *tview.InputField,
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
			commandInputField.SetDisabled(true)
			s.sendCommand(ctx, commandsTextView, command)
			// Sending $G enables tracking of G-Code parsing state
			s.sendCommand(ctx, commandsTextView, "$G")
			// Sending $G enables tracking of G-Code parameters
			s.sendCommand(ctx, commandsTextView, "$#")
			commandInputField.SetText("")
			commandInputField.SetDisabled(false)
		}
	}
}

func (s *Shell) sendRealTimeCommand(
	commandsTextView *tview.TextView,
	cmd grblMod.RealTimeCommand,
) {
	if s.options.DisplayStatusComms || cmd != grblMod.RealTimeCommandStatusReportQuery {
		fmt.Fprintf(commandsTextView, "[%s]%s[-]\n", tcell.ColorWhite, tview.Escape(cmd.String()))
	}
	if err := s.grbl.SendRealTimeCommand(cmd); err != nil {
		fmt.Fprintf(commandsTextView, "[%s]Failed to send soft reset: %s[-]\n", tcell.ColorRed, err)
	}
}

func (s *Shell) sendRealTimeCommandWorker(
	ctx context.Context,
	commandsTextView *tview.TextView,
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
			s.sendRealTimeCommand(commandsTextView, realTimeCommand)
		}
	}
}

//gocyclo:ignore
func (s *Shell) writePositionStatus(w io.Writer, statusReport *grblMod.MessagePushStatusReport) {
	var mx, my, mz, ma, wx, wy, wz, wa *float64
	if statusReport.MachinePosition != nil {
		mx = &statusReport.MachinePosition.X
		my = &statusReport.MachinePosition.Y
		mz = &statusReport.MachinePosition.Z
		ma = statusReport.MachinePosition.A
		if s.grbl.GetWorkCoordinateOffset() != nil {
			wxv := statusReport.MachinePosition.X - s.grbl.GetWorkCoordinateOffset().X
			wx = &wxv
			wyv := statusReport.MachinePosition.Y - s.grbl.GetWorkCoordinateOffset().Y
			wy = &wyv
			wzv := statusReport.MachinePosition.Z - s.grbl.GetWorkCoordinateOffset().Z
			wz = &wzv
			if statusReport.MachinePosition.A != nil && s.grbl.GetWorkCoordinateOffset().A != nil {
				wav := *statusReport.MachinePosition.A - *s.grbl.GetWorkCoordinateOffset().A
				wa = &wav
			}
		}
	}
	if statusReport.WorkPosition != nil {
		wx = &statusReport.WorkPosition.X
		wy = &statusReport.WorkPosition.Y
		wz = &statusReport.WorkPosition.Z
		wa = statusReport.WorkPosition.A
		if s.grbl.GetWorkCoordinateOffset() != nil {
			mxv := statusReport.WorkPosition.X - s.grbl.GetWorkCoordinateOffset().X
			mx = &mxv
			myv := statusReport.WorkPosition.Y - s.grbl.GetWorkCoordinateOffset().Y
			my = &myv
			mzv := statusReport.WorkPosition.Z - s.grbl.GetWorkCoordinateOffset().Z
			mz = &mzv
			if statusReport.WorkPosition.A != nil && s.grbl.GetWorkCoordinateOffset().A != nil {
				mav := *statusReport.WorkPosition.A - *s.grbl.GetWorkCoordinateOffset().A
				ma = &mav
			}
		}
	}
	var nl bool
	if wx != nil || wy != nil || wz != nil || wa != nil {
		fmt.Fprintf(w, "Work\n")
		nl = true
	}
	if wx != nil {
		fmt.Fprintf(w, "X:%.3f\n", *wx)
	}
	if wy != nil {
		fmt.Fprintf(w, "Y:%.3f\n", *wy)
	}
	if wz != nil {
		fmt.Fprintf(w, "Z:%.3f\n", *wz)
	}
	if wa != nil {
		fmt.Fprintf(w, "A:%.3f\n", *wa)
	}
	if mx != nil || my != nil || mz != nil || ma != nil {
		if nl {
			fmt.Fprintf(w, "\n")
		}
		fmt.Fprintf(w, "Machine\n")
	}
	if mx != nil {
		fmt.Fprintf(w, "X:%.3f\n", *mx)
	}
	if my != nil {
		fmt.Fprintf(w, "Y:%.3f\n", *my)
	}
	if mz != nil {
		fmt.Fprintf(w, "Z:%.3f\n", *mz)
	}
	if ma != nil {
		fmt.Fprintf(w, "A:%.3f\n", *ma)
	}
}

func (s *Shell) updateState(
	stateView *tview.TextView,
	state string,
	subState string,
) {
	stateColor := getMachineStateColor(state)

	stateView.Clear()
	stateView.SetBackgroundColor(stateColor)
	_, _, stateViewWidth, _ := stateView.GetRect()
	fmt.Fprintf(
		stateView, "%s%s\n",
		strings.Repeat(" ", (stateViewWidth-2-len(state))/2),
		tview.Escape(state),
	)
	if len(subState) > 0 {
		fmt.Fprintf(
			stateView, "%s(%s)\n",
			strings.Repeat(" ", (stateViewWidth-4-len(subState))/2),
			tview.Escape(subState),
		)
	}
}

//gocyclo:ignore
func (s *Shell) updateStatusReport(
	stateView *tview.TextView,
	statusView *tview.TextView,
	statusReport *grblMod.MessagePushStatusReport,
) {
	s.updateState(stateView, statusReport.MachineState.State, statusReport.MachineState.SubStateString())

	statusView.Clear()

	s.writePositionStatus(statusView, statusReport)

	if statusReport.BufferState != nil {
		fmt.Fprint(statusView, "\nBuffer\n")
		fmt.Fprintf(statusView, "Blocks:%d\n", statusReport.BufferState.AvailableBlocks)
		fmt.Fprintf(statusView, "Bytes:%d\n", statusReport.BufferState.AvailableBytes)
	}

	if statusReport.LineNumber != nil {
		fmt.Fprintf(statusView, "\nLine:%d\n", *statusReport.LineNumber)
	}

	if statusReport.Feed != nil {
		fmt.Fprintf(statusView, "\nFeed:%.1f\n", *statusReport.Feed)
	}

	if statusReport.FeedSpindle != nil {
		fmt.Fprintf(statusView, "\nFeed:%.0f\n", statusReport.FeedSpindle.Feed)
		fmt.Fprintf(statusView, "Speed:%.0f\n", statusReport.FeedSpindle.Speed)
	}

	if statusReport.PinState != nil {
		fmt.Fprintf(statusView, "\nPin:%s\n", statusReport.PinState)
	}

	if s.grbl.GetOverrideValues() != nil {
		fmt.Fprint(statusView, "\nOverrides\n")
		fmt.Fprintf(statusView, "Feed:%.0f%%\n", s.grbl.GetOverrideValues().Feed)
		fmt.Fprintf(statusView, "Rapids:%.0f%%\n", s.grbl.GetOverrideValues().Rapids)
		fmt.Fprintf(statusView, "Spindle:%.0f%%\n", s.grbl.GetOverrideValues().Spindle)
	}

	if statusReport.AccessoryState != nil {
		fmt.Fprint(statusView, "\nAccessory\n")
		if statusReport.AccessoryState.SpindleCW != nil && *statusReport.AccessoryState.SpindleCW {
			fmt.Fprint(statusView, "Spindle: CW")
		}
		if statusReport.AccessoryState.SpindleCCW != nil && *statusReport.AccessoryState.SpindleCCW {
			fmt.Fprint(statusView, "Spindle: CCW")
		}
		if statusReport.AccessoryState.FloodCoolant != nil && *statusReport.AccessoryState.FloodCoolant {
			fmt.Fprint(statusView, "Flood Coolant")
		}
		if statusReport.AccessoryState.MistCoolant != nil && *statusReport.AccessoryState.MistCoolant {
			fmt.Fprint(statusView, "Mist Coolant")
		}
	}
}

//gocyclo:ignore
func (s *Shell) processMessagePushGcodeState(
	messagePushGcodeState *grblMod.MessagePushGcodeState,
	gcodeParserTextView *tview.TextView,
) (func(), tcell.Color) {
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

	gcodeParserTextView.Clear()
	fmt.Fprint(gcodeParserTextView, tview.Escape(buf.String()))

	return nil, tcell.ColorGreen
}

//gocyclo:ignore
func (s *Shell) processMessagePushGcodeParam(
	gcodeParamsTextView *tview.TextView,
) (func(), tcell.Color) {
	color := tcell.ColorGreen

	params := s.grbl.GetGcodeParameters()
	if params == nil {
		return nil, color
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

	gcodeParamsTextView.Clear()
	fmt.Fprint(gcodeParamsTextView, tview.Escape(buf.String()))

	return nil, color
}

func (s *Shell) processMessagePushWelcome(
	ctx context.Context,
	_ *grblMod.MessagePushWelcome,
	pushMessagesLogsTextView *tview.TextView,
	commandsTextView *tview.TextView,
	gcodeParserTextView *tview.TextView,
	gcodeParamsTextView *tview.TextView,
	stateTextView *tview.TextView,
	statusTextView *tview.TextView,
	feedbackTextView *tview.TextView,
) (func(), tcell.Color) {
	color := tcell.ColorYellow
	detailsFn := func() {
		fmt.Fprintf(pushMessagesLogsTextView, "[%s]Soft-Reset detected[-]\n", color)
	}
	gcodeParserTextView.Clear()
	gcodeParamsTextView.Clear()
	stateTextView.SetBackgroundColor(tview.Styles.PrimitiveBackgroundColor)
	stateTextView.Clear()
	statusTextView.Clear()
	feedbackTextView.SetText("")
	// Sending $G enables tracking of G-Code parsing state
	s.sendCommand(ctx, commandsTextView, "$G")
	// Sending $G enables tracking of G-Code parameters
	s.sendCommand(ctx, commandsTextView, "$#")
	return detailsFn, color
}

func (s *Shell) processMessagePushAlarm(
	messagePushAlarm *grblMod.MessagePushAlarm,
	pushMessagesLogsTextView *tview.TextView,
	stateTextView *tview.TextView,
) (func(), tcell.Color) {
	color := tcell.ColorRed
	detailsFn := func() {
		fmt.Fprintf(pushMessagesLogsTextView, "[%s]%s[-]\n", color, tview.Escape(messagePushAlarm.Error().Error()))
	}
	s.updateState(stateTextView, "Alarm", "")
	return detailsFn, color
}

func (s *Shell) processMessagePushStatusReport(
	messagePushStatusReport *grblMod.MessagePushStatusReport,
	stateTextView *tview.TextView,
	statusTextView *tview.TextView,
) (func(), tcell.Color) {
	color := getMachineStateColor(messagePushStatusReport.MachineState.State)
	s.updateStatusReport(stateTextView, statusTextView, messagePushStatusReport)
	return nil, color
}

func (s *Shell) processMessagePushFeedback(
	messagePushFeedback *grblMod.MessagePushFeedback,
	feedbackTextView *tview.TextView,
) (func(), tcell.Color) {
	feedbackTextView.SetText(messagePushFeedback.Text())
	return nil, tcell.ColorGreen
}

//gocyclo:ignore
func (s *Shell) pushMessageWorker(
	ctx context.Context,
	pushMessagesLogsTextView *tview.TextView,
	commandsTextView *tview.TextView,
	gcodeParserTextView *tview.TextView,
	gcodeParamsTextView *tview.TextView,
	stateTextView *tview.TextView,
	statusTextView *tview.TextView,
	feedbackTextView *tview.TextView,
	pushMessageCh chan grblMod.Message,
) error {
	for {
		select {
		case <-ctx.Done():
			err := ctx.Err()
			if errors.Is(err, context.Canceled) {
				err = nil
			}
			return err
		case message, ok := <-pushMessageCh:
			if !ok {
				return fmt.Errorf("push message channel closed")
			}

			var color = tcell.ColorGreen
			var detailsFn func()
			if messagePushGcodeState, ok := message.(*grblMod.MessagePushGcodeState); ok {
				detailsFn, color = s.processMessagePushGcodeState(
					messagePushGcodeState, gcodeParserTextView,
				)
				if !s.options.DisplayGcodeParserStateComms {
					continue
				}
			}
			if _, ok := message.(*grblMod.MessagePushGcodeParam); ok {
				detailsFn, color = s.processMessagePushGcodeParam(gcodeParamsTextView)
				if !s.options.DisplayGcodeParamStateComms {
					continue
				}
			}

			if messagePushWelcome, ok := message.(*grblMod.MessagePushWelcome); ok {
				detailsFn, color = s.processMessagePushWelcome(
					ctx,
					messagePushWelcome,
					pushMessagesLogsTextView,
					commandsTextView,
					gcodeParserTextView,
					gcodeParamsTextView,
					stateTextView,
					statusTextView,
					feedbackTextView,
				)
			}
			if messagePushAlarm, ok := message.(*grblMod.MessagePushAlarm); ok {
				detailsFn, color = s.processMessagePushAlarm(
					messagePushAlarm, pushMessagesLogsTextView, stateTextView,
				)
			}
			if messagePushStatusReport, ok := message.(*grblMod.MessagePushStatusReport); ok {
				detailsFn, color = s.processMessagePushStatusReport(
					messagePushStatusReport, stateTextView, statusTextView,
				)
				if !s.options.DisplayStatusComms {
					continue
				}
			}
			if messagePushFeedback, ok := message.(*grblMod.MessagePushFeedback); ok {
				detailsFn, color = s.processMessagePushFeedback(
					messagePushFeedback, feedbackTextView,
				)
			}

			text := message.String()
			if len(text) == 0 {
				fmt.Fprintf(pushMessagesLogsTextView, "\n\n")
			} else {
				fmt.Fprintf(pushMessagesLogsTextView, "[%s]%s[-]\n", color, tview.Escape(text))
			}
			if detailsFn != nil {
				detailsFn()
			}
		}
	}
}

func (s *Shell) statusQueryWorker(ctx context.Context) error {
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

func (s *Shell) Run(ctx context.Context) (err error) {
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

	app,
		commandsTextView,
		pushMessagesLogsTextView,
		feedbackTextView,
		gcodeParserTextView,
		gcodeParamsTextView,
		stateTextView,
		statusTextView,
		commandInputField := s.getApp(sendCommandCh, sendRealTimeCommandCh)

	logger = slog.New(NewViewLogHandler(
		logger.Handler(),
		pushMessagesLogsTextView,
	))
	ctx = log.WithLogger(ctx, logger)

	go func() {
		defer cancelFn()
		defer app.Stop()
		sendCommandWorkerErrCh <- s.sendCommandWorker(
			ctx, commandsTextView, commandInputField, sendCommandCh,
		)
	}()
	go func() {
		defer cancelFn()
		defer app.Stop()
		sendRealTimeCommandWorkerErrCh <- s.sendRealTimeCommandWorker(
			ctx, commandsTextView, sendRealTimeCommandCh,
		)
	}()
	go func() {
		defer cancelFn()
		defer app.Stop()
		// Sending $G enables tracking of G-Code parsing state
		s.sendCommand(ctx, commandsTextView, "$G")
		// Sending $G enables tracking of G-Code parameters
		s.sendCommand(ctx, commandsTextView, "$#")
		pushMessageErrCh <- s.pushMessageWorker(
			ctx,
			pushMessagesLogsTextView,
			commandsTextView,
			gcodeParserTextView,
			gcodeParamsTextView,
			stateTextView,
			statusTextView,
			feedbackTextView,
			pushMessageCh,
		)
	}()
	go func() {
		defer cancelFn()
		defer app.Stop()
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

	return app.Run()
}
