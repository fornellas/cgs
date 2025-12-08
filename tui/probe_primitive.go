package tui

import (
	"bytes"
	"context"
	"fmt"

	iFmt "github.com/fornellas/cgs/internal/fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	grblMod "github.com/fornellas/cgs/grbl"
)

var probeTowardPieceText = fmt.Sprintf("Toward piece[%s]G38.2[-]", gcodeColor)
var probeFromPieceText = fmt.Sprintf("From piece[%s]G38.4[-]", gcodeColor)

type ProbePrimitive struct {
	*tview.Flex
	app              *tview.Application
	controlPrimitive *ControlPrimitive

	straightMoveOrientationDropdown *tview.DropDown
	straightXInputField             *tview.InputField
	straightYInputField             *tview.InputField
	straightZInputField             *tview.InputField
	straightFeedRateInputField      *tview.InputField
	straightOvershootTextView       *tview.TextView
	straightProbeButton             *tview.Button
	straightFlex                    *tview.Flex
	statusTextView                  *tview.TextView
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

	// TODO set disabled state

	return pp
}
func (pp *ProbePrimitive) newStraight() {
	// Move Orientation
	straightMoveOrientationDropdown := tview.NewDropDown()
	straightMoveOrientationDropdown.SetLabel("Move Orientation:")
	straightMoveOrientationDropdown.SetOptions([]string{probeTowardPieceText, probeFromPieceText}, func(text string, index int) {
		// TODO
	})
	straightMoveOrientationDropdown.SetCurrentOption(0)
	pp.straightMoveOrientationDropdown = straightMoveOrientationDropdown

	// x
	straightXInputField := tview.NewInputField()
	straightXInputField.SetFieldWidth(coordinateWidth)
	straightXInputField.SetLabel("X:")
	straightXInputField.SetAcceptanceFunc(acceptFloat)
	straightXInputField.SetChangedFunc(func(string) {
		// TODO
	})
	pp.straightXInputField = straightXInputField

	// y
	straightYInputField := tview.NewInputField()
	straightYInputField.SetLabel("Y:")
	straightYInputField.SetFieldWidth(coordinateWidth)
	straightYInputField.SetAcceptanceFunc(acceptFloat)
	straightYInputField.SetChangedFunc(func(string) {
		// TODO
	})
	pp.straightYInputField = straightYInputField

	// z
	straightZInputField := tview.NewInputField()
	straightZInputField.SetLabel("Z:")
	straightZInputField.SetFieldWidth(coordinateWidth)
	straightZInputField.SetAcceptanceFunc(acceptFloat)
	straightZInputField.SetChangedFunc(func(string) {
		// TODO
	})
	pp.straightZInputField = straightZInputField

	// Feed rate
	straightFeedRateInputField := tview.NewInputField()
	straightFeedRateInputField.SetLabel("Feed rate:")
	straightFeedRateInputField.SetFieldWidth(feedWidth)
	straightFeedRateInputField.SetAcceptanceFunc(acceptUFloat)
	straightFeedRateInputField.SetChangedFunc(func(string) {
		// TODO
	})
	pp.straightFeedRateInputField = straightFeedRateInputField

	// Overshoot
	straightOvershootTextView := tview.NewTextView()
	straightOvershootTextView.SetLabel("Overshoot:")
	pp.straightOvershootTextView = straightOvershootTextView

	// Probe
	straightProbeButton := tview.NewButton("Probe")
	straightProbeButton.SetSelectedFunc(func() {
		n, moveOrientation := straightMoveOrientationDropdown.GetCurrentOption()
		if n < 0 {
			return
		}
		var buf bytes.Buffer
		switch moveOrientation {
		case probeTowardPieceText:
			fmt.Fprintf(&buf, "G38.2")
		case probeFromPieceText:
			fmt.Fprintf(&buf, "G38.4")
		default:
			panic(fmt.Sprintf("bug: unknown move orientation option: %#v", moveOrientation))
		}
		if x := straightXInputField.GetText(); x != "" {
			fmt.Fprintf(&buf, "X%s", x)
		}
		if y := straightYInputField.GetText(); y != "" {
			fmt.Fprintf(&buf, "Y%s", y)
		}
		if z := straightZInputField.GetText(); z != "" {
			fmt.Fprintf(&buf, "Z%s", z)
		}

		feedRate := straightFeedRateInputField.GetText()
		if feedRate == "" {
			return
		}
		fmt.Fprintf(&buf, "F%s", feedRate)

		pp.controlPrimitive.QueueCommandIgnoreResponse(buf.String())
	})
	pp.straightProbeButton = straightProbeButton

	// Status
	pp.statusTextView = tview.NewTextView()
	pp.statusTextView.SetDynamicColors(true)

	// Straight
	straightFlex := tview.NewFlex()
	straightFlex.SetBorder(true)
	straightFlex.SetTitle("Straight")
	straightFlex.SetDirection(tview.FlexRow)
	straightFlex.AddItem(straightMoveOrientationDropdown, 1, 0, false)
	straightFlex.AddItem(straightXInputField, 1, 0, false)
	straightFlex.AddItem(straightYInputField, 1, 0, false)
	straightFlex.AddItem(straightZInputField, 1, 0, false)
	straightFlex.AddItem(straightFeedRateInputField, 1, 0, false)
	straightFlex.AddItem(straightOvershootTextView, 1, 0, false)
	straightFlex.AddItem(straightProbeButton, 3, 0, false)
	straightFlex.AddItem(pp.statusTextView, 1, 0, false)
	// TODO probe result
	pp.straightFlex = straightFlex
}

func (pp *ProbePrimitive) processPushMessage(pushMessage grblMod.PushMessage) {
	// TODO set last probing cycle status from g-code param message
	// TODO reset last probing cycle status on welcome

	pp.app.QueueUpdateDraw(func() {
		if gcodeParamPushMessage, ok := pushMessage.(*grblMod.GcodeParamPushMessage); ok {
			if gcodeParamPushMessage.GcodeParameters.Probe != nil {
				if gcodeParamPushMessage.GcodeParameters.Probe.Successful {
					fmt.Fprintf(
						pp.statusTextView,
						"[%s]Successful[-] X:%s Y:%s Z:%s",
						tcell.ColorGreen,
						iFmt.SprintFloat(gcodeParamPushMessage.GcodeParameters.Probe.Coordinates.X, 4),
						iFmt.SprintFloat(gcodeParamPushMessage.GcodeParameters.Probe.Coordinates.Y, 4),
						iFmt.SprintFloat(gcodeParamPushMessage.GcodeParameters.Probe.Coordinates.Z, 4),
					)
				} else {
					fmt.Fprintf(pp.statusTextView, "[%s]Failed[-]", tcell.ColorRed)
				}
			} else {
				pp.statusTextView.SetText("")
			}
		}
	})
}

func (pp *ProbePrimitive) Worker(
	ctx context.Context,
	pushMessageCh <-chan grblMod.PushMessage,
	trackedStateCh <-chan *TrackedState,
) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case pushMessage, ok := <-pushMessageCh:
			if !ok {
				return fmt.Errorf("push message channel closed")
			}
			pp.processPushMessage(pushMessage)

		case _, ok := <-trackedStateCh:
			if !ok {
				return fmt.Errorf("tracked state channel closed")
			}
			// TODO enable / disabled
		}
	}
}
