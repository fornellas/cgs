package control

import (
	"context"
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	grblMod "github.com/fornellas/cgs/grbl"
)

type RootPrimitive struct {
	*tview.Flex
	app                *tview.Application
	statusPrimitive    *StatusPrimitive
	controlPrimitive   *ControlPrimitive
	joggingPrimitive   *JoggingPrimitive
	probePrimitive     *ProbePrimitive
	overridesPrimitive *OverridesPrimitive
	streamPrimitive    *StreamPrimitive
	settingsPrimitive  *SettingsPrimitive
	logsPrimitive      *LogsPrimitive
	infoTextView       *tview.TextView
	homeButton         *tview.Button
	unlockButton       *tview.Button
	resetButton        *tview.Button
	checkButton        *tview.Button
	doorButton         *tview.Button
	sleepButton        *tview.Button
	helpButton         *tview.Button
	holdButton         *tview.Button
	resumeButton       *tview.Button
}

func NewRootPrimitive(
	ctx context.Context,
	app *tview.Application,
	statusPrimitive *StatusPrimitive,
	controlPrimitive *ControlPrimitive,
	joggingPrimitive *JoggingPrimitive,
	probePrimitive *ProbePrimitive,
	overridesPrimitive *OverridesPrimitive,
	streamPrimitive *StreamPrimitive,
	settingsPrimitive *SettingsPrimitive,
	logsPrimitive *LogsPrimitive,
) *RootPrimitive {
	rp := &RootPrimitive{
		app:                app,
		statusPrimitive:    statusPrimitive,
		controlPrimitive:   controlPrimitive,
		joggingPrimitive:   joggingPrimitive,
		probePrimitive:     probePrimitive,
		overridesPrimitive: overridesPrimitive,
		streamPrimitive:    streamPrimitive,
		settingsPrimitive:  settingsPrimitive,
		logsPrimitive:      logsPrimitive,
	}

	getButtonText := func(name, command string) string {
		return fmt.Sprintf("%s[lightblue]%s[-]", name, command)
	}

	rp.newInfoTextView()
	rp.homeButton = tview.NewButton(getButtonText("Home", grblMod.GrblCommandRunHomingCycle)).SetSelectedFunc(func() {
		rp.controlPrimitive.QueueCommand(grblMod.GrblCommandRunHomingCycle)
	}).SetDisabled(true)
	rp.unlockButton = tview.NewButton(getButtonText("Unlock", grblMod.GrblCommandKillAlarmLock)).SetSelectedFunc(func() {
		rp.controlPrimitive.QueueCommand(grblMod.GrblCommandKillAlarmLock)
	}).SetDisabled(true)
	rp.resetButton = tview.NewButton(getButtonText("Reset", "^X")).SetSelectedFunc(func() {
		rp.controlPrimitive.QueueRealTimeCommand(grblMod.RealTimeCommandSoftReset)
	}).SetDisabled(true)
	rp.checkButton = tview.NewButton(getButtonText("Check", grblMod.GrblCommandCheckGcodeMode)).SetSelectedFunc(func() {
		rp.controlPrimitive.QueueCommand(grblMod.GrblCommandCheckGcodeMode)
	}).SetDisabled(true)
	rp.doorButton = tview.NewButton(getButtonText("Door", fmt.Sprintf("0x%x", int(grblMod.RealTimeCommandSafetyDoor)))).SetSelectedFunc(func() {
		rp.controlPrimitive.QueueRealTimeCommand(grblMod.RealTimeCommandSafetyDoor)
	}).SetDisabled(true)
	rp.sleepButton = tview.NewButton(getButtonText("Sleep", grblMod.GrblCommandEnableSleepMode)).SetSelectedFunc(func() {
		rp.controlPrimitive.QueueCommand(grblMod.GrblCommandEnableSleepMode)
	}).SetDisabled(true)
	rp.helpButton = tview.NewButton(getButtonText("Help", grblMod.GrblCommandHelp)).SetSelectedFunc(func() {
		rp.controlPrimitive.QueueCommand(grblMod.GrblCommandHelp)
	}).SetDisabled(true)
	rp.holdButton = tview.NewButton(getButtonText("Hold", string(grblMod.RealTimeCommandFeedHold))).SetSelectedFunc(func() {
		rp.controlPrimitive.QueueRealTimeCommand(grblMod.RealTimeCommandFeedHold)
	}).SetDisabled(true)
	rp.resumeButton = tview.NewButton(getButtonText("Resume", string(grblMod.RealTimeCommandCycleStartResume))).SetSelectedFunc(func() {
		rp.controlPrimitive.QueueRealTimeCommand(grblMod.RealTimeCommandCycleStartResume)
	}).SetDisabled(true)

	rp.newRootFlex()

	return rp
}

