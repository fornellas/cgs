package oldshell

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/fornellas/slogxt/log"
	"github.com/jroimartin/gocui"

	"github.com/fornellas/cgs/grbl"
)

type ViewManager interface {
	GetManagerFn(gui *gocui.Gui, x0, y0, x1, y1 int) func(gui *gocui.Gui) error
	InitKeybindings(g *gocui.Gui) error
}

type Shell struct {
	grbl                        *grbl.Grbl
	grblCommandViewName         string
	grblRealtimeCommandViewName string
	gcodeParserStateViewName    string
	statusViewName              string
	feedbackMessageViewName     string
	promptViewName              string
	enableStatusMessages        bool
}

func NewShell(grbl *grbl.Grbl, enableStatusMessages bool) *Shell {
	return &Shell{
		grbl:                        grbl,
		grblCommandViewName:         "grblCommand",
		grblRealtimeCommandViewName: "grblRealtimeCommand",
		gcodeParserStateViewName:    "gcodeParserState",
		statusViewName:              "status",
		feedbackMessageViewName:     "feedbackMessage",
		promptViewName:              "prompt",
		enableStatusMessages:        enableStatusMessages,
	}
}

func (s *Shell) getManagerFn(
	gui *gocui.Gui,
	grblRealtimeCommandViewManager *GrblRealTimeCommandViewManager,
	grblCommandViewManager *GrblCommandViewManager,
	promptViewManager *PromptViewManager,
) func(*gocui.Gui) error {
	return func(*gocui.Gui) error {
		maxX, maxY := gui.Size()

		feedbackMessageHeight := 3
		promptHeight := 3
		gcodeWidth := 20
		statusWidth := 14
		realtimeCommandY1 := (maxY - (1 + feedbackMessageHeight + promptHeight)) / 2
		commandY0 := realtimeCommandY1 + 1

		// Grbl Real-time Command
		grblRealtimeCommandManagerFn := grblRealtimeCommandViewManager.GetManagerFn(gui, 0, 0, maxX-(gcodeWidth+statusWidth+3), realtimeCommandY1)
		if err := grblRealtimeCommandManagerFn(gui); err != nil {
			return fmt.Errorf("shell: manager: Grbl view manager failed: %w", err)
		}

		// Grbl Command
		grblCommandManagerFn := grblCommandViewManager.GetManagerFn(gui, 0, commandY0, maxX-(gcodeWidth+statusWidth+3), maxY-(1+feedbackMessageHeight+promptHeight))
		if err := grblCommandManagerFn(gui); err != nil {
			return fmt.Errorf("shell: manager: Grbl view manager failed: %w", err)
		}

		// G-Code Parser
		if gCodeParserView, err := gui.SetView(s.gcodeParserStateViewName, maxX-(gcodeWidth+statusWidth+2), 0, maxX-statusWidth-1, maxY-(1+feedbackMessageHeight+promptHeight)); err != nil {
			if err != gocui.ErrUnknownView {
				return err
			}
			gCodeParserView.Title = "G-Code Parser"
			gCodeParserView.Wrap = true
		}

		// Status
		if statusView, err := gui.SetView(s.statusViewName, maxX-statusWidth, 0, maxX-1, maxY-(1+feedbackMessageHeight+promptHeight)); err != nil {
			if err != gocui.ErrUnknownView {
				return err
			}
			statusView.Title = "Status"
		}

		// Feedback Message
		if feedbackMessageView, err := gui.SetView(s.feedbackMessageViewName, 0, maxY-6, maxX-1, maxY-(1+feedbackMessageHeight)); err != nil {
			if err != gocui.ErrUnknownView {
				return err
			}
			feedbackMessageView.Title = "Feedback Message"
			feedbackMessageView.Wrap = true
			feedbackMessageView.Autoscroll = true
		}

		// Prompt
		promptManagerFn := promptViewManager.GetManagerFn(gui, 0, maxY-promptHeight, maxX-1, maxY-1)
		if err := promptManagerFn(gui); err != nil {
			return fmt.Errorf("shell: manager: prompt view manager failed: %w", err)
		}
		if _, err := gui.SetCurrentView(s.promptViewName); err != nil {
			return fmt.Errorf("shell: manager: failed to set current view to prompt: %w", err)
		}

		return nil
	}
}

