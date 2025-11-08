package grbl

import (
	"errors"
	"fmt"
	"strings"

	"github.com/jroimartin/gocui"
)

type PromptView struct {
	name, prompt string
	enterFn      func(gui *gocui.Gui, command string) error
}

func NewPromptView(name, prompt string, enterFn func(gui *gocui.Gui, command string) error) *PromptView {
	return &PromptView{
		name:    name,
		prompt:  prompt,
		enterFn: enterFn,
	}
}

//gocyclo:ignore
func (p *PromptView) editorFn(view *gocui.View, key gocui.Key, ch rune, mod gocui.Modifier) {
	switch {
	case ch != 0 && mod == 0:
		view.EditWrite(ch)
	case key == gocui.KeySpace:
		view.EditWrite(' ')
	case key == gocui.KeyBackspace || key == gocui.KeyBackspace2:
		if cx, _ := view.Cursor(); cx > len(p.prompt) {
			view.EditDelete(true)
		}
	case key == gocui.KeyDelete:
		view.EditDelete(false)
	case key == gocui.KeyInsert:
		view.Overwrite = !view.Overwrite
	case key == gocui.KeyArrowLeft || key == gocui.KeyCtrlB:
		if cx, _ := view.Cursor(); cx > len(p.prompt) {
			view.MoveCursor(-1, 0, false)
		}
	case key == gocui.KeyArrowRight || key == gocui.KeyCtrlF:
		view.MoveCursor(1, 0, false)
	case key == gocui.KeyHome || key == gocui.KeyCtrlA:
		if err := view.SetCursor(len(p.prompt), 0); err != nil {
			panic(err)
		}
		if err := view.SetOrigin(0, 0); err != nil {
			panic(err)
		}
	case key == gocui.KeyEnd || key == gocui.KeyCtrlE:
		line := strings.TrimSuffix(view.Buffer(), "\n")
		if err := view.SetCursor(len(line), 0); err != nil {
			panic(err)
		}
	}
}

func (p *PromptView) GetManagerFn(gui *gocui.Gui, x0, y0, x1, y1 int) func(gui *gocui.Gui) error {
	return func(gui *gocui.Gui) error {
		if view, err := gui.SetView(p.name, x0, y0, x1, y1); err != nil {
			if err != gocui.ErrUnknownView {
				return err
			}
			view.Editable = true
			view.Editor = gocui.EditorFunc(p.editorFn)
			view.Wrap = false
			fmt.Fprint(view, p.prompt)
			view.SetCursor(len(p.prompt), 0)
			return nil
		}
		return nil
	}
}

func (p *PromptView) handleKeyBindEnter(gui *gocui.Gui, view *gocui.View) (err error) {
	buff := view.Buffer()

	command := ""
	if len(buff) > len(p.prompt) {
		command = strings.TrimSuffix(buff[len(p.prompt):], "\n")
	}

	if len(command) == 0 {
		return nil
	}

	if enterFnErr := p.enterFn(gui, command); enterFnErr != nil {
		err = errors.Join(err, enterFnErr)
	}

	view.Clear()

	fmt.Fprint(view, p.prompt)

	if setCursorErr := view.SetCursor(len(p.prompt), 0); setCursorErr != nil {
		err = errors.Join(err, setCursorErr)
	}

	if setOriginErr := view.SetOrigin(0, 0); setOriginErr != nil {
		err = errors.Join(err, setOriginErr)
	}

	return err
}

func (p *PromptView) InitKeybindings(gui *gocui.Gui) error {
	// Enter
	if err := gui.SetKeybinding(p.name, gocui.KeyEnter, gocui.ModNone, p.handleKeyBindEnter); err != nil {
		return err
	}

	return nil
}