func (rp *RootPrimitive) newInfoTextView() {
	textView := tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true).
		SetWrap(true)
	textView.SetChangedFunc(func() {})
	rp.infoTextView = textView
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

func (rp *RootPrimitive) getScriptPrimitive() tview.Primitive {
	return tview.NewBox().SetBorder(true).SetTitle("Script")
}

func (rp *RootPrimitive) getMainFlex() *tview.Flex {
	script := rp.getScriptPrimitive()

	page := tview.NewPages()
	page.AddPage("Control", rp.controlPrimitive, true, true)
	page.AddPage("Jogging", rp.joggingPrimitive, true, true)
	page.AddPage("Probe", rp.probePrimitive, true, true)
	page.AddPage("Overrides", rp.overridesPrimitive, true, true)
	page.AddPage("Stream", rp.streamPrimitive, true, true)
	page.AddPage("Script", script, true, true)
	page.AddPage("Settings", rp.settingsPrimitive, true, true)
	page.AddPage("Logs", rp.logsPrimitive, true, true)

	buttonsFlex := tview.NewFlex()
	buttonsFlex.SetDirection(tview.FlexColumn)
	buttonsFlex.AddItem(tview.NewButton("Control").SetSelectedFunc(func() {
		page.SwitchToPage("Control")
	}), 0, 1, false)
	buttonsFlex.AddItem(tview.NewButton("Jogging").SetSelectedFunc(func() {
		page.SwitchToPage("Jogging")
	}), 0, 1, false)
	buttonsFlex.AddItem(tview.NewButton("Probe").SetSelectedFunc(func() {
		page.SwitchToPage("Probe")
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
	column0Flex.AddItem(rp.infoTextView, 1, 0, false)
	column0Flex.AddItem(rp.getButtonsFLex(), 4, 0, false)

	rootFlex := tview.NewFlex()
	rootFlex.SetDirection(tview.FlexColumn)
	rootFlex.AddItem(column0Flex, 0, 1, false)
	rootFlex.AddItem(rp.statusPrimitive, rp.statusPrimitive.FixedSize()+1, 0, false)

	rp.Flex = rootFlex
}

func (rp *RootPrimitive) processTrackedState(trackedState *TrackedState) {
	rp.app.QueueUpdateDraw(func() {
		switch trackedState.State {
		case grblMod.StateIdle:
			rp.homeButton.SetDisabled(false)
			rp.unlockButton.SetDisabled(true)
			rp.resetButton.SetDisabled(false)
			rp.checkButton.SetDisabled(false)
			rp.doorButton.SetDisabled(false)
			rp.sleepButton.SetDisabled(false)
			rp.helpButton.SetDisabled(false)
			rp.holdButton.SetDisabled(false)
			rp.resumeButton.SetDisabled(true)
		case grblMod.StateRun:
			rp.homeButton.SetDisabled(true)
			rp.unlockButton.SetDisabled(true)
			rp.resetButton.SetDisabled(false)
			rp.checkButton.SetDisabled(true)
			rp.doorButton.SetDisabled(false)
			rp.sleepButton.SetDisabled(true)
			rp.helpButton.SetDisabled(true)
			rp.holdButton.SetDisabled(false)
			rp.resumeButton.SetDisabled(true)
		case grblMod.StateHold:
			rp.homeButton.SetDisabled(true)
			rp.unlockButton.SetDisabled(true)
			rp.resetButton.SetDisabled(false)
			rp.checkButton.SetDisabled(true)
			rp.doorButton.SetDisabled(false)
			rp.sleepButton.SetDisabled(true)
			rp.helpButton.SetDisabled(true)
			rp.holdButton.SetDisabled(true)
			rp.resumeButton.SetDisabled(false)
		case grblMod.StateJog:
			rp.homeButton.SetDisabled(true)
			rp.unlockButton.SetDisabled(true)
			rp.resetButton.SetDisabled(false)
			rp.checkButton.SetDisabled(true)
			rp.doorButton.SetDisabled(false)
			rp.sleepButton.SetDisabled(true)
			rp.helpButton.SetDisabled(true)
			rp.holdButton.SetDisabled(false)
			rp.resumeButton.SetDisabled(true)
		case grblMod.StateAlarm:
			rp.homeButton.SetDisabled(false)
			rp.unlockButton.SetDisabled(false)
			rp.resetButton.SetDisabled(false)
			rp.checkButton.SetDisabled(true)
			rp.doorButton.SetDisabled(true)
			rp.sleepButton.SetDisabled(false)
			rp.helpButton.SetDisabled(false)
			rp.holdButton.SetDisabled(true)
			rp.resumeButton.SetDisabled(true)
			if trackedState.Error != nil {
				rp.infoTextView.SetText(fmt.Sprintf("[%s]%s[-]", tcell.ColorRed, tview.Escape(trackedState.Error.Error())))
			}
		case grblMod.StateDoor:
			rp.homeButton.SetDisabled(true)
			rp.unlockButton.SetDisabled(true)
			rp.resetButton.SetDisabled(false)
			rp.checkButton.SetDisabled(true)
			rp.doorButton.SetDisabled(true)
			rp.sleepButton.SetDisabled(true)
			rp.helpButton.SetDisabled(true)
			rp.holdButton.SetDisabled(true)
			rp.resumeButton.SetDisabled(false)
		case grblMod.StateCheck:
			rp.homeButton.SetDisabled(true)
			rp.unlockButton.SetDisabled(true)
			rp.resetButton.SetDisabled(false)
			rp.checkButton.SetDisabled(false)
			rp.doorButton.SetDisabled(true)
			rp.sleepButton.SetDisabled(false)
			rp.helpButton.SetDisabled(false)
			rp.holdButton.SetDisabled(true)
			rp.resumeButton.SetDisabled(true)
		case grblMod.StateHome:
			rp.homeButton.SetDisabled(true)
			rp.unlockButton.SetDisabled(true)
			rp.resetButton.SetDisabled(false)
			rp.checkButton.SetDisabled(true)
			rp.doorButton.SetDisabled(false)
			rp.sleepButton.SetDisabled(true)
			rp.helpButton.SetDisabled(true)
			rp.holdButton.SetDisabled(true)
			rp.resumeButton.SetDisabled(true)
		case grblMod.StateSleep:
			rp.homeButton.SetDisabled(true)
			rp.unlockButton.SetDisabled(true)
			rp.resetButton.SetDisabled(false)
			rp.checkButton.SetDisabled(true)
			rp.doorButton.SetDisabled(true)
			rp.sleepButton.SetDisabled(true)
			rp.helpButton.SetDisabled(true)
			rp.holdButton.SetDisabled(true)
			rp.resumeButton.SetDisabled(true)
		case grblMod.StateUnknown:
			rp.homeButton.SetDisabled(true)
			rp.unlockButton.SetDisabled(true)
			rp.resetButton.SetDisabled(true)
			rp.checkButton.SetDisabled(true)
			rp.doorButton.SetDisabled(true)
			rp.sleepButton.SetDisabled(true)
			rp.helpButton.SetDisabled(true)
			rp.holdButton.SetDisabled(true)
			rp.resumeButton.SetDisabled(true)
			rp.infoTextView.SetText("")
		default:
			panic(fmt.Errorf("unknown state: %s", trackedState.State))
		}
	})
}

func (rp *RootPrimitive) processFeedbackPushMessage(
	feedbackPushMessage *grblMod.FeedbackPushMessage,
) {
	rp.app.QueueUpdateDraw(func() {
		text := tview.Escape(feedbackPushMessage.Text())
		if text == rp.infoTextView.GetText(false) {
			return
		}
		rp.infoTextView.SetText(text)
	})

	if feedbackPushMessage.Text() == "Reset to continue" {
		rp.app.QueueUpdateDraw(func() {
			rp.homeButton.SetDisabled(true)
			rp.unlockButton.SetDisabled(true)
			rp.resetButton.SetDisabled(false)
			rp.checkButton.SetDisabled(true)
			rp.doorButton.SetDisabled(true)
			rp.sleepButton.SetDisabled(true)
			rp.helpButton.SetDisabled(true)
			rp.holdButton.SetDisabled(true)
			rp.resumeButton.SetDisabled(true)
		})
	}
}

func (rp *RootPrimitive) Worker(
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
			if feedbackPushMessage, ok := pushMessage.(*grblMod.FeedbackPushMessage); ok {
				rp.processFeedbackPushMessage(feedbackPushMessage)
			}
		case trackedState, ok := <-trackedStateCh:
			if !ok {
				return fmt.Errorf("tracked state channel closed")
			}
			rp.processTrackedState(trackedState)
		}
	}
}