func (s *Shell) grblSendCommand(ctx context.Context, gui *gocui.Gui, command string) error {
	grblCommandView, err := gui.View(s.grblCommandViewName)
	if err != nil {
		return fmt.Errorf("shell: handleCommand: failed to get Grbl view: %w", err)
	}

	line := fmt.Sprintf("> %s\n", command)
	n, err := fmt.Fprint(grblCommandView, line)
	if err != nil {
		return fmt.Errorf("shell: handleCommand: failed to write to Grbl view: %w", err)
	}
	if n != len(line) {
		return fmt.Errorf("shell: handleCommand: short write to Grbl view: expected %d, got %d", len(line), n)
	}

	message, err := s.grbl.SendCommand(ctx, command)
	messageResponse := message.(*grbl.MessageResponse)
	if err != nil {
		line := fmt.Sprintf("< %s\n", messageResponse.String())
		n, err := fmt.Fprint(grblCommandView, line)
		if err != nil {
			return fmt.Errorf("shell: handleCommand: failed to write to Grbl view: %w", err)
		}
		if n != len(line) {
			return fmt.Errorf("shell: handleCommand: short write to Grbl view: expected %d, got %d", len(line), n)
		}
	}

	return nil
}

func (s *Shell) grblSendRealTimeCommand(ctx context.Context, gui *gocui.Gui, command grbl.RealTimeCommand) error {
	if err := s.grbl.SendRealTimeCommand(ctx, command); err != nil {
		return fmt.Errorf("shell: handleCommand: failed to send command to Grbl: %w", err)
	}
	if !s.enableStatusMessages && command == grbl.RealTimeCommandStatusReportQuery {
		return nil
	}
	grblCommandView, err := gui.View(s.grblRealtimeCommandViewName)
	if err != nil {
		return fmt.Errorf("shell: handleCommand: failed to get Grbl view: %w", err)
	}
	line := fmt.Sprintf("> %s\n", command)
	n, err := fmt.Fprint(grblCommandView, line)
	if err != nil {
		return fmt.Errorf("shell: handleCommand: failed to write to Grbl view: %w", err)
	}
	if n != len(line) {
		return fmt.Errorf("shell: handleCommand: short write to Grbl view: expected %d, got %d", len(line), n)
	}
	return nil
}

func (s *Shell) getHandleSendCommandFn(ctx context.Context) func(gui *gocui.Gui, command string) error {
	return func(gui *gocui.Gui, command string) error {
		if err := s.grblSendCommand(ctx, gui, command); err != nil {
			return err
		}
		// $G after each sent block enables accurate tracking of g-code parser state
		if err := s.grblSendCommand(ctx, gui, "$G"); err != nil {
			return err
		}
		return nil
	}
}

func (s *Shell) getHandleResetFn(ctx context.Context) func(gui *gocui.Gui) error {
	return func(gui *gocui.Gui) error {
		return s.grblSendRealTimeCommand(ctx, gui, grbl.RealTimeCommandSoftReset)
	}
}

func (s *Shell) handleKeyBindQuit(g *gocui.Gui, v *gocui.View) error {
	return gocui.ErrQuit
}

func (s *Shell) setKeybindings(gui *gocui.Gui, viewManagers []ViewManager) error {
	if err := gui.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, s.handleKeyBindQuit); err != nil {
		return fmt.Errorf("shell: failed so set keybinding: %w", err)
	}

	for _, viewManager := range viewManagers {
		if err := viewManager.InitKeybindings(gui); err != nil {
			return fmt.Errorf("shell: failed so set keybinding: %w", err)
		}
	}

	return nil
}

