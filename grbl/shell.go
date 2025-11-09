package grbl

import (
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
		return nil, err
	}
	s.gui.Highlight = true
	s.gui.Cursor = true
	s.gui.SetManagerFunc(s.manager)

	s.grblViewName = "grbl"
	s.grblViewManager = NewGrblView(s.grblViewName)
	if err := s.grblViewManager.InitKeybindings(s.gui); err != nil {
		return nil, err
	}

	s.promptViewName = "prompt"
	s.promptViewManoger = NewPromptView(s.promptViewName, "> ", s.handleCommand)
	if err := s.promptViewManoger.InitKeybindings(s.gui); err != nil {
		return nil, err
	}

	if err := s.setKeybindings(); err != nil {
		return nil, err
	}

	return s, nil
}

func (s *Shell) manager(gui *gocui.Gui) error {
	maxX, maxY := gui.Size()

	grblViewManagerFn := s.grblViewManager.GetManagerFn(gui, 0, 0, maxX-1, maxY-4)
	if err := grblViewManagerFn(gui); err != nil {
		return err
	}

	promptViewManagerFn := s.promptViewManoger.GetManagerFn(gui, 0, maxY-3, maxX-1, maxY-1)
	if err := promptViewManagerFn(gui); err != nil {
		return err
	}

	if _, err := gui.SetCurrentView(s.promptViewName); err != nil {
		return err
	}

	return nil
}

func (s *Shell) handleCommand(gui *gocui.Gui, command string) error {
	if err := s.grbl.Send(command); err != nil {
		return err
	}
	grblView, err := gui.View(s.grblViewName)
	if err != nil {
		return err
	}
	fmt.Fprintf(grblView, "> %#v\n", command)
	return nil
}

func (s *Shell) handleKeyBindQuit(g *gocui.Gui, v *gocui.View) error {
	return gocui.ErrQuit
}

func (s *Shell) setKeybindings() error {
	if err := s.gui.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, s.handleKeyBindQuit); err != nil {
		return err
	}

	if err := s.grblViewManager.InitKeybindings(s.gui); err != nil {
		return err
	}

	if err := s.promptViewManoger.InitKeybindings(s.gui); err != nil {
		return err
	}

	return nil
}

func (s *Shell) receiver() error {
	for {
		message, err := s.grbl.Receive()
		if err != nil {
			return err
		}

		grblView, err := s.gui.View(s.grblViewName)
		if err != nil {
			return err
		}
		fmt.Fprintf(grblView, "< %#v\n", message)
		s.gui.Update(s.manager)
	}
}

func (s *Shell) Execute() (err error) {
	if err := s.grbl.Connect(); err != nil {
		return err
	}

	go func() {
		err = errors.Join(err, s.receiver())
	}()

	if mainLoopErr := s.gui.MainLoop(); mainLoopErr != nil && mainLoopErr != gocui.ErrQuit {
		return errors.Join(err, mainLoopErr)
	}
	return nil
}

func (s *Shell) Close() {
	// TODO stop receiver
	s.gui.Close()
}
