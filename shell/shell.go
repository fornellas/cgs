package shell

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/fornellas/slogxt/log"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	grblMod "github.com/fornellas/cgs/grbl"
)

type Shell struct {
	grbl               *grblMod.Grbl
	displayStatusComms bool
}

func NewShell(grbl *grblMod.Grbl, displayStatusComms bool) *Shell {
	return &Shell{
		grbl:               grbl,
		displayStatusComms: displayStatusComms,
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
	commandsTextView := tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true).
		SetWrap(true)
	commandsTextView.SetBorder(true).SetTitle("Commands")
	commandsTextView.SetChangedFunc(func() {
		commandsTextView.ScrollToEnd()
		app.Draw()
	})
	return commandsTextView
}

func (s *Shell) getRealTimeTextView(app *tview.Application) *tview.TextView {
	realTimeTextView := tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true).
		SetWrap(true)
	realTimeTextView.SetBorder(true).SetTitle("Real-Time")
	realTimeTextView.SetChangedFunc(func() {
		realTimeTextView.ScrollToEnd()
		app.Draw()
	})
	return realTimeTextView
}

func (s *Shell) getFeedbackTextView(app *tview.Application) *tview.TextView {
	feedbackTextView := tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true).
		SetWrap(true)
	feedbackTextView.SetTitle("Feedback Message")
	feedbackTextView.SetChangedFunc(func() {
		feedbackTextView.ScrollToEnd()
		app.Draw()
	})
	return feedbackTextView
}

func (s *Shell) getStateTextView(app *tview.Application) *tview.TextView {
	stateTextView := tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true).
		SetWrap(true)
	stateTextView.SetBorder(true).SetTitle("State")
	stateTextView.SetChangedFunc(func() {
		app.Draw()
	})
	return stateTextView
}

func (s *Shell) getStatusTextView(app *tview.Application) *tview.TextView {
	statusTextView := tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true).
		SetWrap(true)
	statusTextView.SetBorder(true).SetTitle("Status")
	statusTextView.SetChangedFunc(func() {
		app.Draw()
	})
	return statusTextView
}

func (s *Shell) getCommandInputField(commandCh chan string) *tview.InputField {
	commandInputField := tview.NewInputField().
		SetLabel("Command: ")
	commandInputField.SetDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEscape:
			commandInputField.SetText("")
		case tcell.KeyEnter:
			command := commandInputField.GetText()
			if command == "" {
				return
			}
			commandCh <- command
			commandInputField.SetText("")
		}
	})
	return commandInputField
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
	realTimeTextView := s.getRealTimeTextView(app)
	feedbackTextView := s.getFeedbackTextView(app)
	stateTextView := s.getStateTextView(app)
	statusTextView := s.getStatusTextView(app)
	commandInputField := s.getCommandInputField(sendCommandCh)
	rootFlex := tview.NewFlex().
		AddItem(
			tview.NewFlex().SetDirection(tview.FlexRow).
				AddItem(
					tview.NewFlex().
						AddItem(
							tview.NewFlex().SetDirection(tview.FlexRow).
								AddItem(realTimeTextView, 0, 1, false).
								AddItem(commandsTextView, 0, 1, false),
							0, 3, false,
						).
						AddItem(tview.NewBox().SetBorder(true).SetTitle("G-Code Parser"), 0, 1, false).
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
		)
	app.SetRoot(rootFlex, true).SetFocus(commandInputField)
	return app,
		commandsTextView,
		realTimeTextView,
		feedbackTextView,
		stateTextView,
		statusTextView
}

func (s *Shell) sendCommandWorker(
	ctx context.Context,
	commandsTextView *tview.TextView,
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
			fmt.Fprintf(commandsTextView, "[%s]%s[-]\n", tcell.ColorWhite, tview.Escape(command))
			// FIXME $H (and maybe others) require bigger timeout
			// FIXME ! (and maybe others) "sometimes" don't return message
			ctx, cancel := context.WithDeadline(ctx, time.Now().Add(time.Second))
			defer cancel()
			message, err := s.grbl.SendCommand(ctx, command)
			if err != nil {
				fmt.Fprintf(commandsTextView, "[%s]Failed to send: %s[-]\n", tcell.ColorRed, tview.Escape(err.Error()))
			} else {
				messageResponse := message.(*grblMod.MessageResponse)
				if messageResponse.Error() == nil {
					fmt.Fprintf(commandsTextView, "[%s]%s[-]\n", tcell.ColorGreen, tview.Escape(messageResponse.String()))
				} else {
					fmt.Fprintf(commandsTextView, "[%s]%s[-]\n", tcell.ColorRed, tview.Escape(messageResponse.String()))
					fmt.Fprintf(commandsTextView, "[%s]%s[-]\n", tcell.ColorRed, tview.Escape(messageResponse.Error().Error()))
				}
			}
		}
	}
}