func (s *Shell) receiverHandleMessagePushGcodeState(
	ctx context.Context,
	gui *gocui.Gui,
	messagePushFeedback *grbl.MessagePushGcodeState,
) bool {
	logger := log.MustLogger(ctx)

	gcodeParserStateView, err := gui.View(s.gcodeParserStateViewName)
	if err != nil {
		logger.Error("Receiver", "err", fmt.Errorf("shell: receiver: failed to get Grbl view: %w", err))
		return true
	}
	gcodeParserStateView.Clear()

	var buf bytes.Buffer

	if messagePushFeedback.ModalGroup.Motion != nil {
		fmt.Fprintf(&buf, "%s:%s\n", messagePushFeedback.ModalGroup.Motion.NormalizedString(), messagePushFeedback.ModalGroup.Motion.Name())
	}
	if messagePushFeedback.ModalGroup.PlaneSelection != nil {
		fmt.Fprintf(&buf, "%s:%s\n", messagePushFeedback.ModalGroup.PlaneSelection.NormalizedString(), messagePushFeedback.ModalGroup.PlaneSelection.Name())
	}
	if messagePushFeedback.ModalGroup.DistanceMode != nil {
		fmt.Fprintf(&buf, "%s:%s\n", messagePushFeedback.ModalGroup.DistanceMode.NormalizedString(), messagePushFeedback.ModalGroup.DistanceMode.Name())
	}
	if messagePushFeedback.ModalGroup.Units != nil {
		fmt.Fprintf(&buf, "%s:%s\n", messagePushFeedback.ModalGroup.Units.NormalizedString(), messagePushFeedback.ModalGroup.Units.Name())
	}
	if messagePushFeedback.ModalGroup.ToolLengthOffset != nil {
		fmt.Fprintf(&buf, "%s:%s\n", messagePushFeedback.ModalGroup.ToolLengthOffset.NormalizedString(), messagePushFeedback.ModalGroup.ToolLengthOffset.Name())
	}
	if messagePushFeedback.ModalGroup.CoordinateSystemSelection != nil {
		fmt.Fprintf(&buf, "%s:%s\n", messagePushFeedback.ModalGroup.CoordinateSystemSelection.NormalizedString(), messagePushFeedback.ModalGroup.CoordinateSystemSelection.Name())
	}
	if messagePushFeedback.ModalGroup.SpindleTurning != nil {
		fmt.Fprintf(&buf, "%s:%s\n", messagePushFeedback.ModalGroup.SpindleTurning.NormalizedString(), messagePushFeedback.ModalGroup.SpindleTurning.Name())
	}
	for _, word := range messagePushFeedback.ModalGroup.Coolant {
		fmt.Fprintf(&buf, "%s:%s\n", word.NormalizedString(), word.Name())
	}
	if messagePushFeedback.Tool != nil {
		fmt.Fprintf(&buf, "Tool: %.0f\n", *messagePushFeedback.Tool)
	}
	if messagePushFeedback.FeedRate != nil {
		fmt.Fprintf(&buf, "Feed Rate: %.0f\n", *messagePushFeedback.FeedRate)
	}
	if messagePushFeedback.SpindleSpeed != nil {
		fmt.Fprintf(&buf, "Speed: %.0f\n", *messagePushFeedback.SpindleSpeed)
	}

	n, err := gcodeParserStateView.Write(buf.Bytes())
	if err != nil {
		logger.Error("Receiver", "err", fmt.Errorf("shell: receiver: failed to write to Grbl view: %w", err))
		return true
	}
	if n != len(buf.Bytes()) {
		logger.Error("Receiver", "err", fmt.Errorf("shell: receiver: short write to Grbl view: expected %d, got %d", len(buf.Bytes()), n))
		return true
	}
	return false
}

func (s *Shell) receiverHandleMessagePushFeedback(
	ctx context.Context,
	gui *gocui.Gui,
	messagePushFeedback *grbl.MessagePushFeedback,
) bool {
	logger := log.MustLogger(ctx)
	feedbackMessageView, err := gui.View(s.feedbackMessageViewName)
	if err != nil {
		logger.Error("Receiver", "err", fmt.Errorf("shell: receiver: failed to get Grbl view: %w", err))
		return true
	}
	feedbackMessageView.Clear()
	text := messagePushFeedback.Text()
	n, err := fmt.Fprint(feedbackMessageView, text)
	if err != nil {
		logger.Error("Receiver", "err", fmt.Errorf("shell: receiver: failed to write to Grbl view: %w", err))
		return true
	}
	if n != len(text) {
		logger.Error("Receiver", "err", fmt.Errorf("shell: receiver: short write to Grbl view: expected %d, got %d", len(text), n))
		return true
	}
	return false
}

