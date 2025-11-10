package grbl

import (
	"github.com/jroimartin/gocui"
)

type GrblCommandViewManager struct {
	name string
}

func NewGrblCommandViewManager(name string) *GrblCommandViewManager {
	return &GrblCommandViewManager{
		name: name,
	}
}

func (p *GrblCommandViewManager) GetManagerFn(gui *gocui.Gui, x0, y0, x1, y1 int) func(gui *gocui.Gui) error {
	return func(gui *gocui.Gui) error {
		if view, err := gui.SetView(p.name, x0, y0, x1, y1); err != nil {
			if err != gocui.ErrUnknownView {
				return err
			}
			view.Title = "Grbl Commands"
			view.Wrap = true
			view.Autoscroll = true
		}
		return nil
	}
}

func (p *GrblCommandViewManager) InitKeybindings(g *gocui.Gui) error {
	return nil
}
