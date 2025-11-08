package grbl

import (
	"fmt"
	"io"

	"github.com/jroimartin/gocui"
)

type ViewManager interface {
	GetManagerFn(gui *gocui.Gui, x0, y0, x1, y1 int) func(gui *gocui.Gui) error
	InitKeybindings(g *gocui.Gui) error
}

type Shell struct {
	port           io.ReadWriter
	gui            *gocui.Gui
	grblViewName   string
	grblView       ViewManager
	promptViewName string
	promptView     ViewManager
}

func NewShell(port io.ReadWriter) (*Shell, error) {
	s := &Shell{}

	s.port = port

	var err error
	s.gui, err = gocui.NewGui(gocui.OutputNormal)
	if err != nil {
		return nil, err
	}
	s.gui.Highlight = true
	s.gui.Cursor = true
	s.gui.SetManagerFunc(s.manager)

	s.grblViewName = "grbl"
	s.grblView = NewGrblView(s.grblViewName)
	if err := s.grblView.InitKeybindings(s.gui); err != nil {
		return nil, err
	}

	s.promptViewName = "prompt"
	s.promptView = NewPromptView(s.promptViewName, "> ", s.handleCommand)
	if err := s.promptView.InitKeybindings(s.gui); err != nil {
		return nil, err
	}

	if err := s.setKeybindings(); err != nil {
		return nil, err
	}

	return s, nil
}

func (s *Shell) manager(gui *gocui.Gui) error {
	maxX, maxY := gui.Size()

	grblViewManagerFn := s.grblView.GetManagerFn(gui, 0, 0, maxX-1, maxY-4)
	if err := grblViewManagerFn(gui); err != nil {
		return err
	}

	promptViewManagerFn := s.promptView.GetManagerFn(gui, 0, maxY-3, maxX-1, maxY-1)
	if err := promptViewManagerFn(gui); err != nil {
		return err
	}

	if _, err := gui.SetCurrentView(s.promptViewName); err != nil {
		return err
	}

	return nil
}

func (s *Shell) handleCommand(gui *gocui.Gui, command string) error {
	grblView, err := gui.View(s.grblViewName)
	if err != nil {
		return err
	}
	fmt.Fprintf(grblView, "< %#v\n", command)
	return nil
}

func (s *Shell) handleKeyBindQuit(g *gocui.Gui, v *gocui.View) error {
	return gocui.ErrQuit
}

func (s *Shell) setKeybindings() error {
	if err := s.gui.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, s.handleKeyBindQuit); err != nil {
		return err
	}

	if err := s.grblView.InitKeybindings(s.gui); err != nil {
		return err
	}

	if err := s.promptView.InitKeybindings(s.gui); err != nil {
		return err
	}

	return nil
}

func (s *Shell) Execute() error {
	// go simulateGrblReplies(gui)

	if err := s.gui.MainLoop(); err != nil && err != gocui.ErrQuit {
		return err
	}
	return nil
}

func (s *Shell) Close() {
	s.gui.Close()
}
