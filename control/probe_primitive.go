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

	straightMoveOrientationDropdown *tview.DropDown
	straightXInputField             *tview.InputField
	straightYInputField             *tview.InputField
	straightZInputField             *tview.InputField
	straightFeedRateInputField      *tview.InputField
	straightProbeButton             *tview.Button
	straightFlex                    *tview.Flex
}

func NewProbePrimitive(
	ctx context.Context,
	app *tview.Application,
	controlPrimitive *ControlPrimitive,
) *ProbePrimitive {
	pp := &ProbePrimitive{
		app:              app,
		controlPrimitive: controlPrimitive,
	}

	pp.newStraight()

	// Angle
	anfleFlex := tview.NewFlex()
	anfleFlex.SetBorder(true)
	anfleFlex.SetTitle("Angle")
	anfleFlex.SetDirection(tview.FlexRow)
	// TODO axis probe positions (first, second)
	// TODO probe axis & direction (x-,x+,y-,y+)
	// TODO feed rate
	// TODO result: error | angle

	rootFlex := tview.NewFlex()
	rootFlex.SetBorder(true)
	rootFlex.SetTitle("Probe")
	rootFlex.SetDirection(tview.FlexRow)
	rootFlex.AddItem(pp.straightFlex, 0, 1, false)
	rootFlex.AddItem(anfleFlex, 0, 1, false)
	pp.Flex = rootFlex

	return pp
}
func (pp *ProbePrimitive) newStraight() {
	// Move Orientation
	straightMoveOrientationDropdown := tview.NewDropDown()
	straightMoveOrientationDropdown.SetLabel("Move Orientation ")
	straightMoveOrientationDropdown.SetOptions([]string{"Toward piece[lightblue]G38.2[-]", "From piece[lightblue]G38.4[-]"}, func(text string, index int) {
		// TODO
	})
	straightMoveOrientationDropdown.SetCurrentOption(0)
	pp.straightMoveOrientationDropdown = straightMoveOrientationDropdown

	// x
	straightXInputField := tview.NewInputField()
	straightXInputField.SetFieldWidth(coordinateWidth)
	straightXInputField.SetLabel("X ")
	straightXInputField.SetAcceptanceFunc(acceptFloat)
	straightXInputField.SetChangedFunc(func(string) {
		// TODO
	})
	pp.straightXInputField = straightXInputField

	// y
	straightYInputField := tview.NewInputField()
	straightYInputField.SetLabel("Y ")
	straightYInputField.SetFieldWidth(coordinateWidth)
	straightYInputField.SetAcceptanceFunc(acceptFloat)
	straightYInputField.SetChangedFunc(func(string) {
		// TODO
	})
	pp.straightYInputField = straightYInputField

	// z
	straightZInputField := tview.NewInputField()
	straightZInputField.SetLabel("Z ")
	straightZInputField.SetFieldWidth(coordinateWidth)
	straightZInputField.SetAcceptanceFunc(acceptFloat)
	straightZInputField.SetChangedFunc(func(string) {
		// TODO
	})
	pp.straightZInputField = straightZInputField

	// Feed rate
	straightFeedRateInputField := tview.NewInputField()
	straightFeedRateInputField.SetLabel("Feed rate ")
	straightFeedRateInputField.SetFieldWidth(feedWidth)
	straightFeedRateInputField.SetAcceptanceFunc(acceptUFloat)
	straightFeedRateInputField.SetChangedFunc(func(string) {
		// TODO
	})
	pp.straightFeedRateInputField = straightFeedRateInputField

	// Probe
	straightProbeButton := tview.NewButton("Probe")
	straightProbeButton.SetSelectedFunc(func() {
		// TODO
	})
	pp.straightProbeButton = straightProbeButton

	// Probe
	probeFlex := tview.NewFlex()
	probeFlex.SetDirection(tview.FlexRow)
	probeFlex.AddItem(straightMoveOrientationDropdown, 1, 0, false)
	probeFlex.AddItem(straightXInputField, 1, 0, false)
	probeFlex.AddItem(straightYInputField, 1, 0, false)
	probeFlex.AddItem(straightZInputField, 1, 0, false)
	probeFlex.AddItem(straightFeedRateInputField, 1, 0, false)
	probeFlex.AddItem(straightProbeButton, 3, 0, false)
	// TODO probe result

	// Set
	setFlex := tview.NewFlex()
	setFlex.SetBorder(true)
	setFlex.SetTitle("Set")
	// TODO add buttons to G92 (Coordinate System Offset) - X, Y, Z
	// TODO add buttons to G92.1 (Reset G92 Offsets)
	// TODO add buttons to G10 L2 P[1-9] (Set Coordinate System, By Offsets) - P, X, Y, Z
	// TODO add buttons to G10 L20 P[1-9] (Set Coordinate System, From Current Position)

	straightFlex := tview.NewFlex()
	straightFlex.SetBorder(true)
	straightFlex.SetTitle("Straight")
	straightFlex.SetBorderPadding(1, 1, 1, 1)
	straightFlex.SetDirection(tview.FlexColumn)
	straightFlex.AddItem(probeFlex, 0, 1, false)
	straightFlex.AddItem(setFlex, 0, 1, false)
	pp.straightFlex = straightFlex
}

func (pp *ProbePrimitive) ProcessPushMessage(ctx context.Context, pushMessage grblMod.PushMessage) {
	// TODO set last probing cycle status from g-code param message
	// TODO reset last probing cycle status on welcome
}
