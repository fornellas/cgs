package control

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	grblMod "github.com/fornellas/cgs/grbl"
)

type RootPrimitive struct {
	*tview.Flex
	app                *tview.Application
	grbl               *grblMod.Grbl
	statusPrimitive    *StatusPrimitive
	controlPrimitive   *ControlPrimitive
	overridesPrimitive *OverridesPrimitive
	joggingPrimitive   *JoggingPrimitive
	logsPrimitive      *LogsPrimitive
	// Common
	feedbackTextView *tview.TextView
	homeButton       *tview.Button
	unlockButton     *tview.Button
	resetButton      *tview.Button
	checkButton      *tview.Button
	doorButton       *tview.Button
	sleepButton      *tview.Button
	helpButton       *tview.Button
	holdButton       *tview.Button
	resumeButton     *tview.Button

	machineState *grblMod.StatusReportMachineState
}

func NewRootPrimitive(
	app *tview.Application,
	grbl *grblMod.Grbl,
	statusPrimitive *StatusPrimitive,
	controlPrimitive *ControlPrimitive,
	overridesPrimitive *OverridesPrimitive,
	joggingPrimitive *JoggingPrimitive,
	logsPrimitive *LogsPrimitive,
) *RootPrimitive {
	rp := &RootPrimitive{
		grbl:               grbl,
		app:                app,
		statusPrimitive:    statusPrimitive,
		controlPrimitive:   controlPrimitive,
		overridesPrimitive: overridesPrimitive,
		joggingPrimitive:   joggingPrimitive,
		logsPrimitive:      logsPrimitive,
	}

	rp.newFeedbackTextView()
	rp.homeButton = tview.NewButton("Home").
		SetSelectedFunc(func() { rp.controlPrimitive.QueueCommand("$H") })
	rp.unlockButton = tview.NewButton("Unlock").
		SetSelectedFunc(func() { rp.controlPrimitive.QueueCommand("$X") })
	rp.resetButton = tview.NewButton("Reset").
		SetSelectedFunc(func() { rp.controlPrimitive.QueueRealTimeCommand(grblMod.RealTimeCommandSoftReset) })
	rp.checkButton = tview.NewButton("Check").
		SetSelectedFunc(func() { rp.controlPrimitive.QueueCommand("$C") })
	rp.doorButton = tview.NewButton("Door").
		SetSelectedFunc(func() { rp.controlPrimitive.QueueRealTimeCommand(grblMod.RealTimeCommandSafetyDoor) })
	rp.sleepButton = tview.NewButton("Sleep").
		SetSelectedFunc(func() { rp.controlPrimitive.QueueCommand("$SLP") })
	rp.helpButton = tview.NewButton("Help").
		SetSelectedFunc(func() { rp.controlPrimitive.QueueCommand("$") })
	rp.holdButton = tview.NewButton("Hold").
		SetSelectedFunc(func() { rp.controlPrimitive.QueueRealTimeCommand(grblMod.RealTimeCommandFeedHold) })
	rp.resumeButton = tview.NewButton("Resume").
		SetSelectedFunc(func() {
			rp.controlPrimitive.QueueRealTimeCommand(grblMod.RealTimeCommandCycleStartResume)
		})

	rp.newRootFlex()

	return rp
}

func (rp *RootPrimitive) newFeedbackTextView() {
	textView := tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true).
		SetWrap(true)
	textView.SetTitle("Feedback Message")
	textView.SetChangedFunc(func() {
		textView.ScrollToEnd()
		rp.app.Draw()
	})
	rp.feedbackTextView = textView
}

func (rp *RootPrimitive) getButtonsFLex() *tview.Flex {
	commandButtonsFlex := tview.NewFlex()
	commandButtonsFlex.SetTitle("Commands")
	commandButtonsFlex.SetBorder(true)
	commandButtonsFlex.SetDirection(tview.FlexRow)
	commandButtonsFlex.AddItem(
		tview.NewFlex().SetDirection(tview.FlexColumn).
			AddItem(rp.homeButton, 0, 1, false).
			AddItem(rp.unlockButton, 0, 1, false).
			AddItem(rp.checkButton, 0, 1, false),
		0, 1, false,
	)
	commandButtonsFlex.AddItem(
		tview.NewFlex().SetDirection(tview.FlexColumn).
			AddItem(rp.sleepButton, 0, 1, false).
			AddItem(rp.helpButton, 0, 1, false),
		0, 1, false,
	)

	realtimeCommandButtonsFlex := tview.NewFlex()
	realtimeCommandButtonsFlex.SetTitle("Realtime Commands")
	realtimeCommandButtonsFlex.SetBorder(true)
	realtimeCommandButtonsFlex.SetDirection(tview.FlexRow)
	realtimeCommandButtonsFlex.AddItem(
		tview.NewFlex().SetDirection(tview.FlexColumn).
			AddItem(rp.resetButton, 0, 1, false).
			AddItem(rp.doorButton, 0, 1, false),
		0, 1, false,
	)
	realtimeCommandButtonsFlex.AddItem(
		tview.NewFlex().SetDirection(tview.FlexColumn).
			AddItem(rp.holdButton, 0, 1, false).
			AddItem(rp.resumeButton, 0, 1, false),
		0, 1, false,
	)

	buttonsFlex := tview.NewFlex()
	buttonsFlex.SetDirection(tview.FlexColumn)
	buttonsFlex.AddItem(commandButtonsFlex, 0, 1, false)
	buttonsFlex.AddItem(realtimeCommandButtonsFlex, 0, 1, false)

	return buttonsFlex
}

