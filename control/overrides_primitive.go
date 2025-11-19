package control

import (
	"github.com/rivo/tview"

	grblMod "github.com/fornellas/cgs/grbl"
)

type OverridesPrimitive struct {
	*tview.Flex
	controlPrimitive *ControlPrimitive
}

func NewOverridesPrimitive(
	controlPrimitive *ControlPrimitive,
) *OverridesPrimitive {
	overridesPrimitive := &OverridesPrimitive{
		controlPrimitive: controlPrimitive,
	}

	feedOverridesFlex := tview.NewFlex()
	feedOverridesFlex.SetBorder(true)
	feedOverridesFlex.SetTitle("Feed")
	feedOverridesFlex.SetDirection(tview.FlexColumn)
	feedOverridesFlex.AddItem(tview.NewButton("-10%").SetSelectedFunc(func() {
		overridesPrimitive.controlPrimitive.QueueRealTimeCommand(grblMod.RealTimeCommandFeedOverrideDecrease10)
	}), 0, 1, false)
	feedOverridesFlex.AddItem(tview.NewButton("-1%").SetSelectedFunc(func() {
		overridesPrimitive.controlPrimitive.QueueRealTimeCommand(grblMod.RealTimeCommandFeedOverrideDecrease1)
	}), 0, 1, false)
	feedOverridesFlex.AddItem(tview.NewButton("100%").SetSelectedFunc(func() {
		overridesPrimitive.controlPrimitive.QueueRealTimeCommand(grblMod.RealTimeCommandFeedOverrideSet100OfProgrammedRate)
	}), 0, 1, false)
	feedOverridesFlex.AddItem(tview.NewButton("+1%").SetSelectedFunc(func() {
		overridesPrimitive.controlPrimitive.QueueRealTimeCommand(grblMod.RealTimeCommandFeedOverrideIncrease1)
	}), 0, 1, false)
	feedOverridesFlex.AddItem(tview.NewButton("+10%").SetSelectedFunc(func() {
		overridesPrimitive.controlPrimitive.QueueRealTimeCommand(grblMod.RealTimeCommandFeedOverrideIncrease10)
	}), 0, 1, false)

	rapidOverridesFlex := tview.NewFlex()
	rapidOverridesFlex.SetBorder(true)
	rapidOverridesFlex.SetTitle("Rapid")
	rapidOverridesFlex.SetDirection(tview.FlexColumn)
	rapidOverridesFlex.AddItem(tview.NewButton("25%").SetSelectedFunc(func() {
		overridesPrimitive.controlPrimitive.QueueRealTimeCommand(grblMod.RealTimeCommandRapidOverrideSetTo25OfRapidRate)
	}), 0, 1, false)
	rapidOverridesFlex.AddItem(tview.NewButton("50%").SetSelectedFunc(func() {
		overridesPrimitive.controlPrimitive.QueueRealTimeCommand(grblMod.RealTimeCommandRapidOverrideSetTo50OfRapidRate)
	}), 0, 1, false)
	rapidOverridesFlex.AddItem(tview.NewButton("100%").SetSelectedFunc(func() {
		overridesPrimitive.controlPrimitive.QueueRealTimeCommand(grblMod.RealTimeCommandRapidOverrideSetTo100FullRapidRate)
	}), 0, 1, false)

	spindleOverridesFlex := tview.NewFlex()
	spindleOverridesFlex.SetBorder(true)
	spindleOverridesFlex.SetTitle("Spindle")
	spindleOverridesFlex.SetDirection(tview.FlexColumn)
	spindleOverridesFlex.AddItem(tview.NewButton("Stop").SetSelectedFunc(func() {
		overridesPrimitive.controlPrimitive.QueueRealTimeCommand(grblMod.RealTimeCommandToggleSpindleStop)
	}), 0, 1, false)
	spindleOverridesFlex.AddItem(tview.NewButton("-10%").SetSelectedFunc(func() {
		overridesPrimitive.controlPrimitive.QueueRealTimeCommand(grblMod.RealTimeCommandSpindleSpeedOverrideDecrease10)
	}), 0, 1, false)
	spindleOverridesFlex.AddItem(tview.NewButton("-1%").SetSelectedFunc(func() {
		overridesPrimitive.controlPrimitive.QueueRealTimeCommand(grblMod.RealTimeCommandSpindleSpeedOverrideDecrease1)
	}), 0, 1, false)
	spindleOverridesFlex.AddItem(tview.NewButton("100%").SetSelectedFunc(func() {
		overridesPrimitive.controlPrimitive.QueueRealTimeCommand(grblMod.RealTimeCommandSpindleSpeedOverrideSet100OfProgrammedSpindleSpeed)
	}), 0, 1, false)
	spindleOverridesFlex.AddItem(tview.NewButton("+1%").SetSelectedFunc(func() {
		overridesPrimitive.controlPrimitive.QueueRealTimeCommand(grblMod.RealTimeCommandSpindleSpeedOverrideIncrease1)
	}), 0, 1, false)
	spindleOverridesFlex.AddItem(tview.NewButton("+10%").SetSelectedFunc(func() {
		overridesPrimitive.controlPrimitive.QueueRealTimeCommand(grblMod.RealTimeCommandSpindleSpeedOverrideIncrease10)
	}), 0, 1, false)

	coolantOverridesFlex := tview.NewFlex()
	coolantOverridesFlex.SetBorder(true)
	coolantOverridesFlex.SetTitle("Coolant")
	coolantOverridesFlex.SetDirection(tview.FlexColumn)
	coolantOverridesFlex.AddItem(tview.NewButton("Flood").SetSelectedFunc(func() {
		overridesPrimitive.controlPrimitive.QueueRealTimeCommand(grblMod.RealTimeCommandToggleFloodCoolant)
	}), 0, 1, false)
	coolantOverridesFlex.AddItem(tview.NewButton("Mist").SetSelectedFunc(func() {
		overridesPrimitive.controlPrimitive.QueueRealTimeCommand(grblMod.RealTimeCommandToggleMistCoolant)
	}), 0, 1, false)

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

func (cp *OverridesPrimitive) ProcessMessage(message grblMod.Message) {

}