//gocyclo:ignore
func (s *Shell) receiverHandleMessagePushStatusReportPosition(
	buf *bytes.Buffer,
	statusReport *grbl.MessagePushStatusReport,
) {
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
	if wx != nil || wy != nil || wz != nil || wa != nil {
		fmt.Fprintf(buf, "\nWork\n")
	}
	if wx != nil {
		fmt.Fprintf(buf, "X:%.3f\n", *wx)
	}
	if wy != nil {
		fmt.Fprintf(buf, "Y:%.3f\n", *wy)
	}
	if wz != nil {
		fmt.Fprintf(buf, "Z:%.3f\n", *wz)
	}
	if wa != nil {
		fmt.Fprintf(buf, "A:%.3f\n", *wa)
	}
	if mx != nil || my != nil || mz != nil || ma != nil {
		fmt.Fprintf(buf, "Machine\n")
	}
	if mx != nil {
		fmt.Fprintf(buf, "X:%.3f\n", *mx)
	}
	if my != nil {
		fmt.Fprintf(buf, "Y:%.3f\n", *my)
	}
	if mz != nil {
		fmt.Fprintf(buf, "Z:%.3f\n", *mz)
	}
	if ma != nil {
		fmt.Fprintf(buf, "A:%.3f\n", *ma)
	}
}

//gocyclo:ignore
func (s *Shell) receiverHandleMessagePushStatusReport(
	ctx context.Context,
	gui *gocui.Gui,
	statusReport *grbl.MessagePushStatusReport,
) bool {
	logger := log.MustLogger(ctx)

	statusView, err := gui.View(s.statusViewName)
	if err != nil {
		logger.Error("Receiver", "err", fmt.Errorf("shell: receiver: failed to get Grbl view: %w", err))
		return true
	}

	var buf bytes.Buffer

	fmt.Fprintf(&buf, "[%s]\n", statusReport.MachineState.State)
	if statusReport.MachineState.SubState != nil {
		fmt.Fprintf(&buf, "(%s)\n", statusReport.MachineState.SubStateString())
	}

	s.receiverHandleMessagePushStatusReportPosition(&buf, statusReport)

	if statusReport.BufferState != nil {
		fmt.Fprint(&buf, "\nBuffer\n")
		fmt.Fprintf(&buf, "Blocks:%d\n", statusReport.BufferState.AvailableBlocks)
		fmt.Fprintf(&buf, "Bytes:%d\n", statusReport.BufferState.AvailableBytes)
	}

	if statusReport.LineNumber != nil {
		fmt.Fprintf(&buf, "\nLine:%d\n", *statusReport.LineNumber)
	}

	if statusReport.Feed != nil {
		fmt.Fprintf(&buf, "\nFeed:%.1f\n", *statusReport.Feed)
	}

	if statusReport.FeedSpindle != nil {
		fmt.Fprintf(&buf, "\nFeed:%.0f\n", statusReport.FeedSpindle.Feed)
		fmt.Fprintf(&buf, "Speed:%.0f\n", statusReport.FeedSpindle.Speed)
	}

	if statusReport.PinState != nil {
		fmt.Fprintf(&buf, "\nPin:%s\n", statusReport.PinState)
	}

	if s.grbl.GetOverrideValues() != nil {
		fmt.Fprint(&buf, "\nOverrides\n")
		fmt.Fprintf(&buf, "Feed:%.0f%%\n", s.grbl.GetOverrideValues().Feed)
		fmt.Fprintf(&buf, "Rapids:%.0f%%\n", s.grbl.GetOverrideValues().Rapids)
		fmt.Fprintf(&buf, "Spindle:%.0f%%\n", s.grbl.GetOverrideValues().Spindle)
	}

	if statusReport.AccessoryState != nil {
		fmt.Fprint(&buf, "\nAccessory\n")
		if statusReport.AccessoryState.SpindleCW != nil && *statusReport.AccessoryState.SpindleCW {
			fmt.Fprint(&buf, "Spindle: CW")
		}
		if statusReport.AccessoryState.SpindleCCW != nil && *statusReport.AccessoryState.SpindleCCW {
			fmt.Fprint(&buf, "Spindle: CCW")
		}
		if statusReport.AccessoryState.FloodCoolant != nil && *statusReport.AccessoryState.FloodCoolant {
			fmt.Fprint(&buf, "Flood Coolant")
		}
		if statusReport.AccessoryState.MistCoolant != nil && *statusReport.AccessoryState.MistCoolant {
			fmt.Fprint(&buf, "Mist Coolant")
		}
	}

	statusView.Clear()
	n, err := statusView.Write(buf.Bytes())
	if err != nil {
		logger.Error("Receiver", "err", fmt.Errorf("shell: receiver: failed to write to Grbl view: %w", err))
		return true
	}
	if n != len(buf.Bytes()) {
		logger.Error("Receiver", "err", fmt.Errorf("shell: 2receiver: short write to Grbl view: expected %d, got %d", len(buf.Bytes()), n))
		return true
	}

	return false
}

