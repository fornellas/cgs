package control

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	grblMod "github.com/fornellas/cgs/grbl"
)

type AppManager struct {
	CommandDispatcher        *CommandDispatcher
	App                      *tview.Application
	CommandsTextView         *tview.TextView
	PushMessagesLogsTextView *tview.TextView
	FeedbackTextView         *tview.TextView
	GcodeParserTextView      *tview.TextView
	GcodeParamsTextView      *tview.TextView
	StateTextView            *tview.TextView
	StatusTextView           *tview.TextView
	CommandInputField        *tview.InputField
	HomingButton             *tview.Button
	UnlockButton             *tview.Button
	ResetButton              *tview.Button
	CheckButton              *tview.Button
	DoorButton               *tview.Button
	SleepButton              *tview.Button
	HelpButton               *tview.Button
	HoldButton               *tview.Button
	ResumeButton             *tview.Button
	RootFlex                 *tview.Flex
	disableCommandInput      bool
	machineState             *grblMod.StatusReportMachineState
}

func NewAppManager() *AppManager {
	am := &AppManager{}

	tviewApp := tview.NewApplication()
	tviewApp.EnableMouse(true)
	tviewApp.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyCtrlX {
			am.CommandDispatcher.QueueRealTimeCommand(grblMod.RealTimeCommandSoftReset)
			return nil
		}
		return event
	})
	am.App = tviewApp

	am.newCommandsTextView()
	am.newPushMessagesLogsTextView()
	am.newFeedbackTextView()
	am.newGcodeParserTextView()
	am.newGcodeParamsTextView()
	am.newStateTextView()
	am.newStatusTextView()
	am.newCommandInputField()
	am.HomingButton = tview.NewButton("Homing").
		SetSelectedFunc(func() { am.CommandDispatcher.QueueCommand("$H") })
	am.UnlockButton = tview.NewButton("Unlock").
		SetSelectedFunc(func() { am.CommandDispatcher.QueueCommand("$X") })
	am.ResetButton = tview.NewButton("Reset").
		SetSelectedFunc(func() { am.CommandDispatcher.QueueRealTimeCommand(grblMod.RealTimeCommandSoftReset) })
	am.CheckButton = tview.NewButton("Check").
		SetSelectedFunc(func() { am.CommandDispatcher.QueueCommand("$C") })
	am.DoorButton = tview.NewButton("Door").
		SetSelectedFunc(func() { am.CommandDispatcher.QueueRealTimeCommand(grblMod.RealTimeCommandSafetyDoor) })
	am.SleepButton = tview.NewButton("Sleep").
		SetSelectedFunc(func() { am.CommandDispatcher.QueueCommand("$SLP") })
	am.HelpButton = tview.NewButton("Help").
		SetSelectedFunc(func() { am.CommandDispatcher.QueueCommand("$") })
	am.HoldButton = tview.NewButton("Hold").
		SetSelectedFunc(func() { am.CommandDispatcher.QueueRealTimeCommand(grblMod.RealTimeCommandFeedHold) })
	am.ResumeButton = tview.NewButton("Resume").
		SetSelectedFunc(func() { am.CommandDispatcher.QueueRealTimeCommand(grblMod.RealTimeCommandCycleStartResume) })

	am.newRootFlex()

	am.App.SetRoot(am.RootFlex, true)

	return am
}

func (am *AppManager) newCommandsTextView() {
	textView := tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true).
		SetWrap(true)
	textView.SetBorder(true).SetTitle("Commands")
	textView.SetChangedFunc(func() {
		textView.ScrollToEnd()
		am.App.Draw()
	})
	am.CommandsTextView = textView
}

func (am *AppManager) newPushMessagesLogsTextView() {
	textView := tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true).
		SetWrap(true)
	textView.SetBorder(true).SetTitle("Push Messages / Logs")
	textView.SetChangedFunc(func() {
		textView.ScrollToEnd()
		am.App.Draw()
	})
	am.PushMessagesLogsTextView = textView
}