func (s *Shell) sendRealTimeCommand(
	ctx context.Context,
	cmd grblMod.RealTimeCommand,
	realTimeTextView *tview.TextView,
) error {
	if err := s.grbl.SendRealTimeCommand(ctx, cmd); err != nil {
		return err
	}
	if s.displayStatusComms || cmd != grblMod.RealTimeCommandStatusReportQuery {
		fmt.Fprintf(realTimeTextView, "[%s]%s[-]\n", tcell.ColorWhite, tview.Escape(cmd.String()))
	}
	return nil
}

func (s *Shell) sendRealTimeCommandWorker(
	ctx context.Context,
	realTimeTextView *tview.TextView,
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
			if err := s.sendRealTimeCommand(ctx, realTimeCommand, realTimeTextView); err != nil {
				fmt.Fprintf(realTimeTextView, "[%s]Failed to send soft reset: %s[-]\n", tcell.ColorRed, err)
			}
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

func (s *Shell) processMessagePushWelcome(
	_ *grblMod.MessagePushWelcome,
	realTimeTextView *tview.TextView,
	stateTextView *tview.TextView,
	statusTextView *tview.TextView,
	feedbackTextView *tview.TextView,
) (func(), tcell.Color) {
	color := tcell.ColorYellow
	detailsFn := func() {
		fmt.Fprintf(realTimeTextView, "[%s]Soft-Reset detected[-]\n", color)
	}
	stateTextView.SetBackgroundColor(tview.Styles.PrimitiveBackgroundColor)
	stateTextView.Clear()
	statusTextView.Clear()
	feedbackTextView.SetText("")
	return detailsFn, color
}

func (s *Shell) processMessagePushAlarm(
	messagePushAlarm *grblMod.MessagePushAlarm,
	realTimeTextView *tview.TextView,
	stateTextView *tview.TextView,
) (func(), tcell.Color) {
	color := tcell.ColorRed
	detailsFn := func() {
		fmt.Fprintf(realTimeTextView, "[%s]%s[-]\n", color, tview.Escape(messagePushAlarm.Error().Error()))
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

func (s *Shell) pushMessageWorker(
	ctx context.Context,
	realTimeTextView *tview.TextView,
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
			if messagePushWelcome, ok := message.(*grblMod.MessagePushWelcome); ok {
				detailsFn, color = s.processMessagePushWelcome(
					messagePushWelcome, realTimeTextView, stateTextView, statusTextView, feedbackTextView,
				)
			}
			if messagePushAlarm, ok := message.(*grblMod.MessagePushAlarm); ok {
				detailsFn, color = s.processMessagePushAlarm(
					messagePushAlarm, realTimeTextView, stateTextView,
				)
			}
			if messagePushStatusReport, ok := message.(*grblMod.MessagePushStatusReport); ok {
				detailsFn, color = s.processMessagePushStatusReport(
					messagePushStatusReport, stateTextView, statusTextView,
				)
				if !s.displayStatusComms {
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
				fmt.Fprintf(realTimeTextView, "\n\n")
			} else {
				fmt.Fprintf(realTimeTextView, "[%s]%s[-]\n", color, tview.Escape(text))
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
			if err := s.grbl.SendRealTimeCommand(ctx, grblMod.RealTimeCommandStatusReportQuery); err != nil {
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
		realTimeTextView,
		feedbackTextView,
		stateTextView,
		statusTextView := s.getApp(sendCommandCh, sendRealTimeCommandCh)

	go func() {
		defer cancelFn()
		defer app.Stop()
		sendCommandWorkerErrCh <- s.sendCommandWorker(
			ctx, commandsTextView, sendCommandCh,
		)
	}()
	go func() {
		defer cancelFn()
		defer app.Stop()
		sendRealTimeCommandWorkerErrCh <- s.sendRealTimeCommandWorker(
			ctx, realTimeTextView, sendRealTimeCommandCh,
		)
	}()
	go func() {
		defer cancelFn()
		defer app.Stop()
		pushMessageErrCh <- s.pushMessageWorker(
			ctx, realTimeTextView, stateTextView, statusTextView, feedbackTextView, pushMessageCh,
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
		err = errors.Join(err, s.grbl.Disconnect(ctx))
	}()

	return app.Run()
}
