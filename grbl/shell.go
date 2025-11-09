package grbl

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/fornellas/slogxt/log"
	"github.com/jroimartin/gocui"
)

type ViewManager interface {
	GetManagerFn(gui *gocui.Gui, x0, y0, x1, y1 int) func(gui *gocui.Gui) error
	InitKeybindings(g *gocui.Gui) error
}

type Shell struct {
	grbl           *Grbl
	grblViewName   string
	promptViewName string
}

func NewShell(grbl *Grbl) *Shell {
	return &Shell{
		grbl:           grbl,
		grblViewName:   "grbl",
		promptViewName: "prompt",
	}
}

func (s *Shell) getManagerFn(
	gui *gocui.Gui,
	grblViewManager *GrblView,
	promptViewManoger *PromptView,
) func(*gocui.Gui) error {
	return func(*gocui.Gui) error {
		maxX, maxY := gui.Size()

		grblViewManagerFn := grblViewManager.GetManagerFn(gui, 0, 0, maxX-1, maxY-4)
		if err := grblViewManagerFn(gui); err != nil {
			return fmt.Errorf("shell: manager: Grbl view manager failed: %w", err)
		}

		promptViewManagerFn := promptViewManoger.GetManagerFn(gui, 0, maxY-3, maxX-1, maxY-1)
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
		line := fmt.Sprintf("> %#v\n", block)
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

func (s *Shell) receiver(ctx context.Context, gui *gocui.Gui, managerFn gocui.ManagerFunc) error {
	for {
		message, err := s.grbl.Receive(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return nil
			}
			return fmt.Errorf("shell: receive error: %w (%#v)", err, err)
		}

		grblView, err := gui.View(s.grblViewName)
		if err != nil {
			return fmt.Errorf("shell: receiver: failed to get Grbl view: %w", err)
		}

		line := fmt.Sprintf("< %#v\n", message)

		n, err := fmt.Fprint(grblView, line)
		if err != nil {
			return fmt.Errorf("shell: receiver: failed to write to Grbl view: %w", err)
		}
		if n != len(line) {
			return fmt.Errorf("shell: handleCommand: short write to Grbl view: expected %d, got %d", len(line), n)
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

	// Main Loop
	if mainLoopErr := gui.MainLoop(); mainLoopErr != nil && mainLoopErr != gocui.ErrQuit {
		err = errors.Join(err, mainLoopErr)
	}

	return
}
