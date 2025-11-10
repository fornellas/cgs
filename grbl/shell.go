package grbl

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/fornellas/slogxt/log"
	"github.com/jroimartin/gocui"
)

type ViewManager interface {
	GetManagerFn(gui *gocui.Gui, x0, y0, x1, y1 int) func(gui *gocui.Gui) error
	InitKeybindings(g *gocui.Gui) error
}

type Shell struct {
	grbl                    *Grbl
	grblViewName            string
	statusViewName          string
	feedbackMessageViewName string
	promptViewName          string
}

func NewShell(grbl *Grbl) *Shell {
	return &Shell{
		grbl:                    grbl,
		grblViewName:            "grbl",
		statusViewName:          "status",
		feedbackMessageViewName: "feedbackMessage",
		promptViewName:          "prompt",
	}
}

func (s *Shell) getManagerFn(
	gui *gocui.Gui,
	grblViewManager *GrblView,
	promptViewManoger *PromptView,
) func(*gocui.Gui) error {
	return func(*gocui.Gui) error {
		maxX, maxY := gui.Size()

		feedbackMessageHeight := 3
		promptHeight := 3
		statusWidth := 13

		grblViewManagerFn := grblViewManager.GetManagerFn(gui, 0, 0, maxX-(1+statusWidth), maxY-(1+feedbackMessageHeight+promptHeight))
		if err := grblViewManagerFn(gui); err != nil {
			return fmt.Errorf("shell: manager: Grbl view manager failed: %w", err)
		}

		if view, err := gui.SetView(s.statusViewName, maxX-statusWidth, 0, maxX-1, maxY-(1+feedbackMessageHeight+promptHeight)); err != nil {
			if err != gocui.ErrUnknownView {
				return err
			}
			view.Title = "Status"
			view.Wrap = true
			view.Autoscroll = true
		}

		if view, err := gui.SetView(s.feedbackMessageViewName, 0, maxY-6, maxX-1, maxY-(1+feedbackMessageHeight)); err != nil {
			if err != gocui.ErrUnknownView {
				return err
			}
			view.Title = "Feedback Message"
			view.Wrap = true
			view.Autoscroll = true
		}

		promptViewManagerFn := promptViewManoger.GetManagerFn(gui, 0, maxY-promptHeight, maxX-1, maxY-1)
		if err := promptViewManagerFn(gui); err != nil {
			return fmt.Errorf("shell: manager: prompt view manager failed: %w", err)
		}

		if _, err := gui.SetCurrentView(s.promptViewName); err != nil {
			return fmt.Errorf("shell: manager: failed to set current view to prompt: %w", err)
		}

		return nil
	}
}

