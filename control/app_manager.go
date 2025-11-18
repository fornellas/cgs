package control

import (
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
	JoggingButton            *tview.Button
	OverridesButton          *tview.Button
	CheckButton              *tview.Button
	DoorButton               *tview.Button
	SleepButton              *tview.Button
	HoldButton               *tview.Button
	ResumeButton             *tview.Button
	SettingsButton           *tview.Button
	SpindleButton            *tview.Button
	CoolantButton            *tview.Button
	ExitButton               *tview.Button
	RootFlex                 *tview.Flex
}

func NewAppManager(
	sendRealTimeCommandCh chan grblMod.RealTimeCommand,
) *AppManager {
	am := &AppManager{}

	tviewApp := tview.NewApplication()
	tviewApp.EnableMouse(true)
	tviewApp.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyCtrlX {
			sendRealTimeCommandCh <- grblMod.RealTimeCommandSoftReset
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
		SetSelectedFunc(func() { sendRealTimeCommandCh <- grblMod.RealTimeCommandSoftReset })

	am.JoggingButton = tview.NewButton("Jogging").
		SetDisabled(true)
	// 	SetSelectedFunc(func() { am.commandDispatcher.QueueCommand("TODO") })

	am.OverridesButton = tview.NewButton("Overrides").
		SetDisabled(true)
	// 	SetSelectedFunc(func() { sendRealTimeCommandCh <- "TODO" })

	am.CheckButton = tview.NewButton("Check").
		SetSelectedFunc(func() { am.CommandDispatcher.QueueCommand("$C") })

	am.DoorButton = tview.NewButton("Door").
		SetSelectedFunc(func() { sendRealTimeCommandCh <- grblMod.RealTimeCommandSafetyDoor })

	am.SleepButton = tview.NewButton("Sleep").
		SetSelectedFunc(func() { am.CommandDispatcher.QueueCommand("$SLP") })

	am.HoldButton = tview.NewButton("Hold").
		SetSelectedFunc(func() { sendRealTimeCommandCh <- grblMod.RealTimeCommandFeedHold })

	am.ResumeButton = tview.NewButton("Resume").
		SetSelectedFunc(func() { sendRealTimeCommandCh <- grblMod.RealTimeCommandCycleStartResume })

	am.SettingsButton = tview.NewButton("Settings").
		SetDisabled(true)
	// 	SetSelectedFunc(func() { am.commandDispatcher.QueueCommand("TODO") })

	am.SpindleButton = tview.NewButton("Spindle").
		SetSelectedFunc(func() { sendRealTimeCommandCh <- grblMod.RealTimeCommandToggleSpindleStop })

	am.CoolantButton = tview.NewButton("Coolant").
		SetSelectedFunc(func() { sendRealTimeCommandCh <- grblMod.RealTimeCommandToggleMistCoolant })

	am.ExitButton = tview.NewButton("Exit").
		SetSelectedFunc(func() { am.App.Stop() })

	am.newRootFlex()

	am.App.SetRoot(am.RootFlex, true).SetFocus(am.CommandInputField)

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
	am.RootFlex = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(
			tview.NewFlex().SetDirection(tview.FlexRow).
				AddItem(
					tview.NewFlex().
						AddItem(
							tview.NewFlex().SetDirection(tview.FlexRow).
								AddItem(am.PushMessagesLogsTextView, 0, 1, false).
								AddItem(am.CommandsTextView, 0, 1, false),
							0, 3, false,
						).
						AddItem(
							tview.NewFlex().SetDirection(tview.FlexRow).
								AddItem(am.GcodeParserTextView, 0, 1, false).
								AddItem(am.GcodeParamsTextView, 0, 1, false),
							0, 2, false,
						).
						AddItem(
							tview.NewFlex().SetDirection(tview.FlexRow).
								AddItem(am.StateTextView, 4, 0, false).
								AddItem(am.StatusTextView, 0, 1, false),
							14, 0, false,
						),
					0, 1, false,
				).
				AddItem(am.FeedbackTextView, 1, 0, false).
				AddItem(am.CommandInputField, 1, 0, false),
			0, 1, false,
		).
		AddItem(
			tview.NewFlex().
				AddItem(am.HomingButton, 0, 1, false).
				AddItem(am.ResetButton, 0, 1, false).
				AddItem(am.OverridesButton, 0, 1, false).
				AddItem(am.DoorButton, 0, 1, false).
				AddItem(am.HoldButton, 0, 1, false).
				AddItem(am.SettingsButton, 0, 1, false).
				AddItem(am.CoolantButton, 0, 1, false),
			1, 0, false,
		).
		AddItem(
			tview.NewFlex().
				AddItem(am.UnlockButton, 0, 1, false).
				AddItem(am.JoggingButton, 0, 1, false).
				AddItem(am.CheckButton, 0, 1, false).
				AddItem(am.SleepButton, 0, 1, false).
				AddItem(am.ResumeButton, 0, 1, false).
				AddItem(am.SpindleButton, 0, 1, false).
				AddItem(am.ExitButton, 0, 1, false),
			1, 0, false,
		)
}