func (s *Shell) handleReset(ctx context.Context, gui *gocui.Gui) bool {
	logger := log.MustLogger(ctx)

	statusView, err := gui.View(s.statusViewName)
	if err != nil {
		logger.Error("Receiver", "err", fmt.Errorf("shell: receiver: failed to get Grbl view: %w", err))
		return true
	}
	statusView.Clear()

	gcodeView, err := gui.View(s.gcodeParserStateViewName)
	if err != nil {
		logger.Error("Receiver", "err", fmt.Errorf("shell: receiver: failed to get Grbl view: %w", err))
		return true
	}
	gcodeView.Clear()
	return false
}

//gocyclo:ignore
func (s *Shell) receiverLoop(
	ctx context.Context,
	gui *gocui.Gui,
	managerFn gocui.ManagerFunc,
	pushMessageCh chan grbl.Message,
) error {
	logger := log.MustLogger(ctx)
	for {
		message, ok := <-pushMessageCh
		if !ok {
			return fmt.Errorf("shell: receiver: push message channel closed")
		}

		var view *gocui.View
		if messageResponse, ok := message.(*grbl.MessageResponse); ok {
			var err error
			view, err = gui.View(s.grblCommandViewName)
			if err != nil {
				logger.Error("Receiver", "err", fmt.Errorf("shell: receiver: failed to get Grbl view: %w", err))
				continue
			}
			if err := messageResponse.Error(); err != nil {
				logger.Error("Response", "err", err)
			}
		} else {
			var err error
			view, err = gui.View(s.grblRealtimeCommandViewName)
			if err != nil {
				logger.Error("Receiver", "err", fmt.Errorf("shell: receiver: failed to get Grbl view: %w", err))
				continue
			}
		}

		_, isStatusReport := message.(*grbl.MessagePushStatusReport)
		if !(!s.enableStatusMessages && isStatusReport) {
			line := fmt.Sprintf("< %s\n", message)
			n, err := fmt.Fprint(view, line)
			if err != nil {
				logger.Error("Receiver", "err", fmt.Errorf("shell: receiver: failed to write to Grbl view: %w", err))
				gui.Update(managerFn)
				continue
			}
			if n != len(line) {
				logger.Error("Receiver", "err", fmt.Errorf("shell: receiver: short write to Grbl view: expected %d, got %d", len(line), n))
				gui.Update(managerFn)
				continue
			}
		}

		if messagePushGcodeState, ok := message.(*grbl.MessagePushGcodeState); ok {
			if s.receiverHandleMessagePushGcodeState(ctx, gui, messagePushGcodeState) {
				gui.Update(managerFn)
				continue
			}
		}

		if messagePushFeedback, ok := message.(*grbl.MessagePushFeedback); ok {
			if s.receiverHandleMessagePushFeedback(ctx, gui, messagePushFeedback) {
				gui.Update(managerFn)
				continue
			}
		}

		if statusReport, ok := message.(*grbl.MessagePushStatusReport); ok {
			if s.receiverHandleMessagePushStatusReport(ctx, gui, statusReport) {
				gui.Update(managerFn)
				continue
			}
		}

		if messagePushAlarm, ok := message.(*grbl.MessagePushAlarm); ok {
			if s.handleReset(ctx, gui) {
				gui.Update(managerFn)
				continue
			}
			logger.Error("Alarm", "reason", messagePushAlarm.Error())
		}

		if _, ok := message.(*grbl.MessagePushWelcome); ok {
			if s.handleReset(ctx, gui) {
				gui.Update(managerFn)
				continue
			}
		}

		gui.Update(managerFn)
	}
}

