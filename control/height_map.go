package control

import (
	"context"

	"github.com/rivo/tview"

	grblMod "github.com/fornellas/cgs/grbl"
)

type HeightMapPrimitive struct {
	*tview.Flex
	app              *tview.Application
	controlPrimitive *ControlPrimitive
}

func NewHeightMapPrimitive(
	ctx context.Context,
	app *tview.Application,
	controlPrimitive *ControlPrimitive,
) *HeightMapPrimitive {
	sp := &HeightMapPrimitive{
		app:              app,
		controlPrimitive: controlPrimitive,
	}

	// Parameters
	// x0
	// y0
	// x1
	// y1
	// maxDistance

	rootFlex := tview.NewFlex()
	rootFlex.SetBorder(true)
	rootFlex.SetTitle("Height Map")
	rootFlex.SetDirection(tview.FlexRow)
	sp.Flex = rootFlex

	return sp
}

func (hm *HeightMapPrimitive) ProcessPushMessage(ctx context.Context, pushMessage grblMod.PushMessage) {
}
