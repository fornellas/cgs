package control

import (
	"context"
	"fmt"

	"github.com/rivo/tview"

	grblMod "github.com/fornellas/cgs/grbl"
)

type OverridesPrimitive struct {
	*tview.Flex
	app                 *tview.Application
	controlPrimitive    *ControlPrimitive
	feedDecr10Button    *tview.Button
	feedDecr1Button     *tview.Button
	feed100Button       *tview.Button
	feedIncr1Button     *tview.Button
	feedIncr10Button    *tview.Button
	rapid25Button       *tview.Button
	rapid50Button       *tview.Button
	rapid100Button      *tview.Button
	spindleStopButton   *tview.Button
	spindleDecr10Button *tview.Button
	spindleDecr1Button  *tview.Button
	spindle100Button    *tview.Button
	spindleIncr1Button  *tview.Button
	spindleIncr10Button *tview.Button
	coolantFloodButton  *tview.Button
	coolantMistButton   *tview.Button
}

func NewOverridesPrimitive(
	ctx context.Context,
	app *tview.Application,
	controlPrimitive *ControlPrimitive,
) *OverridesPrimitive {
	op := &OverridesPrimitive{
		app:              app,
		controlPrimitive: controlPrimitive,
	}

	// Feed Buttons
	op.feedDecr10Button = tview.NewButton("-10%")
	op.feedDecr10Button.SetSelectedFunc(func() {
		op.controlPrimitive.QueueRealTimeCommand(grblMod.RealTimeCommandFeedOverrideDecrease10)
	})
	op.feedDecr1Button = tview.NewButton("-1%")
	op.feedDecr1Button.SetSelectedFunc(func() {
		op.controlPrimitive.QueueRealTimeCommand(grblMod.RealTimeCommandFeedOverrideDecrease1)
	})
	op.feed100Button = tview.NewButton("100%")
	op.feed100Button.SetSelectedFunc(func() {
		op.controlPrimitive.QueueRealTimeCommand(grblMod.RealTimeCommandFeedOverrideSet100OfProgrammedRate)
	})
	op.feedIncr1Button = tview.NewButton("+1%")
	op.feedIncr1Button.SetSelectedFunc(func() {
		op.controlPrimitive.QueueRealTimeCommand(grblMod.RealTimeCommandFeedOverrideIncrease1)
	})
	op.feedIncr10Button = tview.NewButton("+10%")
	op.feedIncr10Button.SetSelectedFunc(func() {
		op.controlPrimitive.QueueRealTimeCommand(grblMod.RealTimeCommandFeedOverrideIncrease10)
	})

	// Feed Flex
	feedOverridesFlex := tview.NewFlex()
	feedOverridesFlex.SetBorder(true)
	feedOverridesFlex.SetTitle("Feed")
	feedOverridesFlex.SetDirection(tview.FlexColumn)
	feedOverridesFlex.AddItem(op.feedDecr10Button, 0, 1, false)
	feedOverridesFlex.AddItem(op.feedDecr1Button, 0, 1, false)
	feedOverridesFlex.AddItem(op.feed100Button, 0, 1, false)
	feedOverridesFlex.AddItem(op.feedIncr1Button, 0, 1, false)
	feedOverridesFlex.AddItem(op.feedIncr10Button, 0, 1, false)

	// Rapid Buttons
	op.rapid25Button = tview.NewButton("25%")
	op.rapid25Button.SetSelectedFunc(func() {
		op.controlPrimitive.QueueRealTimeCommand(grblMod.RealTimeCommandRapidOverrideSetTo25OfRapidRate)
	})
	op.rapid50Button = tview.NewButton("50%")
	op.rapid50Button.SetSelectedFunc(func() {
		op.controlPrimitive.QueueRealTimeCommand(grblMod.RealTimeCommandRapidOverrideSetTo50OfRapidRate)
	})
	op.rapid100Button = tview.NewButton("100%")
	op.rapid100Button.SetSelectedFunc(func() {
		op.controlPrimitive.QueueRealTimeCommand(grblMod.RealTimeCommandRapidOverrideSetTo100FullRapidRate)
	})

	// Rapid Flex
	rapidOverridesFlex := tview.NewFlex()
	rapidOverridesFlex.SetBorder(true)
	rapidOverridesFlex.SetTitle("Rapid")
	rapidOverridesFlex.SetDirection(tview.FlexColumn)
	rapidOverridesFlex.AddItem(op.rapid25Button, 0, 1, false)
	rapidOverridesFlex.AddItem(op.rapid50Button, 0, 1, false)
	rapidOverridesFlex.AddItem(op.rapid100Button, 0, 1, false)

	// Spindle Buttons
	op.spindleStopButton = tview.NewButton("Stop")
	op.spindleStopButton.SetSelectedFunc(func() {
		op.controlPrimitive.QueueRealTimeCommand(grblMod.RealTimeCommandToggleSpindleStop)
	})
	op.spindleDecr10Button = tview.NewButton("-10%")
	op.spindleDecr10Button.SetSelectedFunc(func() {
		op.controlPrimitive.QueueRealTimeCommand(grblMod.RealTimeCommandSpindleSpeedOverrideDecrease10)
	})
	op.spindleDecr1Button = tview.NewButton("-1%")
	op.spindleDecr1Button.SetSelectedFunc(func() {
		op.controlPrimitive.QueueRealTimeCommand(grblMod.RealTimeCommandSpindleSpeedOverrideDecrease1)
	})
	op.spindle100Button = tview.NewButton("100%")
	op.spindle100Button.SetSelectedFunc(func() {
		op.controlPrimitive.QueueRealTimeCommand(grblMod.RealTimeCommandSpindleSpeedOverrideSet100OfProgrammedSpindleSpeed)
	})
	op.spindleIncr1Button = tview.NewButton("+1%")
	op.spindleIncr1Button.SetSelectedFunc(func() {
		op.controlPrimitive.QueueRealTimeCommand(grblMod.RealTimeCommandSpindleSpeedOverrideIncrease1)
	})
	op.spindleIncr10Button = tview.NewButton("+10%")
	op.spindleIncr10Button.SetSelectedFunc(func() {
		op.controlPrimitive.QueueRealTimeCommand(grblMod.RealTimeCommandSpindleSpeedOverrideIncrease10)
	})

	// Spindle Flex
	spindleOverridesFlex := tview.NewFlex()
	spindleOverridesFlex.SetBorder(true)
	spindleOverridesFlex.SetTitle("Spindle")
	spindleOverridesFlex.SetDirection(tview.FlexColumn)
	spindleOverridesFlex.AddItem(op.spindleStopButton, 0, 1, false)
	spindleOverridesFlex.AddItem(op.spindleDecr10Button, 0, 1, false)
	spindleOverridesFlex.AddItem(op.spindleDecr1Button, 0, 1, false)
	spindleOverridesFlex.AddItem(op.spindle100Button, 0, 1, false)
	spindleOverridesFlex.AddItem(op.spindleIncr1Button, 0, 1, false)
	spindleOverridesFlex.AddItem(op.spindleIncr10Button, 0, 1, false)

	// Coolant Buttons
	op.coolantFloodButton = tview.NewButton("Flood").SetSelectedFunc(func() {
		op.controlPrimitive.QueueRealTimeCommand(grblMod.RealTimeCommandToggleFloodCoolant)
	})
	op.coolantMistButton = tview.NewButton("Mist").SetSelectedFunc(func() {
		op.controlPrimitive.QueueRealTimeCommand(grblMod.RealTimeCommandToggleMistCoolant)
	})

	// Coolant Flex
	coolantOverridesFlex := tview.NewFlex()
	coolantOverridesFlex.SetBorder(true)
	coolantOverridesFlex.SetTitle("Coolant")
	coolantOverridesFlex.SetDirection(tview.FlexColumn)
	coolantOverridesFlex.AddItem(op.coolantFloodButton, 0, 1, false)
	coolantOverridesFlex.AddItem(op.coolantMistButton, 0, 1, false)

	// Overrides Flex
	overridesFlex := tview.NewFlex()
	overridesFlex.SetBorder(true)
	overridesFlex.SetTitle("Overrides")
	overridesFlex.SetDirection(tview.FlexRow)
	overridesFlex.AddItem(feedOverridesFlex, 0, 1, false)
	overridesFlex.AddItem(rapidOverridesFlex, 0, 1, false)
	overridesFlex.AddItem(spindleOverridesFlex, 0, 1, false)
	overridesFlex.AddItem(coolantOverridesFlex, 0, 1, false)
	op.Flex = overridesFlex

	return op
}