func (s *Shell) newGui(ctx context.Context) (*gocui.Gui, gocui.ManagerFunc, error) {
	gui, err := gocui.NewGui(gocui.OutputNormal)
	if err != nil {
		return nil, nil, fmt.Errorf("shell: failed to initialize gui: %w", err)
	}
	gui.Highlight = true
	gui.Cursor = true

	viewManagers := []ViewManager{}

	grblCommandViewManager := NewGrblCommandViewManager(s.grblCommandViewName)
	viewManagers = append(viewManagers, grblCommandViewManager)

	grblRealtimeCommandViewManager := NewGrblRealTimeCommandViewManager(s.grblRealtimeCommandViewName)
	viewManagers = append(viewManagers, grblRealtimeCommandViewManager)

	promptViewManager := NewPromptViewManager(s.promptViewName, "> ", s.getHandleSendCommandFn(ctx), s.getHandleResetFn(ctx))
	viewManagers = append(viewManagers, promptViewManager)

	managerFn := s.getManagerFn(
		gui,
		grblRealtimeCommandViewManager,
		grblCommandViewManager,
		promptViewManager,
	)
	gui.SetManagerFunc(managerFn)

	if err := s.setKeybindings(gui, viewManagers); err != nil {
		return nil, nil, fmt.Errorf("shell: failed to initialize key bindings: %w", err)
	}

	return gui, managerFn, nil
}

// Execute opens the connection with Grbl and executes the UI main loop; the connection is closed
// before it returns.
func (s *Shell) Execute(ctx context.Context) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("shell: execute failed: %w", err)
		}
	}()

	// Logger handler
	var gui *gocui.Gui
	logger := log.MustLogger(ctx)
	viewLogHandler := NewViewLogHandler(logger.Handler(), &gui, s.grblRealtimeCommandViewName)
	logger = slog.New(viewLogHandler)
	ctx = log.WithLogger(ctx, logger)

	// Open Grbl
	pushMessageCh, err := s.grbl.Connect(ctx)
	if err != nil {
		return
	}
	defer func() {
		err = errors.Join(err, s.grbl.Disconnect(ctx))
	}()

	// Gui
	var managerFn gocui.ManagerFunc
	gui, managerFn, err = s.newGui(ctx)
	if err != nil {
		return
	}
	defer func() {
		gui.Close()
	}()

	// Receiver
	receiverCtx, receiverCancel := context.WithCancel(ctx)
	receiverDone := make(chan struct{})
	defer func() {
		receiverCancel()
		<-receiverDone
	}()
	go func() {
		defer func() {
			close(receiverDone)
		}()
		receiverErr := s.receiverLoop(receiverCtx, gui, managerFn, pushMessageCh)
		if receiverErr != nil {
			gui.Close()
		}
		err = errors.Join(err, receiverErr)
	}()

	// Status
	statusCtx, statusCancel := context.WithCancel(ctx)
	statusDone := make(chan struct{})
	defer func() {
		statusCancel()
		<-statusDone
	}()
	go func() {
		defer func() {
			close(statusDone)
		}()
		for {
			if statusCtx.Err() != nil {
				break
			}
			if err := s.grblSendRealTimeCommand(statusCtx, gui, grbl.RealTimeCommandStatusReportQuery); err != nil {
				logger.Error("Failed to request status report query", "err", err, "%#v", fmt.Sprintf("%#v", err))
			}

			time.Sleep(200 * time.Millisecond)
		}
	}()

	// Main Loop
	if mainLoopErr := gui.MainLoop(); mainLoopErr != nil && mainLoopErr != gocui.ErrQuit {
		err = errors.Join(err, mainLoopErr)
	}

	return
}
