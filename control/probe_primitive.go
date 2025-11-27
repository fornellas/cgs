package control

import (
	"context"

	"github.com/rivo/tview"

	grblMod "github.com/fornellas/cgs/grbl"
)

type ProbePrimitive struct {
	*tview.Flex
	app              *tview.Application
	controlPrimitive *ControlPrimitive
}

func NewProbePrimitive(
	ctx context.Context,
	app *tview.Application,
	controlPrimitive *ControlPrimitive,
) *ProbePrimitive {
	sp := &ProbePrimitive{
		app:              app,
		controlPrimitive: controlPrimitive,
	}

	streamRootFlex := tview.NewFlex()
	streamRootFlex.SetBorder(true)
	streamRootFlex.SetTitle("Probe")
	streamRootFlex.SetDirection(tview.FlexRow)
	sp.Flex = streamRootFlex

	return sp
}

func (sp *ProbePrimitive) ProcessPushMessage(ctx context.Context, pushMessage grblMod.PushMessage) {
}