func (am *AppManager) newFeedbackTextView() {
	textView := tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true).
		SetWrap(true)
	textView.SetTitle("Feedback Message")
	textView.SetChangedFunc(func() {
		textView.ScrollToEnd()
		am.App.Draw()
	})
	am.FeedbackTextView = textView
}

func (am *AppManager) newGcodeParserTextView() {
	textView := tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true).
		SetWrap(true)
	textView.SetBorder(true).SetTitle("G-Code Parser")
	textView.SetChangedFunc(func() {
		textView.ScrollToBeginning()
		am.App.Draw()
	})
	am.GcodeParserTextView = textView
}

func (am *AppManager) newGcodeParamsTextView() {
	textView := tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true).
		SetWrap(true)
	textView.SetBorder(true).SetTitle("G-Code Parameters")
	textView.SetChangedFunc(func() {
		textView.ScrollToBeginning()
		am.App.Draw()
	})
	am.GcodeParamsTextView = textView
}

func (am *AppManager) newStateTextView() {
	textView := tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true).
		SetTextAlign(tview.AlignCenter).
		SetWrap(true)
	textView.SetBorder(true).SetTitle("State")
	textView.SetChangedFunc(func() {
		am.App.Draw()
	})
	am.StateTextView = textView
}

func (am *AppManager) newStatusTextView() {
	textView := tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true).
		SetWrap(true)
	textView.SetBorder(true).SetTitle("Status")
	textView.SetChangedFunc(func() {
		textView.ScrollToBeginning()
		am.App.Draw()
	})
	am.StatusTextView = textView
}

func (am *AppManager) newCommandInputField() {
	inputField := tview.NewInputField().
		SetLabel("Command: ")
	inputField.SetDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEscape:
			inputField.SetText("")
		case tcell.KeyEnter:
			command := inputField.GetText()
			if command == "" {
				return
			}
			am.CommandDispatcher.QueueCommand(command)
		}
	})
	am.CommandInputField = inputField
}

func (am *AppManager) getButtonsFLex() *tview.Flex {
	commandButtonsFlex := tview.NewFlex()
	commandButtonsFlex.SetTitle("Commands")
	commandButtonsFlex.SetBorder(true)
	commandButtonsFlex.SetDirection(tview.FlexRow)
	commandButtonsFlex.AddItem(
		tview.NewFlex().SetDirection(tview.FlexColumn).
			AddItem(am.HomingButton, 0, 1, false).
			AddItem(am.UnlockButton, 0, 1, false).
			AddItem(am.CheckButton, 0, 1, false),
		0, 1, false,
	)
	commandButtonsFlex.AddItem(
		tview.NewFlex().SetDirection(tview.FlexColumn).
			AddItem(am.SleepButton, 0, 1, false).
			AddItem(am.HelpButton, 0, 1, false),
		0, 1, false,
	)

	realtimeCommandButtonsFlex := tview.NewFlex()
	realtimeCommandButtonsFlex.SetTitle("Realtime Commands")
	realtimeCommandButtonsFlex.SetBorder(true)
	realtimeCommandButtonsFlex.SetDirection(tview.FlexRow)
	realtimeCommandButtonsFlex.AddItem(
		tview.NewFlex().SetDirection(tview.FlexColumn).
			AddItem(am.ResetButton, 0, 1, false).
			AddItem(am.DoorButton, 0, 1, false),
		0, 1, false,
	)
	realtimeCommandButtonsFlex.AddItem(
		tview.NewFlex().SetDirection(tview.FlexColumn).
			AddItem(am.HoldButton, 0, 1, false).
			AddItem(am.ResumeButton, 0, 1, false),
		0, 1, false,
	)

	buttonsFlex := tview.NewFlex()
	buttonsFlex.SetDirection(tview.FlexColumn)
	buttonsFlex.AddItem(commandButtonsFlex, 0, 1, false)
	buttonsFlex.AddItem(realtimeCommandButtonsFlex, 0, 1, false)

	return buttonsFlex
}

func (am *AppManager) getJoggingPrimitive() tview.Primitive {
	joggingFlex := tview.NewFlex()
	joggingFlex.SetBorder(true)
	joggingFlex.SetTitle("Jogging")
	return joggingFlex
}