func (op *OverridesPrimitive) processMessagePushStatusReport(
	ctx context.Context,
	messagePushStatusReport *grblMod.MessagePushStatusReport,
) {
	op.app.QueueUpdateDraw(func() {
		switch messagePushStatusReport.MachineState.State {
		case "Idle":
			op.feedDecr10Button.SetDisabled(false)
			op.feedDecr1Button.SetDisabled(false)
			op.feed100Button.SetDisabled(false)
			op.feedIncr1Button.SetDisabled(false)
			op.feedIncr10Button.SetDisabled(false)
			op.rapid25Button.SetDisabled(false)
			op.rapid50Button.SetDisabled(false)
			op.rapid100Button.SetDisabled(false)
			op.spindleStopButton.SetDisabled(true)
			op.spindleDecr10Button.SetDisabled(false)
			op.spindleDecr1Button.SetDisabled(false)
			op.spindle100Button.SetDisabled(false)
			op.spindleIncr1Button.SetDisabled(false)
			op.spindleIncr10Button.SetDisabled(false)
			op.coolantFloodButton.SetDisabled(false)
			op.coolantMistButton.SetDisabled(false)
		case "Run":
			op.feedDecr10Button.SetDisabled(false)
			op.feedDecr1Button.SetDisabled(false)
			op.feed100Button.SetDisabled(false)
			op.feedIncr1Button.SetDisabled(false)
			op.feedIncr10Button.SetDisabled(false)
			op.rapid25Button.SetDisabled(false)
			op.rapid50Button.SetDisabled(false)
			op.rapid100Button.SetDisabled(false)
			op.spindleStopButton.SetDisabled(true)
			op.spindleDecr10Button.SetDisabled(false)
			op.spindleDecr1Button.SetDisabled(false)
			op.spindle100Button.SetDisabled(false)
			op.spindleIncr1Button.SetDisabled(false)
			op.spindleIncr10Button.SetDisabled(false)
			op.coolantFloodButton.SetDisabled(false)
			op.coolantMistButton.SetDisabled(false)
		case "Hold":
			op.feedDecr10Button.SetDisabled(false)
			op.feedDecr1Button.SetDisabled(false)
			op.feed100Button.SetDisabled(false)
			op.feedIncr1Button.SetDisabled(false)
			op.feedIncr10Button.SetDisabled(false)
			op.rapid25Button.SetDisabled(false)
			op.rapid50Button.SetDisabled(false)
			op.rapid100Button.SetDisabled(false)
			op.spindleStopButton.SetDisabled(false)
			op.spindleDecr10Button.SetDisabled(false)
			op.spindleDecr1Button.SetDisabled(false)
			op.spindle100Button.SetDisabled(false)
			op.spindleIncr1Button.SetDisabled(false)
			op.spindleIncr10Button.SetDisabled(false)
			op.coolantFloodButton.SetDisabled(false)
			op.coolantMistButton.SetDisabled(false)
		case "Jog":
			op.feedDecr10Button.SetDisabled(false)
			op.feedDecr1Button.SetDisabled(false)
			op.feed100Button.SetDisabled(false)
			op.feedIncr1Button.SetDisabled(false)
			op.feedIncr10Button.SetDisabled(false)
			op.rapid25Button.SetDisabled(false)
			op.rapid50Button.SetDisabled(false)
			op.rapid100Button.SetDisabled(false)
			op.spindleStopButton.SetDisabled(true)
			op.spindleDecr10Button.SetDisabled(false)
			op.spindleDecr1Button.SetDisabled(false)
			op.spindle100Button.SetDisabled(false)
			op.spindleIncr1Button.SetDisabled(false)
			op.spindleIncr10Button.SetDisabled(false)
			op.coolantFloodButton.SetDisabled(true)
			op.coolantMistButton.SetDisabled(true)
		case "Alarm":
			op.feedDecr10Button.SetDisabled(true)
			op.feedDecr1Button.SetDisabled(true)
			op.feed100Button.SetDisabled(true)
			op.feedIncr1Button.SetDisabled(true)
			op.feedIncr10Button.SetDisabled(true)
			op.rapid25Button.SetDisabled(true)
			op.rapid50Button.SetDisabled(true)
			op.rapid100Button.SetDisabled(true)
			op.spindleStopButton.SetDisabled(true)
			op.spindleDecr10Button.SetDisabled(true)
			op.spindleDecr1Button.SetDisabled(true)
			op.spindle100Button.SetDisabled(true)
			op.spindleIncr1Button.SetDisabled(true)
			op.spindleIncr10Button.SetDisabled(true)
			op.coolantFloodButton.SetDisabled(true)
			op.coolantMistButton.SetDisabled(true)
		case "Door":
			op.feedDecr10Button.SetDisabled(false)
			op.feedDecr1Button.SetDisabled(false)
			op.feed100Button.SetDisabled(false)
			op.feedIncr1Button.SetDisabled(false)
			op.feedIncr10Button.SetDisabled(false)
			op.rapid25Button.SetDisabled(false)
			op.rapid50Button.SetDisabled(false)
			op.rapid100Button.SetDisabled(false)
			op.spindleStopButton.SetDisabled(true)
			op.spindleDecr10Button.SetDisabled(false)
			op.spindleDecr1Button.SetDisabled(false)
			op.spindle100Button.SetDisabled(false)
			op.spindleIncr1Button.SetDisabled(false)
			op.spindleIncr10Button.SetDisabled(false)
			op.coolantFloodButton.SetDisabled(true)
			op.coolantMistButton.SetDisabled(true)
		case "Check":
			op.feedDecr10Button.SetDisabled(false)
			op.feedDecr1Button.SetDisabled(false)
			op.feed100Button.SetDisabled(false)
			op.feedIncr1Button.SetDisabled(false)
			op.feedIncr10Button.SetDisabled(false)
			op.rapid25Button.SetDisabled(false)
			op.rapid50Button.SetDisabled(false)
			op.rapid100Button.SetDisabled(false)
			op.spindleStopButton.SetDisabled(true)
			op.spindleDecr10Button.SetDisabled(false)
			op.spindleDecr1Button.SetDisabled(false)
			op.spindle100Button.SetDisabled(false)
			op.spindleIncr1Button.SetDisabled(false)
			op.spindleIncr10Button.SetDisabled(false)
			op.coolantFloodButton.SetDisabled(true)
			op.coolantMistButton.SetDisabled(true)
		case "Home":
			op.feedDecr10Button.SetDisabled(false)
			op.feedDecr1Button.SetDisabled(false)
			op.feed100Button.SetDisabled(false)
			op.feedIncr1Button.SetDisabled(false)
			op.feedIncr10Button.SetDisabled(false)
			op.rapid25Button.SetDisabled(false)
			op.rapid50Button.SetDisabled(false)
			op.rapid100Button.SetDisabled(false)
			op.spindleStopButton.SetDisabled(true)
			op.spindleDecr10Button.SetDisabled(false)
			op.spindleDecr1Button.SetDisabled(false)
			op.spindle100Button.SetDisabled(false)
			op.spindleIncr1Button.SetDisabled(false)
			op.spindleIncr10Button.SetDisabled(false)
			op.coolantFloodButton.SetDisabled(true)
			op.coolantMistButton.SetDisabled(true)
		case "Sleep":
			op.feedDecr10Button.SetDisabled(false)
			op.feedDecr1Button.SetDisabled(false)
			op.feed100Button.SetDisabled(false)
			op.feedIncr1Button.SetDisabled(false)
			op.feedIncr10Button.SetDisabled(false)
			op.rapid25Button.SetDisabled(false)
			op.rapid50Button.SetDisabled(false)
			op.rapid100Button.SetDisabled(false)
			op.spindleStopButton.SetDisabled(true)
			op.spindleDecr10Button.SetDisabled(false)
			op.spindleDecr1Button.SetDisabled(false)
			op.spindle100Button.SetDisabled(false)
			op.spindleIncr1Button.SetDisabled(false)
			op.spindleIncr10Button.SetDisabled(false)
			op.coolantFloodButton.SetDisabled(true)
			op.coolantMistButton.SetDisabled(true)
		default:
			panic(fmt.Sprintf("unknown machine state: %#v", messagePushStatusReport.MachineState.State))
		}
	})
}

func (op *OverridesPrimitive) ProcessMessage(ctx context.Context, message grblMod.Message) {
	if messagePushStatusReport, ok := message.(*grblMod.MessagePushStatusReport); ok {
		op.processMessagePushStatusReport(ctx, messagePushStatusReport)
		return
	}
}
