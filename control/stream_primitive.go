package control

import (
	"context"

	"github.com/rivo/tview"

	grblMod "github.com/fornellas/cgs/grbl"
)

type StreamPrimitive struct {
	*tview.Flex
	app              *tview.Application
	controlPrimitive *ControlPrimitive
}

func NewStreamPrimitive(
	ctx context.Context,
	app *tview.Application,
	controlPrimitive *ControlPrimitive,
	heightMapPrimitive *HeightMapPrimitive,
) *StreamPrimitive {
	sp := &StreamPrimitive{
		app:              app,
		controlPrimitive: controlPrimitive,
	}

	// File
	fileFlex := tview.NewFlex()
	fileFlex.SetBorder(true)
	fileFlex.SetTitle("File")

	// Rotation
	rotationFlex := tview.NewFlex()
	rotationFlex.SetBorder(true)
	rotationFlex.SetTitle("Rotation")

	streamRootFlex := tview.NewFlex()
	streamRootFlex.SetBorder(true)
	streamRootFlex.SetTitle("Stream")
	streamRootFlex.SetDirection(tview.FlexRow)
	streamRootFlex.AddItem(fileFlex, 3, 0, false)
	streamRootFlex.AddItem(heightMapPrimitive, 0, 1, false)
	streamRootFlex.AddItem(rotationFlex, 3, 0, false)
	sp.Flex = streamRootFlex

	return sp
}

func (sp *StreamPrimitive) ProcessPushMessage(ctx context.Context, pushMessage grblMod.PushMessage) {
}
