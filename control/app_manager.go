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
	textView.SetBackgroundColor(tcell.ColorDefault)
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
	textView.SetBackgroundColor(tcell.ColorDefault)
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
	textView.SetBackgroundColor(tcell.ColorDefault)
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
	textView.SetBackgroundColor(tcell.ColorDefault)
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
	textView.SetBackgroundColor(tcell.ColorDefault)
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
	textView.SetBackgroundColor(tcell.ColorDefault)
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
	textView.SetBackgroundColor(tcell.ColorDefault)
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
	inputField.SetBackgroundColor(tcell.ColorDefault)
	am.CommandInputField = inputField
}

func (am *AppManager) newRootFlex() {
	newPrimitive := func(text string) tview.Primitive {
		textView := tview.NewTextView()
		textView.SetTextAlign(tview.AlignCenter)
		textView.SetText(text)
		textView.SetBorder(true)
		return textView
	}

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

	column0Flex := tview.NewFlex()
	// column0Flex.SetBorder(true)
	column0Flex.SetDirection(tview.FlexRow)
	column0Flex.AddItem(newPrimitive("Main"), 0, 1, false)
	column0Flex.AddItem(am.FeedbackTextView, 1, 0, false)
	column0Flex.AddItem(buttonsFlex, 4, 0, false)

	column1Flex := tview.NewFlex()
	// column1Flex.SetBorder(true)
	column1Flex.SetDirection(tview.FlexRow)
	column1Flex.AddItem(am.StateTextView, 4, 0, false)
	column1Flex.AddItem(am.StatusTextView, 0, 1, false)

	rootFlex := tview.NewFlex()
	// rootFlex.SetBorder(true)
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
		am.HelpButton.SetDisabled(false) // TODO verify
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
		am.HelpButton.SetDisabled(false) // TODO verify
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