func (am *AppManager) getControlPrimitive() tview.Primitive {
	gcodeFlex := tview.NewFlex()
	gcodeFlex.SetDirection(tview.FlexColumn)
	gcodeFlex.AddItem(am.GcodeParserTextView, 0, 1, false)
	gcodeFlex.AddItem(am.GcodeParamsTextView, 0, 1, false)

	commsFlex := tview.NewFlex()
	commsFlex.SetDirection(tview.FlexColumn)
	commsFlex.AddItem(am.CommandsTextView, 0, 1, false)
	commsFlex.AddItem(am.PushMessagesLogsTextView, 0, 1, false)

	controlFlex := tview.NewFlex()
	controlFlex.SetBorder(true)
	controlFlex.SetTitle("Contrtol")
	controlFlex.SetDirection(tview.FlexRow)
	controlFlex.AddItem(gcodeFlex, 0, 1, false)
	controlFlex.AddItem(commsFlex, 0, 1, false)
	controlFlex.AddItem(am.CommandInputField, 1, 0, true)

	return controlFlex
}

func (am *AppManager) getOverridesPrimitive() tview.Primitive {
	feedOverridesFlex := tview.NewFlex()
	feedOverridesFlex.SetBorder(true)
	feedOverridesFlex.SetTitle("Feed")
	feedOverridesFlex.SetDirection(tview.FlexColumn)
	feedOverridesFlex.AddItem(tview.NewButton("-10%").SetSelectedFunc(func() {
		am.CommandDispatcher.QueueRealTimeCommand(grblMod.RealTimeCommandFeedOverrideDecrease10)
	}), 0, 1, false)
	feedOverridesFlex.AddItem(tview.NewButton("-1%").SetSelectedFunc(func() {
		am.CommandDispatcher.QueueRealTimeCommand(grblMod.RealTimeCommandFeedOverrideDecrease1)
	}), 0, 1, false)
	feedOverridesFlex.AddItem(tview.NewButton("100%").SetSelectedFunc(func() {
		am.CommandDispatcher.QueueRealTimeCommand(grblMod.RealTimeCommandFeedOverrideSet100OfProgrammedRate)
	}), 0, 1, false)
	feedOverridesFlex.AddItem(tview.NewButton("+1%").SetSelectedFunc(func() {
		am.CommandDispatcher.QueueRealTimeCommand(grblMod.RealTimeCommandFeedOverrideIncrease1)
	}), 0, 1, false)
	feedOverridesFlex.AddItem(tview.NewButton("+10%").SetSelectedFunc(func() {
		am.CommandDispatcher.QueueRealTimeCommand(grblMod.RealTimeCommandFeedOverrideIncrease10)
	}), 0, 1, false)

	rapidOverridesFlex := tview.NewFlex()
	rapidOverridesFlex.SetBorder(true)
	rapidOverridesFlex.SetTitle("Rapid")
	rapidOverridesFlex.SetDirection(tview.FlexColumn)
	rapidOverridesFlex.AddItem(tview.NewButton("25%").SetSelectedFunc(func() {
		am.CommandDispatcher.QueueRealTimeCommand(grblMod.RealTimeCommandRapidOverrideSetTo25OfRapidRate)
	}), 0, 1, false)
	rapidOverridesFlex.AddItem(tview.NewButton("50%").SetSelectedFunc(func() {
		am.CommandDispatcher.QueueRealTimeCommand(grblMod.RealTimeCommandRapidOverrideSetTo50OfRapidRate)
	}), 0, 1, false)
	rapidOverridesFlex.AddItem(tview.NewButton("100%").SetSelectedFunc(func() {
		am.CommandDispatcher.QueueRealTimeCommand(grblMod.RealTimeCommandRapidOverrideSetTo100FullRapidRate)
	}), 0, 1, false)

	spindleOverridesFlex := tview.NewFlex()
	spindleOverridesFlex.SetBorder(true)
	spindleOverridesFlex.SetTitle("Spindle")
	spindleOverridesFlex.SetDirection(tview.FlexColumn)
	spindleOverridesFlex.AddItem(tview.NewButton("Stop").SetSelectedFunc(func() {
		am.CommandDispatcher.QueueRealTimeCommand(grblMod.RealTimeCommandToggleSpindleStop)
	}), 0, 1, false)
	spindleOverridesFlex.AddItem(tview.NewButton("-10%").SetSelectedFunc(func() {
		am.CommandDispatcher.QueueRealTimeCommand(grblMod.RealTimeCommandSpindleSpeedOverrideDecrease10)
	}), 0, 1, false)
	spindleOverridesFlex.AddItem(tview.NewButton("-1%").SetSelectedFunc(func() {
		am.CommandDispatcher.QueueRealTimeCommand(grblMod.RealTimeCommandSpindleSpeedOverrideDecrease1)
	}), 0, 1, false)
	spindleOverridesFlex.AddItem(tview.NewButton("100%").SetSelectedFunc(func() {
		am.CommandDispatcher.QueueRealTimeCommand(grblMod.RealTimeCommandSpindleSpeedOverrideSet100OfProgrammedSpindleSpeed)
	}), 0, 1, false)
	spindleOverridesFlex.AddItem(tview.NewButton("+1%").SetSelectedFunc(func() {
		am.CommandDispatcher.QueueRealTimeCommand(grblMod.RealTimeCommandSpindleSpeedOverrideIncrease1)
	}), 0, 1, false)
	spindleOverridesFlex.AddItem(tview.NewButton("+10%").SetSelectedFunc(func() {
		am.CommandDispatcher.QueueRealTimeCommand(grblMod.RealTimeCommandSpindleSpeedOverrideIncrease10)
	}), 0, 1, false)

	coolantOverridesFlex := tview.NewFlex()
	coolantOverridesFlex.SetBorder(true)
	coolantOverridesFlex.SetTitle("Coolant")
	coolantOverridesFlex.SetDirection(tview.FlexColumn)
	coolantOverridesFlex.AddItem(tview.NewButton("Flood").SetSelectedFunc(func() {
		am.CommandDispatcher.QueueRealTimeCommand(grblMod.RealTimeCommandToggleFloodCoolant)
	}), 0, 1, false)
	coolantOverridesFlex.AddItem(tview.NewButton("Mist").SetSelectedFunc(func() {
		am.CommandDispatcher.QueueRealTimeCommand(grblMod.RealTimeCommandToggleMistCoolant)
	}), 0, 1, false)

	overridesFlex := tview.NewFlex()
	overridesFlex.SetBorder(true)
	overridesFlex.SetTitle("Overrides")
	overridesFlex.SetDirection(tview.FlexRow)
	overridesFlex.AddItem(feedOverridesFlex, 0, 1, false)
	overridesFlex.AddItem(rapidOverridesFlex, 0, 1, false)
	overridesFlex.AddItem(spindleOverridesFlex, 0, 1, false)
	overridesFlex.AddItem(coolantOverridesFlex, 0, 1, false)

	return overridesFlex
}