func (rp *RootPrimitive) getStreamPrimitive() tview.Primitive {
	return tview.NewBox().SetBorder(true).SetTitle("Stream")
}

func (rp *RootPrimitive) getScriptPrimitive() tview.Primitive {
	return tview.NewBox().SetBorder(true).SetTitle("Script")
}

func (rp *RootPrimitive) getSettingsPrimitive() tview.Primitive {
	return tview.NewBox().SetBorder(true).SetTitle("Settings")
}

func (rp *RootPrimitive) getMainFlex() *tview.Flex {
	stream := rp.getStreamPrimitive()
	script := rp.getScriptPrimitive()
	settings := rp.getSettingsPrimitive()

	page := tview.NewPages()
	page.AddPage("Control", rp.controlPrimitive, true, true)
	page.AddPage("Jogging", rp.joggingPrimitive, true, true)
	page.AddPage("Overrides", rp.overridesPrimitive, true, true)
	page.AddPage("Stream", stream, true, true)
	page.AddPage("Script", script, true, true)
	page.AddPage("Settings", settings, true, true)
	page.AddPage("Logs", rp.logsPrimitive, true, true)

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
	buttonsFlex.AddItem(tview.NewButton("Logs").SetSelectedFunc(func() {
		page.SwitchToPage("Logs")
	}), 0, 1, false)
	page.SwitchToPage("Control")

	mainFlex := tview.NewFlex()
	mainFlex.SetDirection(tview.FlexRow)
	mainFlex.AddItem(buttonsFlex, 1, 0, false)
	mainFlex.AddItem(page, 0, 1, false)

	return mainFlex
}

func (rp *RootPrimitive) newRootFlex() {
	column0Flex := tview.NewFlex()
	column0Flex.SetDirection(tview.FlexRow)
	column0Flex.AddItem(rp.getMainFlex(), 0, 1, false)
	column0Flex.AddItem(rp.feedbackTextView, 1, 0, false)
	column0Flex.AddItem(rp.getButtonsFLex(), 4, 0, false)

	rootFlex := tview.NewFlex()
	rootFlex.SetDirection(tview.FlexColumn)
	rootFlex.AddItem(column0Flex, 0, 1, false)
	rootFlex.AddItem(rp.statusPrimitive, rp.statusPrimitive.FixedSize()+1, 0, false)

	rp.Flex = rootFlex
}

