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

	// Move Orientation
	moveOrientationDropdown := tview.NewDropDown()
	moveOrientationDropdown.SetLabel("Move Orientation")
	moveOrientationDropdown.SetOptions([]string{"Toward piece", "From piece"}, func(text string, index int) {
		// TODO
	})
	moveOrientationDropdown.SetCurrentOption(0)

	xInputField := tview.NewInputField()
	xInputField.SetFieldWidth(coordinateWidth)
	xInputField.SetLabel("X")
	xInputField.SetAcceptanceFunc(acceptFloat)
	xInputField.SetChangedFunc(func(string) {
		// TODO
	})

	yInputField := tview.NewInputField()
	yInputField.SetLabel("Y")
	yInputField.SetFieldWidth(coordinateWidth)
	yInputField.SetAcceptanceFunc(acceptFloat)
	yInputField.SetChangedFunc(func(string) {
		// TODO
	})

	zInputField := tview.NewInputField()
	zInputField.SetLabel("Z")
	zInputField.SetFieldWidth(coordinateWidth)
	zInputField.SetAcceptanceFunc(acceptFloat)
	zInputField.SetChangedFunc(func(string) {
		// TODO
	})

	feedRateInputField := tview.NewInputField()
	feedRateInputField.SetLabel("Feed rate")
	feedRateInputField.SetFieldWidth(feedWidth)
	feedRateInputField.SetAcceptanceFunc(acceptUFloat)
	feedRateInputField.SetChangedFunc(func(string) {
		// TODO
	})

	probeButton := tview.NewButton("Probe")
	probeButton.SetSelectedFunc(func() {
		// TODO
	})

	rootFlex := tview.NewFlex()
	rootFlex.SetBorder(true)
	rootFlex.SetTitle("Probe")
	rootFlex.SetDirection(tview.FlexRow)
	rootFlex.AddItem(moveOrientationDropdown, 1, 0, false)
	rootFlex.AddItem(xInputField, 1, 0, false)
	rootFlex.AddItem(yInputField, 1, 0, false)
	rootFlex.AddItem(zInputField, 1, 0, false)
	rootFlex.AddItem(feedRateInputField, 1, 0, false)
	rootFlex.AddItem(probeButton, 1, 0, false)
	sp.Flex = rootFlex

	return sp
}

func (sp *ProbePrimitive) ProcessPushMessage(ctx context.Context, pushMessage grblMod.PushMessage) {
}