func (s *Shell) getHandleSendBlockFn(ctx context.Context) func(gui *gocui.Gui, block string) error {
	return func(gui *gocui.Gui, block string) error {
		if err := s.grbl.SendBlock(ctx, block); err != nil {
			return fmt.Errorf("shell: handleCommand: failed to send command to Grbl: %w", err)
		}
		grblView, err := gui.View(s.grblViewName)
		if err != nil {
			return fmt.Errorf("shell: handleCommand: failed to get Grbl view: %w", err)
		}
		line := fmt.Sprintf("> %s\n", block)
		n, err := fmt.Fprint(grblView, line)
		if err != nil {
			return fmt.Errorf("shell: handleCommand: failed to write to Grbl view: %w", err)
		}
		if n != len(line) {
			return fmt.Errorf("shell: handleCommand: short write to Grbl view: expected %d, got %d", len(line), n)
		}
		return nil
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

func (s *Shell) receiverHandleMessagePushFeedback(
	ctx context.Context,
	gui *gocui.Gui,
	messagePushFeedback *MessagePushFeedback,
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
func (s *Shell) receiverHandleMessagePushStatusReport(
	ctx context.Context,
	gui *gocui.Gui,
	statusReport *MessagePushStatusReport,
) bool {
	logger := log.MustLogger(ctx)

	statusView, err := gui.View(s.statusViewName)
	if err != nil {
		logger.Error("Receiver", "err", fmt.Errorf("shell: receiver: failed to get Grbl view: %w", err))
		return true
	}

	var buf bytes.Buffer

	fmt.Fprintf(&buf, "[%s]\n", statusReport.MachineState.State)
	// TODO handle messagePushStatusReport.MachineState.SubState

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
	if wx != nil || wy != nil || wz != nil || wa != nil {
		fmt.Fprintf(&buf, "Work\n")
	}
	if wx != nil {
		fmt.Fprintf(&buf, "X:%.3f\n", *wx)
	}
	if wy != nil {
		fmt.Fprintf(&buf, "Y:%.3f\n", *wy)
	}
	if wz != nil {
		fmt.Fprintf(&buf, "Z:%.3f\n", *wz)
	}
	if wa != nil {
		fmt.Fprintf(&buf, "A:%.3f\n", *wa)
	}
	if mx != nil || my != nil || mz != nil || ma != nil {
		fmt.Fprintf(&buf, "Machine\n")
	}
	if mx != nil {
		fmt.Fprintf(&buf, "X:%.3f\n", *mx)
	}
	if my != nil {
		fmt.Fprintf(&buf, "Y:%.3f\n", *my)
	}
	if mz != nil {
		fmt.Fprintf(&buf, "Z:%.3f\n", *mz)
	}
	if ma != nil {
		fmt.Fprintf(&buf, "A:%.3f\n", *ma)
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

func (s *Shell) receiver(ctx context.Context, gui *gocui.Gui, managerFn gocui.ManagerFunc) error {
	logger := log.MustLogger(ctx)
	for {
		grblView, err := gui.View(s.grblViewName)
		if err != nil {
			logger.Error("Receiver", "err", fmt.Errorf("shell: receiver: failed to get Grbl view: %w", err))
			continue
		}

		message, err := s.grbl.Receive(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return nil
			}
			logger.Error("Receiver", "err", fmt.Errorf("shell: receive error: %w", err))
			gui.Update(managerFn)
			continue
		}

		if messagePushFeedback, ok := message.(*MessagePushFeedback); ok {
			if s.receiverHandleMessagePushFeedback(ctx, gui, messagePushFeedback) {
				gui.Update(managerFn)
				continue
			}
		}

		if statusReport, ok := message.(*MessagePushStatusReport); ok {
			if s.receiverHandleMessagePushStatusReport(ctx, gui, statusReport) {
				gui.Update(managerFn)
				continue
			}
		}

		line := fmt.Sprintf("< %s\n", message)

		n, err := fmt.Fprint(grblView, line)
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
	grblViewManager := NewGrblView(s.grblViewName)
	viewManagers = append(viewManagers, grblViewManager)
	promptViewManoger := NewPromptView(s.promptViewName, "> ", s.getHandleSendBlockFn(ctx))
	viewManagers = append(viewManagers, promptViewManoger)

	managerFn := s.getManagerFn(gui, grblViewManager, promptViewManoger)
	gui.SetManagerFunc(managerFn)

	for _, viewManager := range viewManagers {
		if err := viewManager.InitKeybindings(gui); err != nil {
			return nil, nil, fmt.Errorf("shell: failed to initialize ViewManager key bindings: %w", err)
		}
	}

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
	viewHandler := NewViewHandler(logger.Handler(), &gui, s.grblViewName)
	logger = slog.New(viewHandler)
	ctx = log.WithLogger(ctx, logger)

	// Open Grbl
	if err = s.grbl.Open(ctx); err != nil {
		return
	}
	defer func() {
		err = errors.Join(err, s.grbl.Close(ctx))
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
		err = errors.Join(err, s.receiver(receiverCtx, gui, managerFn))
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
			if err := s.grbl.SendRealTimeCommand(statusCtx, RealTimeCommandStatusReportQuery); err != nil {
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