func (rp *RootPrimitive) updateDisabled() {
	rp.app.QueueUpdateDraw(func() {
		if rp.machineState == nil {
			rp.homeButton.SetDisabled(true)
			rp.unlockButton.SetDisabled(true)
			rp.resetButton.SetDisabled(true)
			rp.checkButton.SetDisabled(true)
			rp.doorButton.SetDisabled(true)
			rp.sleepButton.SetDisabled(true)
			rp.helpButton.SetDisabled(true)
			rp.holdButton.SetDisabled(true)
			rp.resumeButton.SetDisabled(true)
			return
		}
		switch rp.machineState.State {
		case "Idle":
			rp.homeButton.SetDisabled(false)
			rp.unlockButton.SetDisabled(true)
			rp.resetButton.SetDisabled(false)
			rp.checkButton.SetDisabled(false)
			rp.doorButton.SetDisabled(false)
			rp.sleepButton.SetDisabled(false)
			rp.helpButton.SetDisabled(false)
			rp.holdButton.SetDisabled(false)
			rp.resumeButton.SetDisabled(true)
		case "Run":
			rp.homeButton.SetDisabled(true)
			rp.unlockButton.SetDisabled(true)
			rp.resetButton.SetDisabled(false)
			rp.checkButton.SetDisabled(true)
			rp.doorButton.SetDisabled(false)
			rp.sleepButton.SetDisabled(true)
			rp.helpButton.SetDisabled(true)
			rp.holdButton.SetDisabled(false)
			rp.resumeButton.SetDisabled(true)
		case "Hold":
			rp.homeButton.SetDisabled(true)
			rp.unlockButton.SetDisabled(true)
			rp.resetButton.SetDisabled(false)
			rp.checkButton.SetDisabled(true)
			rp.doorButton.SetDisabled(false)
			rp.sleepButton.SetDisabled(true)
			rp.helpButton.SetDisabled(true)
			rp.holdButton.SetDisabled(true)
			rp.resumeButton.SetDisabled(false)
		case "Jog":
			rp.homeButton.SetDisabled(true)
			rp.unlockButton.SetDisabled(true)
			rp.resetButton.SetDisabled(false)
			rp.checkButton.SetDisabled(true)
			rp.doorButton.SetDisabled(false)
			rp.sleepButton.SetDisabled(true)
			rp.helpButton.SetDisabled(true)
			rp.holdButton.SetDisabled(false)
			rp.resumeButton.SetDisabled(true)
		case "Alarm":
			rp.homeButton.SetDisabled(false)
			rp.unlockButton.SetDisabled(false)
			rp.resetButton.SetDisabled(false)
			rp.checkButton.SetDisabled(true)
			rp.doorButton.SetDisabled(true)
			rp.sleepButton.SetDisabled(false)
			rp.helpButton.SetDisabled(false)
			rp.holdButton.SetDisabled(true)
			rp.resumeButton.SetDisabled(true)
		case "Door":
			rp.homeButton.SetDisabled(true)
			rp.unlockButton.SetDisabled(true)
			rp.resetButton.SetDisabled(false)
			rp.checkButton.SetDisabled(true)
			rp.doorButton.SetDisabled(true)
			rp.sleepButton.SetDisabled(true)
			rp.helpButton.SetDisabled(true)
			rp.holdButton.SetDisabled(true)
			rp.resumeButton.SetDisabled(false)
		case "Check":
			rp.homeButton.SetDisabled(true)
			rp.unlockButton.SetDisabled(true)
			rp.resetButton.SetDisabled(false)
			rp.checkButton.SetDisabled(false)
			rp.doorButton.SetDisabled(true)
			rp.sleepButton.SetDisabled(false)
			rp.helpButton.SetDisabled(false)
			rp.holdButton.SetDisabled(true)
			rp.resumeButton.SetDisabled(true)
		case "Home":
			rp.homeButton.SetDisabled(true)
			rp.unlockButton.SetDisabled(true)
			rp.resetButton.SetDisabled(false)
			rp.checkButton.SetDisabled(true)
			rp.doorButton.SetDisabled(false)
			rp.sleepButton.SetDisabled(true)
			rp.helpButton.SetDisabled(true)
			rp.holdButton.SetDisabled(true)
			rp.resumeButton.SetDisabled(true)
		case "Sleep":
			rp.homeButton.SetDisabled(true)
			rp.unlockButton.SetDisabled(true)
			rp.resetButton.SetDisabled(false)
			rp.checkButton.SetDisabled(true)
			rp.doorButton.SetDisabled(true)
			rp.sleepButton.SetDisabled(true)
			rp.helpButton.SetDisabled(true)
			rp.holdButton.SetDisabled(true)
			rp.resumeButton.SetDisabled(true)
		default:
			panic(fmt.Errorf("unknown state: %s", rp.machineState.State))
		}
	})
}

func (rp *RootPrimitive) processMessagePushWelcome() {
	rp.app.QueueUpdateDraw(func() {
		rp.feedbackTextView.SetText("")
	})
}

func getMachineStateColor(state string) tcell.Color {
	switch state {
	case "Idle":
		return tcell.ColorBlack
	case "Run":
		return tcell.ColorGreen
	case "Hold":
		return tcell.ColorYellow
	case "Jog":
		return tcell.ColorDarkGreen
	case "Alarm":
		return tcell.ColorRed
	case "Door":
		return tcell.ColorOrange
	case "Check":
		return tcell.ColorDarkCyan
	case "Home":
		return tcell.ColorLightGreen
	case "Sleep":
		return tcell.ColorDarkBlue
	default:
		return tcell.ColorWhite
	}
}

func (rp *RootPrimitive) processMessagePushFeedback(
	messagePushFeedback *grblMod.MessagePushFeedback,
) {
	rp.feedbackTextView.SetText(messagePushFeedback.Text())
}

func (rp *RootPrimitive) processMessagePushStatusReport(
	statusReport *grblMod.MessagePushStatusReport,
) {
	rp.machineState = &statusReport.MachineState
	rp.updateDisabled()
}

func (rp *RootPrimitive) ProcessMessage(message grblMod.Message) {
	if _, ok := message.(*grblMod.MessagePushWelcome); ok {
		rp.processMessagePushWelcome()
		return
	}
	if messagePushFeedback, ok := message.(*grblMod.MessagePushFeedback); ok {
		rp.processMessagePushFeedback(messagePushFeedback)
		return
	}
	if messagePushStatusReport, ok := message.(*grblMod.MessagePushStatusReport); ok {
		rp.processMessagePushStatusReport(messagePushStatusReport)
		return
	}
}
