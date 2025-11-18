package control

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

//gocyclo:ignore
func (s *Control) writePositionStatus(w io.Writer, statusReport *grblMod.MessagePushStatusReport) {
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

func (s *Control) updateState(
	state string,
	subState string,
) {
	stateColor := getMachineStateColor(state)

	s.AppManager.StateTextView.Clear()
	s.AppManager.StateTextView.SetBackgroundColor(stateColor)
	_, _, stateViewWidth, _ := s.AppManager.StateTextView.GetRect()
	fmt.Fprintf(
		s.AppManager.StateTextView, "%s%s\n",
		strings.Repeat(" ", (stateViewWidth-2-len(state))/2),
		tview.Escape(state),
	)
	if len(subState) > 0 {
		fmt.Fprintf(
			s.AppManager.StateTextView, "%s(%s)\n",
			strings.Repeat(" ", (stateViewWidth-4-len(subState))/2),
			tview.Escape(subState),
		)
	}
}

//gocyclo:ignore
func (s *Control) updateStatusReport(
	statusReport *grblMod.MessagePushStatusReport,
) {
	s.updateState(statusReport.MachineState.State, statusReport.MachineState.SubStateString())

	s.AppManager.StatusTextView.Clear()

	s.writePositionStatus(s.AppManager.StatusTextView, statusReport)

	if statusReport.BufferState != nil {
		fmt.Fprint(s.AppManager.StatusTextView, "\nBuffer\n")
		fmt.Fprintf(s.AppManager.StatusTextView, "Blocks:%d\n", statusReport.BufferState.AvailableBlocks)
		fmt.Fprintf(s.AppManager.StatusTextView, "Bytes:%d\n", statusReport.BufferState.AvailableBytes)
	}

	if statusReport.LineNumber != nil {
		fmt.Fprintf(s.AppManager.StatusTextView, "\nLine:%d\n", *statusReport.LineNumber)
	}

	if statusReport.Feed != nil {
		fmt.Fprintf(s.AppManager.StatusTextView, "\nFeed:%.1f\n", *statusReport.Feed)
	}

	if statusReport.FeedSpindle != nil {
		fmt.Fprintf(s.AppManager.StatusTextView, "\nFeed:%.0f\n", statusReport.FeedSpindle.Feed)
		fmt.Fprintf(s.AppManager.StatusTextView, "Speed:%.0f\n", statusReport.FeedSpindle.Speed)
	}

	if statusReport.PinState != nil {
		fmt.Fprintf(s.AppManager.StatusTextView, "\nPin:%s\n", statusReport.PinState)
	}

	if s.grbl.GetOverrideValues() != nil {
		fmt.Fprint(s.AppManager.StatusTextView, "\nOverrides\n")
		fmt.Fprintf(s.AppManager.StatusTextView, "Feed:%.0f%%\n", s.grbl.GetOverrideValues().Feed)
		fmt.Fprintf(s.AppManager.StatusTextView, "Rapids:%.0f%%\n", s.grbl.GetOverrideValues().Rapids)
		fmt.Fprintf(s.AppManager.StatusTextView, "Spindle:%.0f%%\n", s.grbl.GetOverrideValues().Spindle)
	}

	if statusReport.AccessoryState != nil {
		fmt.Fprint(s.AppManager.StatusTextView, "\nAccessory\n")
		if statusReport.AccessoryState.SpindleCW != nil && *statusReport.AccessoryState.SpindleCW {
			fmt.Fprint(s.AppManager.StatusTextView, "Spindle: CW")
		}
		if statusReport.AccessoryState.SpindleCCW != nil && *statusReport.AccessoryState.SpindleCCW {
			fmt.Fprint(s.AppManager.StatusTextView, "Spindle: CCW")
		}
		if statusReport.AccessoryState.FloodCoolant != nil && *statusReport.AccessoryState.FloodCoolant {
			fmt.Fprint(s.AppManager.StatusTextView, "Flood Coolant")
		}
		if statusReport.AccessoryState.MistCoolant != nil && *statusReport.AccessoryState.MistCoolant {
			fmt.Fprint(s.AppManager.StatusTextView, "Mist Coolant")
		}
	}
}

//gocyclo:ignore
func (s *Control) processMessagePushGcodeState(
	messagePushGcodeState *grblMod.MessagePushGcodeState,
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

	s.AppManager.GcodeParserTextView.Clear()
	fmt.Fprint(s.AppManager.GcodeParserTextView, tview.Escape(buf.String()))

	return nil, tcell.ColorGreen
}

//gocyclo:ignore
func (s *Control) processMessagePushGcodeParam() (func(), tcell.Color) {
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

	s.AppManager.GcodeParamsTextView.Clear()
	fmt.Fprint(s.AppManager.GcodeParamsTextView, tview.Escape(buf.String()))

	return nil, color
}

func (s *Control) processMessagePushWelcome(
	ctx context.Context,
	_ *grblMod.MessagePushWelcome,
) (func(), tcell.Color) {
	color := tcell.ColorYellow
	detailsFn := func() {
		fmt.Fprintf(s.AppManager.PushMessagesLogsTextView, "[%s]Soft-Reset detected[-]\n", color)
	}
	s.AppManager.GcodeParserTextView.Clear()
	s.AppManager.GcodeParamsTextView.Clear()
	s.AppManager.StateTextView.SetBackgroundColor(tview.Styles.PrimitiveBackgroundColor)
	s.AppManager.StateTextView.Clear()
	s.AppManager.StatusTextView.Clear()
	s.AppManager.FeedbackTextView.SetText("")
	// Sending $G enables tracking of G-Code parsing state
	s.sendCommand(ctx, "$G")
	// Sending $G enables tracking of G-Code parameters
	s.sendCommand(ctx, "$#")
	return detailsFn, color
}

func (s *Control) processMessagePushAlarm(
	messagePushAlarm *grblMod.MessagePushAlarm,
) (func(), tcell.Color) {
	color := tcell.ColorRed
	detailsFn := func() {
		fmt.Fprintf(s.AppManager.PushMessagesLogsTextView, "[%s]%s[-]\n", color, tview.Escape(messagePushAlarm.Error().Error()))
	}
	s.updateState("Alarm", "")
	return detailsFn, color
}

func (s *Control) processMessagePushStatusReport(
	messagePushStatusReport *grblMod.MessagePushStatusReport,
) (func(), tcell.Color) {
	color := getMachineStateColor(messagePushStatusReport.MachineState.State)
	s.updateStatusReport(messagePushStatusReport)
	return nil, color
}

func (s *Control) processMessagePushFeedback(
	messagePushFeedback *grblMod.MessagePushFeedback,
) (func(), tcell.Color) {
	s.AppManager.FeedbackTextView.SetText(messagePushFeedback.Text())
	return nil, tcell.ColorGreen
}

//gocyclo:ignore
func (s *Control) pushMessageWorker(
	ctx context.Context,
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
				detailsFn, color = s.processMessagePushGcodeState(messagePushGcodeState)
				if !s.options.DisplayGcodeParserStateComms {
					continue
				}
			}
			if _, ok := message.(*grblMod.MessagePushGcodeParam); ok {
				detailsFn, color = s.processMessagePushGcodeParam()
				if !s.options.DisplayGcodeParamStateComms {
					continue
				}
			}

			if messagePushWelcome, ok := message.(*grblMod.MessagePushWelcome); ok {
				detailsFn, color = s.processMessagePushWelcome(
					ctx,
					messagePushWelcome,
				)
			}
			if messagePushAlarm, ok := message.(*grblMod.MessagePushAlarm); ok {
				detailsFn, color = s.processMessagePushAlarm(messagePushAlarm)
			}
			if messagePushStatusReport, ok := message.(*grblMod.MessagePushStatusReport); ok {
				detailsFn, color = s.processMessagePushStatusReport(messagePushStatusReport)
				if !s.options.DisplayStatusComms {
					continue
				}
			}
			if messagePushFeedback, ok := message.(*grblMod.MessagePushFeedback); ok {
				detailsFn, color = s.processMessagePushFeedback(messagePushFeedback)
			}

			text := message.String()
			if len(text) == 0 {
				fmt.Fprintf(s.AppManager.PushMessagesLogsTextView, "\n\n")
			} else {
				fmt.Fprintf(s.AppManager.PushMessagesLogsTextView, "[%s]%s[-]\n", color, tview.Escape(text))
			}
			if detailsFn != nil {
				detailsFn()
			}
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
		pushMessageErrCh <- s.pushMessageWorker(ctx, pushMessageCh)
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
