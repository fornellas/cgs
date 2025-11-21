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
	overridesPrimitive := &OverridesPrimitive{
		app:              app,
		controlPrimitive: controlPrimitive,
	}

	overridesPrimitive.feedDecr10Button = tview.NewButton("-10%").SetSelectedFunc(func() {
		overridesPrimitive.controlPrimitive.QueueRealTimeCommand(grblMod.RealTimeCommandFeedOverrideDecrease10)
	})
	overridesPrimitive.feedDecr1Button = tview.NewButton("-1%").SetSelectedFunc(func() {
		overridesPrimitive.controlPrimitive.QueueRealTimeCommand(grblMod.RealTimeCommandFeedOverrideDecrease1)
	})
	overridesPrimitive.feed100Button = tview.NewButton("100%").SetSelectedFunc(func() {
		overridesPrimitive.controlPrimitive.QueueRealTimeCommand(grblMod.RealTimeCommandFeedOverrideSet100OfProgrammedRate)
	})
	overridesPrimitive.feedIncr1Button = tview.NewButton("+1%").SetSelectedFunc(func() {
		overridesPrimitive.controlPrimitive.QueueRealTimeCommand(grblMod.RealTimeCommandFeedOverrideIncrease1)
	})
	overridesPrimitive.feedIncr10Button = tview.NewButton("+10%").SetSelectedFunc(func() {
		overridesPrimitive.controlPrimitive.QueueRealTimeCommand(grblMod.RealTimeCommandFeedOverrideIncrease10)
	})

	feedOverridesFlex := tview.NewFlex()
	feedOverridesFlex.SetBorder(true)
	feedOverridesFlex.SetTitle("Feed")
	feedOverridesFlex.SetDirection(tview.FlexColumn)
	feedOverridesFlex.AddItem(overridesPrimitive.feedDecr10Button, 0, 1, false)
	feedOverridesFlex.AddItem(overridesPrimitive.feedDecr1Button, 0, 1, false)
	feedOverridesFlex.AddItem(overridesPrimitive.feed100Button, 0, 1, false)
	feedOverridesFlex.AddItem(overridesPrimitive.feedIncr1Button, 0, 1, false)
	feedOverridesFlex.AddItem(overridesPrimitive.feedIncr10Button, 0, 1, false)

	overridesPrimitive.rapid25Button = tview.NewButton("25%").SetSelectedFunc(func() {
		overridesPrimitive.controlPrimitive.QueueRealTimeCommand(grblMod.RealTimeCommandRapidOverrideSetTo25OfRapidRate)
	})
	overridesPrimitive.rapid50Button = tview.NewButton("50%").SetSelectedFunc(func() {
		overridesPrimitive.controlPrimitive.QueueRealTimeCommand(grblMod.RealTimeCommandRapidOverrideSetTo50OfRapidRate)
	})
	overridesPrimitive.rapid100Button = tview.NewButton("100%").SetSelectedFunc(func() {
		overridesPrimitive.controlPrimitive.QueueRealTimeCommand(grblMod.RealTimeCommandRapidOverrideSetTo100FullRapidRate)
	})

	rapidOverridesFlex := tview.NewFlex()
	rapidOverridesFlex.SetBorder(true)
	rapidOverridesFlex.SetTitle("Rapid")
	rapidOverridesFlex.SetDirection(tview.FlexColumn)
	rapidOverridesFlex.AddItem(overridesPrimitive.rapid25Button, 0, 1, false)
	rapidOverridesFlex.AddItem(overridesPrimitive.rapid50Button, 0, 1, false)
	rapidOverridesFlex.AddItem(overridesPrimitive.rapid100Button, 0, 1, false)

	overridesPrimitive.spindleStopButton = tview.NewButton("Stop").SetSelectedFunc(func() {
		overridesPrimitive.controlPrimitive.QueueRealTimeCommand(grblMod.RealTimeCommandToggleSpindleStop)
	})
	overridesPrimitive.spindleDecr10Button = tview.NewButton("-10%").SetSelectedFunc(func() {
		overridesPrimitive.controlPrimitive.QueueRealTimeCommand(grblMod.RealTimeCommandSpindleSpeedOverrideDecrease10)
	})
	overridesPrimitive.spindleDecr1Button = tview.NewButton("-1%").SetSelectedFunc(func() {
		overridesPrimitive.controlPrimitive.QueueRealTimeCommand(grblMod.RealTimeCommandSpindleSpeedOverrideDecrease1)
	})
	overridesPrimitive.spindle100Button = tview.NewButton("100%").SetSelectedFunc(func() {
		overridesPrimitive.controlPrimitive.QueueRealTimeCommand(grblMod.RealTimeCommandSpindleSpeedOverrideSet100OfProgrammedSpindleSpeed)
	})
	overridesPrimitive.spindleIncr1Button = tview.NewButton("+1%").SetSelectedFunc(func() {
		overridesPrimitive.controlPrimitive.QueueRealTimeCommand(grblMod.RealTimeCommandSpindleSpeedOverrideIncrease1)
	})
	overridesPrimitive.spindleIncr10Button = tview.NewButton("+10%").SetSelectedFunc(func() {
		overridesPrimitive.controlPrimitive.QueueRealTimeCommand(grblMod.RealTimeCommandSpindleSpeedOverrideIncrease10)
	})

	spindleOverridesFlex := tview.NewFlex()
	spindleOverridesFlex.SetBorder(true)
	spindleOverridesFlex.SetTitle("Spindle")
	spindleOverridesFlex.SetDirection(tview.FlexColumn)
	spindleOverridesFlex.AddItem(overridesPrimitive.spindleStopButton, 0, 1, false)
	spindleOverridesFlex.AddItem(overridesPrimitive.spindleDecr10Button, 0, 1, false)
	spindleOverridesFlex.AddItem(overridesPrimitive.spindleDecr1Button, 0, 1, false)
	spindleOverridesFlex.AddItem(overridesPrimitive.spindle100Button, 0, 1, false)
	spindleOverridesFlex.AddItem(overridesPrimitive.spindleIncr1Button, 0, 1, false)
	spindleOverridesFlex.AddItem(overridesPrimitive.spindleIncr10Button, 0, 1, false)

	overridesPrimitive.coolantFloodButton = tview.NewButton("Flood").SetSelectedFunc(func() {
		overridesPrimitive.controlPrimitive.QueueRealTimeCommand(grblMod.RealTimeCommandToggleFloodCoolant)
	})
	overridesPrimitive.coolantMistButton = tview.NewButton("Mist").SetSelectedFunc(func() {
		overridesPrimitive.controlPrimitive.QueueRealTimeCommand(grblMod.RealTimeCommandToggleMistCoolant)
	})

	coolantOverridesFlex := tview.NewFlex()
	coolantOverridesFlex.SetBorder(true)
	coolantOverridesFlex.SetTitle("Coolant")
	coolantOverridesFlex.SetDirection(tview.FlexColumn)
	coolantOverridesFlex.AddItem(overridesPrimitive.coolantFloodButton, 0, 1, false)
	coolantOverridesFlex.AddItem(overridesPrimitive.coolantMistButton, 0, 1, false)

	overridesFlex := tview.NewFlex()
	overridesFlex.SetBorder(true)
	overridesFlex.SetTitle("Overrides")
	overridesFlex.SetDirection(tview.FlexRow)
	overridesFlex.AddItem(feedOverridesFlex, 0, 1, false)
	overridesFlex.AddItem(rapidOverridesFlex, 0, 1, false)
	overridesFlex.AddItem(spindleOverridesFlex, 0, 1, false)
	overridesFlex.AddItem(coolantOverridesFlex, 0, 1, false)
	overridesPrimitive.Flex = overridesFlex

	return overridesPrimitive
}

