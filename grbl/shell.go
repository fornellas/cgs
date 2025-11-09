package grbl

import (
	"context"
	"errors"
	"fmt"

	"github.com/jroimartin/gocui"
)

type ViewManager interface {
	GetManagerFn(gui *gocui.Gui, x0, y0, x1, y1 int) func(gui *gocui.Gui) error
	InitKeybindings(g *gocui.Gui) error
}

type Shell struct {
	grbl              *Grbl
	gui               *gocui.Gui
	grblViewName      string
	grblViewManager   ViewManager
	promptViewName    string
	promptViewManoger ViewManager
}

func NewShell(grbl *Grbl) (*Shell, error) {
	s := &Shell{}

	s.grbl = grbl

	var err error
	s.gui, err = gocui.NewGui(gocui.OutputNormal)
	if err != nil {
		return nil, fmt.Errorf("shell: failed to initialize gui: %w", err)
	}
	s.gui.Highlight = true
	s.gui.Cursor = true
	s.gui.SetManagerFunc(s.manager)

	s.grblViewName = "grbl"
	s.grblViewManager = NewGrblView(s.grblViewName)
	if err := s.grblViewManager.InitKeybindings(s.gui); err != nil {
		return nil, fmt.Errorf("shell: failed to initialize Grbl view key bindings: %w", err)
	}

	s.promptViewName = "prompt"
	s.promptViewManoger = NewPromptView(s.promptViewName, "> ", s.handleSendCommand)
	if err := s.promptViewManoger.InitKeybindings(s.gui); err != nil {
		return nil, fmt.Errorf("shell: failed to initialize prompt view key bindings: %w", err)
	}

	if err := s.setKeybindings(); err != nil {
		return nil, fmt.Errorf("shell: failed to initialize key bindings: %w", err)
	}

	return s, nil
}

func (s *Shell) manager(gui *gocui.Gui) error {
	maxX, maxY := gui.Size()

	grblViewManagerFn := s.grblViewManager.GetManagerFn(gui, 0, 0, maxX-1, maxY-4)
	if err := grblViewManagerFn(gui); err != nil {
		return fmt.Errorf("shell: manager: Grbl view manager failed: %w", err)
	}

	promptViewManagerFn := s.promptViewManoger.GetManagerFn(gui, 0, maxY-3, maxX-1, maxY-1)
	if err := promptViewManagerFn(gui); err != nil {
		return fmt.Errorf("shell: manager: prompt view manager failed: %w", err)
	}

	if _, err := gui.SetCurrentView(s.promptViewName); err != nil {
		return fmt.Errorf("shell: manager: failed to set current view to prompt: %w", err)
	}

	return nil
}

func (s *Shell) handleSendCommand(gui *gocui.Gui, command string) error {
	if err := s.grbl.Send(command); err != nil {
		return fmt.Errorf("shell: handleCommand: failed to send command to Grbl: %w", err)
	}
	grblView, err := gui.View(s.grblViewName)
	if err != nil {
		return fmt.Errorf("shell: handleCommand: failed to get Grbl view: %w", err)
	}
	line := fmt.Sprintf("> %#v\n", command)
	n, err := fmt.Fprint(grblView, line)
	if err != nil {
		return fmt.Errorf("shell: handleCommand: failed to write to Grbl view: %w", err)
	}
	if n != len(line) {
		return fmt.Errorf("shell: handleCommand: short write to Grbl view: expected %d, got %d", len(line), n)
	}
	return nil
}

func (s *Shell) handleKeyBindQuit(g *gocui.Gui, v *gocui.View) error {
	return gocui.ErrQuit
}

func (s *Shell) setKeybindings() error {
	if err := s.gui.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, s.handleKeyBindQuit); err != nil {
		return fmt.Errorf("shell: failed so set keybinding: %w", err)
	}

	if err := s.grblViewManager.InitKeybindings(s.gui); err != nil {
		return fmt.Errorf("shell: failed so set keybinding: %w", err)
	}

	if err := s.promptViewManoger.InitKeybindings(s.gui); err != nil {
		return fmt.Errorf("shell: failed so set keybinding: %w", err)
	}

	return nil
}

func (s *Shell) receiver(ctx context.Context) error {
	for {
		message, err := s.grbl.Receive(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return nil
			}
			return fmt.Errorf("shell: receive error: %w (%#v)", err, err)
		}

		grblView, err := s.gui.View(s.grblViewName)
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
		s.gui.Update(s.manager)
	}
}

// Execute opens the connection with Grbl and executes the UI main loop; the connection is closed
// before it returns.
func (s *Shell) Execute(ctx context.Context) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("shell: execute failed: %w", err)
		}
	}()

	if err = s.grbl.Open(); err != nil {
		return
	}
	defer func() { err = errors.Join(err, s.grbl.Close()) }()

	receiverCtx, receiverCancel := context.WithCancel(ctx)
	defer receiverCancel()
	go func() { err = errors.Join(err, s.receiver(receiverCtx)) }()

	if mainLoopErr := s.gui.MainLoop(); mainLoopErr != nil && mainLoopErr != gocui.ErrQuit {
		err = errors.Join(err, mainLoopErr)
	}

	return
}

// Close must be called after a successful initialization and when the Shell is not needed anymore.
func (s *Shell) Close() (err error) {
	s.gui.Close()
	return nil
}
