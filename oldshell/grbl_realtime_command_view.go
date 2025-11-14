package oldshell

import (
	"github.com/jroimartin/gocui"
)

type GrblRealTimeCommandViewManager struct {
	name string
}

func NewGrblRealTimeCommandViewManager(name string) *GrblRealTimeCommandViewManager {
	return &GrblRealTimeCommandViewManager{
		name: name,
	}
}

func (p *GrblRealTimeCommandViewManager) GetManagerFn(gui *gocui.Gui, x0, y0, x1, y1 int) func(gui *gocui.Gui) error {
	return func(gui *gocui.Gui) error {
		if view, err := gui.SetView(p.name, x0, y0, x1, y1); err != nil {
			if err != gocui.ErrUnknownView {
				return err
			}
			view.Title = "Grbl Real-Time Commands"
			view.Wrap = true
			view.Autoscroll = true
		}
		return nil
	}
}

func (p *GrblRealTimeCommandViewManager) InitKeybindings(g *gocui.Gui) error {
	return nil
}