func (op *OverridesPrimitive) processMessagePushStatusReport(
	messagePushStatusReport *grblMod.MessagePushStatusReport,
) {
	op.app.QueueUpdateDraw(func() {
		switch messagePushStatusReport.MachineState.State {
		case "Idle":
			op.spindleStopButton.SetDisabled(true)
			op.coolantFloodButton.SetDisabled(false)
			op.coolantMistButton.SetDisabled(false)
		case "Run":
			op.spindleStopButton.SetDisabled(true)
			op.coolantFloodButton.SetDisabled(false)
			op.coolantMistButton.SetDisabled(false)
		case "Hold":
			op.spindleStopButton.SetDisabled(false)
			op.coolantFloodButton.SetDisabled(false)
			op.coolantMistButton.SetDisabled(false)
		case "Jog":
			op.spindleStopButton.SetDisabled(true)
			op.coolantFloodButton.SetDisabled(true)
			op.coolantMistButton.SetDisabled(true)
		case "Alarm":
			op.spindleStopButton.SetDisabled(true)
			op.coolantFloodButton.SetDisabled(true)
			op.coolantMistButton.SetDisabled(true)
		case "Door":
			op.spindleStopButton.SetDisabled(true)
			op.coolantFloodButton.SetDisabled(true)
			op.coolantMistButton.SetDisabled(true)
		case "Check":
			op.spindleStopButton.SetDisabled(true)
			op.coolantFloodButton.SetDisabled(true)
			op.coolantMistButton.SetDisabled(true)
		case "Home":
			op.spindleStopButton.SetDisabled(true)
			op.coolantFloodButton.SetDisabled(true)
			op.coolantMistButton.SetDisabled(true)
		case "Sleep":
			op.spindleStopButton.SetDisabled(true)
			op.coolantFloodButton.SetDisabled(true)
			op.coolantMistButton.SetDisabled(true)
		default:
			panic(fmt.Sprintf("unknown machine state: %#v", messagePushStatusReport.MachineState.State))
		}
	})
}

func (op *OverridesPrimitive) ProcessMessage(ctx context.Context, message grblMod.Message) {
	if messagePushStatusReport, ok := message.(*grblMod.MessagePushStatusReport); ok {
		op.processMessagePushStatusReport(messagePushStatusReport)
		return
	}
}