func (am *AppManager) getStreamPrimitive() tview.Primitive {
	return tview.NewBox().SetBorder(true).SetTitle("Stream")
}

func (am *AppManager) getScriptPrimitive() tview.Primitive {
	return tview.NewBox().SetBorder(true).SetTitle("Script")
}

func (am *AppManager) getSettingsPrimitive() tview.Primitive {
	return tview.NewBox().SetBorder(true).SetTitle("Settings")
}

func (am *AppManager) getMainFlex() *tview.Flex {
	jogging := am.getJoggingPrimitive()
	control := am.getControlPrimitive()
	overrides := am.getOverridesPrimitive()
	stream := am.getStreamPrimitive()
	script := am.getScriptPrimitive()
	settings := am.getSettingsPrimitive()

	page := tview.NewPages()
	page.AddPage("Control", control, true, true)
	page.AddPage("Jogging", jogging, true, true)
	page.AddPage("Overrides", overrides, true, true)
	page.AddPage("Stream", stream, true, true)
	page.AddPage("Script", script, true, true)
	page.AddPage("Settings", settings, true, true)

	buttonsFlex := tview.NewFlex()
	buttonsFlex.SetDirection(tview.FlexColumn)
	buttonsFlex.AddItem(tview.NewButton("Control").SetSelectedFunc(func() {
		page.SwitchToPage("Control")
	}), 0, 1, false)
	buttonsFlex.AddItem(tview.NewButton("Jogging").SetSelectedFunc(func() {
		page.SwitchToPage("Jogging")
	}), 0, 1, false)
	buttonsFlex.AddItem(tview.NewButton("Overrides").SetSelectedFunc(func() {
		page.SwitchToPage("Overrides")
	}), 0, 1, false)
	buttonsFlex.AddItem(tview.NewButton("Stream").SetSelectedFunc(func() {
		page.SwitchToPage("Stream")
	}), 0, 1, false)
	buttonsFlex.AddItem(tview.NewButton("Script").SetSelectedFunc(func() {
		page.SwitchToPage("Script")
	}), 0, 1, false)
	buttonsFlex.AddItem(tview.NewButton("Settings").SetSelectedFunc(func() {
		page.SwitchToPage("Settings")
	}), 0, 1, false)
	page.SwitchToPage("Control")

	mainFlex := tview.NewFlex()
	mainFlex.SetDirection(tview.FlexRow)
	mainFlex.AddItem(buttonsFlex, 1, 0, false)
	mainFlex.AddItem(page, 0, 1, false)

	return mainFlex
}

