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

func (s *Shell) getCommandInputField(
	ctx context.Context,
	commandsTextView *tview.TextView,
) *tview.InputField {
	commandInputField := tview.NewInputField().
		SetLabel("Command: ")
	commandInputField.SetDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEscape:
			commandInputField.SetText("")
		case tcell.KeyEnter:
			command := commandInputField.GetText()
			if command != "" {
				fmt.Fprintf(commandsTextView, "[%s]%s[-]\n", tcell.ColorWhite, tview.Escape(command))

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
				commandInputField.SetText("")
			}
		}
	})
	return commandInputField
}

func (s *Shell) pushMessageReceiver(
	ctx context.Context,
	pushMessageDoneCh chan struct{},
	pushMessageCh chan grblMod.Message,
	realTimeTextView *tview.TextView,
	stateTextView *tview.TextView,
	statusTextView *tview.TextView,
	feedbackTextView *tview.TextView,
) {
	defer func() { pushMessageDoneCh <- struct{}{} }()
	for {
		select {
		case <-ctx.Done():
			return
		case message, ok := <-pushMessageCh:
			if !ok {
				return
			}

			var color = tcell.ColorGreen
			var detailsFn func()
			if _, ok := message.(*grblMod.MessagePushWelcome); ok {
				color = tcell.ColorYellow
				detailsFn = func() {
					fmt.Fprintf(realTimeTextView, "[%s]Soft-Reset detected[-]\n", color)
				}
				stateTextView.SetBackgroundColor(tview.Styles.PrimitiveBackgroundColor)
				stateTextView.Clear()
				statusTextView.Clear()
				feedbackTextView.SetText("")
			}
			if messagePushAlarm, ok := message.(*grblMod.MessagePushAlarm); ok {
				color = tcell.ColorRed
				detailsFn = func() {
					fmt.Fprintf(realTimeTextView, "[%s]%s[-]\n", color, tview.Escape(messagePushAlarm.Error().Error()))
				}
			}
			if messagePushStatusReport, ok := message.(*grblMod.MessagePushStatusReport); ok {
				color = getMachineStateColor(messagePushStatusReport.MachineState.State)
				s.updateStatusReport(stateTextView, statusTextView, messagePushStatusReport)
				if !s.displayStatusComms {
					continue
				}
			}
			if messagePushFeedback, ok := message.(*grblMod.MessagePushFeedback); ok {
				feedbackTextView.SetText(messagePushFeedback.Text())
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

//gocyclo:ignore
func (s *Shell) writePositionStatus(w io.Writer, statusReport *grblMod.MessagePushStatusReport) {
	var mx, my, mz, ma, wx, wy, wz, wa *float64
	if statusReport.MachinePosition != nil {
		mx = &statusReport.MachinePosition.X
		my = &statusReport.MachinePosition.Y
		mz = &statusReport.MachinePosition.Z
		ma = statusReport.MachinePosition.A
		if s.grbl.WorkCoordinateOffset != nil {
			wxv := statusReport.MachinePosition.X - s.grbl.WorkCoordinateOffset.X
			wx = &wxv
			wyv := statusReport.MachinePosition.Y - s.grbl.WorkCoordinateOffset.Y
			wy = &wyv
			wzv := statusReport.MachinePosition.Z - s.grbl.WorkCoordinateOffset.Z
			wz = &wzv
			if statusReport.MachinePosition.A != nil && s.grbl.WorkCoordinateOffset.A != nil {
				wav := *statusReport.MachinePosition.A - *s.grbl.WorkCoordinateOffset.A
				wa = &wav
			}
		}
	}
	if statusReport.WorkPosition != nil {
		wx = &statusReport.WorkPosition.X
		wy = &statusReport.WorkPosition.Y
		wz = &statusReport.WorkPosition.Z
		wa = statusReport.WorkPosition.A
		if s.grbl.WorkCoordinateOffset != nil {
			mxv := statusReport.WorkPosition.X - s.grbl.WorkCoordinateOffset.X
			mx = &mxv
			myv := statusReport.WorkPosition.Y - s.grbl.WorkCoordinateOffset.Y
			my = &myv
			mzv := statusReport.WorkPosition.Z - s.grbl.WorkCoordinateOffset.Z
			mz = &mzv
			if statusReport.WorkPosition.A != nil && s.grbl.WorkCoordinateOffset.A != nil {
				mav := *statusReport.WorkPosition.A - *s.grbl.WorkCoordinateOffset.A
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

//gocyclo:ignore
func (s *Shell) updateStatusReport(
	stateView *tview.TextView,
	statusView *tview.TextView,
	statusReport *grblMod.MessagePushStatusReport,
) {
	stateColor := getMachineStateColor(statusReport.MachineState.State)

	stateView.Clear()
	stateView.SetBackgroundColor(stateColor)
	_, _, stateViewWidth, _ := stateView.GetRect()
	state := statusReport.MachineState.State
	fmt.Fprintf(
		stateView, "%s%s\n",
		strings.Repeat(" ", (stateViewWidth-2-len(state))/2),
		tview.Escape(state),
	)
	if statusReport.MachineState.SubState != nil {
		subState := statusReport.MachineState.SubStateString()
		fmt.Fprintf(
			stateView, "%s(%s)\n",
			strings.Repeat(" ", (stateViewWidth-4-len(subState))/2),
			tview.Escape(subState),
		)
	}

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

	if s.grbl.OverrideValues != nil {
		fmt.Fprint(statusView, "\nOverrides\n")
		fmt.Fprintf(statusView, "Feed:%.0f%%\n", s.grbl.OverrideValues.Feed)
		fmt.Fprintf(statusView, "Rapids:%.0f%%\n", s.grbl.OverrideValues.Rapids)
		fmt.Fprintf(statusView, "Spindle:%.0f%%\n", s.grbl.OverrideValues.Spindle)
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

func (s *Shell) sendRealTimeCommand(ctx context.Context, cmd grblMod.RealTimeCommand, realTimeTextView *tview.TextView) error {
	if err := s.grbl.SendRealTimeCommand(ctx, cmd); err != nil {
		return err
	}
	if s.displayStatusComms || cmd != grblMod.RealTimeCommandStatusReportQuery {
		fmt.Fprintf(realTimeTextView, "[%s]%s[-]\n", tcell.ColorWhite, tview.Escape(cmd.String()))
	}
	return nil
}

func (s *Shell) statusQueryWorker(ctx context.Context, doneCh chan error) {
	for {
		select {
		case <-ctx.Done():
			doneCh <- nil
			return
		case <-time.After(200 * time.Millisecond):
			if err := s.grbl.SendRealTimeCommand(ctx, grblMod.RealTimeCommandStatusReportQuery); err != nil {
				doneCh <- fmt.Errorf("failed to send periodic status query real-time command: %w", err)
				return
			}
		}
	}
}

func (s *Shell) Run(ctx context.Context) error {
	logger := log.MustLogger(ctx)
	logger.Info("Connecting")

	pushMessageCh, err := s.grbl.Connect(ctx)
	if err != nil {
		return err
	}
	pushMessageDoneCh := make(chan struct{})
	statusQueryWorkerErrCh := make(chan error)
	statusQueryWorkerCtx, statusQueryWorkerCancel := context.WithCancel(ctx)
	go s.statusQueryWorker(statusQueryWorkerCtx, statusQueryWorkerErrCh)
	defer func() {
		logger.Info("Stopping status query worker")
		statusQueryWorkerCancel()
		if statusQueryErr := <-statusQueryWorkerErrCh; statusQueryErr != nil {
			err = errors.Join(err, statusQueryErr)
		}
		logger.Info("Disconnecting")
		err = errors.Join(err, s.grbl.Disconnect(ctx))
		logger.Info("waiting for push message receiver")
		<-pushMessageDoneCh
	}()

	app := tview.NewApplication()
	app.EnableMouse(true)

	commandsTextView := s.getCommandsTextView(app)
	realTimeTextView := s.getRealTimeTextView(app)
	feedbackTextView := s.getFeedbackTextView(app)
	stateTextView := s.getStateTextView(app)
	statusTextView := s.getStatusTextView(app)
	commandInputField := s.getCommandInputField(ctx, commandsTextView)

	go s.pushMessageReceiver(
		ctx,
		pushMessageDoneCh,
		pushMessageCh,
		realTimeTextView,
		stateTextView,
		statusTextView,
		feedbackTextView,
	)

	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyCtrlX {
			if err := s.sendRealTimeCommand(ctx, grblMod.RealTimeCommandSoftReset, realTimeTextView); err != nil {
				fmt.Fprintf(realTimeTextView, "[%s]Failed to send soft reset: %s[-]\n", tcell.ColorRed, err)
			}
			return nil
		}
		return event
	})

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

	return app.SetRoot(rootFlex, true).SetFocus(commandInputField).Run()
}
