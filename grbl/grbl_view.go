package grbl

import (
	"github.com/jroimartin/gocui"
)

type GrblView struct {
	name string
}

func NewGrblView(name string) *GrblView {
	return &GrblView{
		name: name,
	}
}

func (p *GrblView) GetManagerFn(gui *gocui.Gui, x0, y0, x1, y1 int) func(gui *gocui.Gui) error {
	return func(gui *gocui.Gui) error {
		if view, err := gui.SetView(p.name, x0, y0, x1, y1); err != nil {
			if err != gocui.ErrUnknownView {
				return err
			}
			view.Title = "Grbl"
			view.Wrap = true
			view.Autoscroll = true
		}
		return nil
	}
}

func (p *GrblView) InitKeybindings(g *gocui.Gui) error {
	return nil
}