func (am *AppManager) newRootFlex() {
	column0Flex := tview.NewFlex()
	column0Flex.SetDirection(tview.FlexRow)
	column0Flex.AddItem(am.getMainFlex(), 0, 1, false)
	column0Flex.AddItem(am.FeedbackTextView, 1, 0, false)
	column0Flex.AddItem(am.getButtonsFLex(), 4, 0, false)

	column1Flex := tview.NewFlex()
	column1Flex.SetDirection(tview.FlexRow)
	column1Flex.AddItem(am.StateTextView, 4, 0, false)
	column1Flex.AddItem(am.StatusTextView, 0, 1, false)

	rootFlex := tview.NewFlex()
	rootFlex.SetDirection(tview.FlexColumn)
	rootFlex.AddItem(column0Flex, 0, 1, false)
	rootFlex.AddItem(column1Flex, 15, 0, false)

	am.RootFlex = rootFlex
}

func (am *AppManager) updateCommandInputRaw() {
	if am.disableCommandInput || am.machineState == nil {
		am.CommandInputField.SetDisabled(true)
		am.HomingButton.SetDisabled(true)
		am.UnlockButton.SetDisabled(true)
		am.ResetButton.SetDisabled(true)
		am.CheckButton.SetDisabled(true)
		am.DoorButton.SetDisabled(true)
		am.SleepButton.SetDisabled(true)
		am.HelpButton.SetDisabled(true)
		am.HoldButton.SetDisabled(true)
		am.ResumeButton.SetDisabled(true)
		return
	}
	switch am.machineState.State {
	case "Idle":
		am.CommandInputField.SetDisabled(false)
		am.HomingButton.SetDisabled(false)
		am.UnlockButton.SetDisabled(true)
		am.ResetButton.SetDisabled(false)
		am.CheckButton.SetDisabled(false)
		am.DoorButton.SetDisabled(false)
		am.SleepButton.SetDisabled(false)
		am.HelpButton.SetDisabled(false)
		am.HoldButton.SetDisabled(false)
		am.ResumeButton.SetDisabled(true)
	case "Run":
		am.CommandInputField.SetDisabled(true)
		am.HomingButton.SetDisabled(true)
		am.UnlockButton.SetDisabled(true)
		am.ResetButton.SetDisabled(false)
		am.CheckButton.SetDisabled(true)
		am.DoorButton.SetDisabled(false)
		am.SleepButton.SetDisabled(true)
		am.HelpButton.SetDisabled(true)
		am.HoldButton.SetDisabled(false)
		am.ResumeButton.SetDisabled(true)
	case "Hold":
		am.CommandInputField.SetDisabled(true)
		am.HomingButton.SetDisabled(true)
		am.UnlockButton.SetDisabled(true)
		am.ResetButton.SetDisabled(false)
		am.CheckButton.SetDisabled(true)
		am.DoorButton.SetDisabled(false)
		am.SleepButton.SetDisabled(true)
		am.HelpButton.SetDisabled(true)
		am.HoldButton.SetDisabled(true)
		am.ResumeButton.SetDisabled(false)
	case "Jog":
		am.CommandInputField.SetDisabled(true)
		am.HomingButton.SetDisabled(true)
		am.UnlockButton.SetDisabled(true)
		am.ResetButton.SetDisabled(false)
		am.CheckButton.SetDisabled(true)
		am.DoorButton.SetDisabled(false)
		am.SleepButton.SetDisabled(true)
		am.HelpButton.SetDisabled(true)
		am.HoldButton.SetDisabled(false)
		am.ResumeButton.SetDisabled(true)
	case "Alarm":
		am.CommandInputField.SetDisabled(true)
		am.HomingButton.SetDisabled(false)
		am.UnlockButton.SetDisabled(false)
		am.ResetButton.SetDisabled(false)
		am.CheckButton.SetDisabled(true)
		am.DoorButton.SetDisabled(true)
		am.SleepButton.SetDisabled(false)
		am.HelpButton.SetDisabled(false)
		am.HoldButton.SetDisabled(true)
		am.ResumeButton.SetDisabled(true)
	case "Door":
		am.CommandInputField.SetDisabled(true)
		am.HomingButton.SetDisabled(true)
		am.UnlockButton.SetDisabled(true)
		am.ResetButton.SetDisabled(false)
		am.CheckButton.SetDisabled(true)
		am.DoorButton.SetDisabled(true)
		am.SleepButton.SetDisabled(true)
		am.HelpButton.SetDisabled(true)
		am.HoldButton.SetDisabled(true)
		am.ResumeButton.SetDisabled(false)
	case "Check":
		am.CommandInputField.SetDisabled(false)
		am.HomingButton.SetDisabled(true)
		am.UnlockButton.SetDisabled(true)
		am.ResetButton.SetDisabled(false)
		am.CheckButton.SetDisabled(false)
		am.DoorButton.SetDisabled(true)
		am.SleepButton.SetDisabled(false)
		am.HelpButton.SetDisabled(false)
		am.HoldButton.SetDisabled(true)
		am.ResumeButton.SetDisabled(true)
	case "Home":
		am.CommandInputField.SetDisabled(true)
		am.HomingButton.SetDisabled(true)
		am.UnlockButton.SetDisabled(true)
		am.ResetButton.SetDisabled(false)
		am.CheckButton.SetDisabled(true)
		am.DoorButton.SetDisabled(false)
		am.SleepButton.SetDisabled(true)
		am.HelpButton.SetDisabled(true)
		am.HoldButton.SetDisabled(true)
		am.ResumeButton.SetDisabled(true)
	case "Sleep":
		am.CommandInputField.SetDisabled(true)
		am.HomingButton.SetDisabled(true)
		am.UnlockButton.SetDisabled(true)
		am.ResetButton.SetDisabled(false)
		am.CheckButton.SetDisabled(true)
		am.DoorButton.SetDisabled(true)
		am.SleepButton.SetDisabled(true)
		am.HelpButton.SetDisabled(true)
		am.HoldButton.SetDisabled(true)
		am.ResumeButton.SetDisabled(true)
	default:
		panic(fmt.Errorf("unknown state: %s", am.machineState.State))
	}
}
func (am *AppManager) UpdateCommandInput(machineState grblMod.StatusReportMachineState) {
	am.machineState = &machineState
	am.updateCommandInputRaw()
}

// Disable all primitives that can receive command input.
func (am *AppManager) DisableCommandInput() {
	am.disableCommandInput = true
	am.updateCommandInputRaw()
}

// Enable all primitives that can receive command input.
func (am *AppManager) EnableCommandInput() {
	am.disableCommandInput = false
	am.updateCommandInputRaw()
}
