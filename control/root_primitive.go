package control

import (
	"fmt"
	"io"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	grblMod "github.com/fornellas/cgs/grbl"
)

type RootPrimitive struct {
	*tview.Flex
	app                *tview.Application
	grbl               *grblMod.Grbl
	controlPrimitive   *ControlPrimitive
	overridesPrimitive *OverridesPrimitive
	joggingPrimitive   *JoggingPrimitive
	logsPrimitive      *LogsPrimitive
	// Common
	feedbackTextView *tview.TextView
	stateTextView    *tview.TextView
	statusTextView   *tview.TextView
	homingButton     *tview.Button
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
	controlPrimitive *ControlPrimitive,
	overridesPrimitive *OverridesPrimitive,
	joggingPrimitive *JoggingPrimitive,
	logsPrimitive *LogsPrimitive,
) *RootPrimitive {
	rp := &RootPrimitive{
		grbl:               grbl,
		app:                app,
		controlPrimitive:   controlPrimitive,
		overridesPrimitive: overridesPrimitive,
		joggingPrimitive:   joggingPrimitive,
		logsPrimitive:      logsPrimitive,
	}

	rp.newFeedbackTextView()
	rp.newStateTextView()
	rp.newStatusTextView()
	rp.homingButton = tview.NewButton("Homing").
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

func (rp *RootPrimitive) newStateTextView() {
	textView := tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true).
		SetTextAlign(tview.AlignCenter).
		SetWrap(true)
	textView.SetBorder(true).SetTitle("State")
	textView.SetChangedFunc(func() {
		rp.app.Draw()
	})
	rp.stateTextView = textView
}

func (rp *RootPrimitive) newStatusTextView() {
	textView := tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true).
		SetWrap(true)
	textView.SetBorder(true).SetTitle("Status")
	textView.SetChangedFunc(func() {
		textView.ScrollToBeginning()
		rp.app.Draw()
	})
	rp.statusTextView = textView
}

func (rp *RootPrimitive) getButtonsFLex() *tview.Flex {
	commandButtonsFlex := tview.NewFlex()
	commandButtonsFlex.SetTitle("Commands")
	commandButtonsFlex.SetBorder(true)
	commandButtonsFlex.SetDirection(tview.FlexRow)
	commandButtonsFlex.AddItem(
		tview.NewFlex().SetDirection(tview.FlexColumn).
			AddItem(rp.homingButton, 0, 1, false).
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

	column1Flex := tview.NewFlex()
	column1Flex.SetDirection(tview.FlexRow)
	column1Flex.AddItem(rp.stateTextView, 4, 0, false)
	column1Flex.AddItem(rp.statusTextView, 0, 1, false)

	rootFlex := tview.NewFlex()
	rootFlex.SetDirection(tview.FlexColumn)
	rootFlex.AddItem(column0Flex, 0, 1, false)
	rootFlex.AddItem(column1Flex, 15, 0, false)

	rp.Flex = rootFlex
}

func (rp *RootPrimitive) updateDisabled() {
	rp.app.QueueUpdateDraw(func() {
		if rp.machineState == nil {
			rp.homingButton.SetDisabled(true)
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
			rp.homingButton.SetDisabled(false)
			rp.unlockButton.SetDisabled(true)
			rp.resetButton.SetDisabled(false)
			rp.checkButton.SetDisabled(false)
			rp.doorButton.SetDisabled(false)
			rp.sleepButton.SetDisabled(false)
			rp.helpButton.SetDisabled(false)
			rp.holdButton.SetDisabled(false)
			rp.resumeButton.SetDisabled(true)
		case "Run":
			rp.homingButton.SetDisabled(true)
			rp.unlockButton.SetDisabled(true)
			rp.resetButton.SetDisabled(false)
			rp.checkButton.SetDisabled(true)
			rp.doorButton.SetDisabled(false)
			rp.sleepButton.SetDisabled(true)
			rp.helpButton.SetDisabled(true)
			rp.holdButton.SetDisabled(false)
			rp.resumeButton.SetDisabled(true)
		case "Hold":
			rp.homingButton.SetDisabled(true)
			rp.unlockButton.SetDisabled(true)
			rp.resetButton.SetDisabled(false)
			rp.checkButton.SetDisabled(true)
			rp.doorButton.SetDisabled(false)
			rp.sleepButton.SetDisabled(true)
			rp.helpButton.SetDisabled(true)
			rp.holdButton.SetDisabled(true)
			rp.resumeButton.SetDisabled(false)
		case "Jog":
			rp.homingButton.SetDisabled(true)
			rp.unlockButton.SetDisabled(true)
			rp.resetButton.SetDisabled(false)
			rp.checkButton.SetDisabled(true)
			rp.doorButton.SetDisabled(false)
			rp.sleepButton.SetDisabled(true)
			rp.helpButton.SetDisabled(true)
			rp.holdButton.SetDisabled(false)
			rp.resumeButton.SetDisabled(true)
		case "Alarm":
			rp.homingButton.SetDisabled(false)
			rp.unlockButton.SetDisabled(false)
			rp.resetButton.SetDisabled(false)
			rp.checkButton.SetDisabled(true)
			rp.doorButton.SetDisabled(true)
			rp.sleepButton.SetDisabled(false)
			rp.helpButton.SetDisabled(false)
			rp.holdButton.SetDisabled(true)
			rp.resumeButton.SetDisabled(true)
		case "Door":
			rp.homingButton.SetDisabled(true)
			rp.unlockButton.SetDisabled(true)
			rp.resetButton.SetDisabled(false)
			rp.checkButton.SetDisabled(true)
			rp.doorButton.SetDisabled(true)
			rp.sleepButton.SetDisabled(true)
			rp.helpButton.SetDisabled(true)
			rp.holdButton.SetDisabled(true)
			rp.resumeButton.SetDisabled(false)
		case "Check":
			rp.homingButton.SetDisabled(true)
			rp.unlockButton.SetDisabled(true)
			rp.resetButton.SetDisabled(false)
			rp.checkButton.SetDisabled(false)
			rp.doorButton.SetDisabled(true)
			rp.sleepButton.SetDisabled(false)
			rp.helpButton.SetDisabled(false)
			rp.holdButton.SetDisabled(true)
			rp.resumeButton.SetDisabled(true)
		case "Home":
			rp.homingButton.SetDisabled(true)
			rp.unlockButton.SetDisabled(true)
			rp.resetButton.SetDisabled(false)
			rp.checkButton.SetDisabled(true)
			rp.doorButton.SetDisabled(false)
			rp.sleepButton.SetDisabled(true)
			rp.helpButton.SetDisabled(true)
			rp.holdButton.SetDisabled(true)
			rp.resumeButton.SetDisabled(true)
		case "Sleep":
			rp.homingButton.SetDisabled(true)
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
		rp.stateTextView.SetBackgroundColor(tview.Styles.PrimitiveBackgroundColor)
		rp.stateTextView.Clear()
		rp.statusTextView.Clear()
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

func (rp *RootPrimitive) updateStateTextView(state string, subState string) {
	stateColor := getMachineStateColor(state)

	rp.app.QueueUpdateDraw(func() {
		rp.stateTextView.Clear()
		rp.stateTextView.SetBackgroundColor(stateColor)
	})
	fmt.Fprintf(
		rp.stateTextView, "%s\n",
		tview.Escape(state),
	)
	if len(subState) > 0 {
		fmt.Fprintf(
			rp.stateTextView, "(%s)\n",
			tview.Escape(subState),
		)
	}
}

func (rp *RootPrimitive) processMessagePushAlarm() {
	rp.updateStateTextView("Alarm", "")
}

//gocyclo:ignore
func (rp *RootPrimitive) writePositionStatus(w io.Writer, statusReport *grblMod.MessagePushStatusReport) {
	var mx, my, mz, ma, wx, wy, wz, wa *float64
	if statusReport.MachinePosition != nil {
		mx = &statusReport.MachinePosition.X
		my = &statusReport.MachinePosition.Y
		mz = &statusReport.MachinePosition.Z
		ma = statusReport.MachinePosition.A
		if rp.grbl.GetWorkCoordinateOffset() != nil {
			wxv := statusReport.MachinePosition.X - rp.grbl.GetWorkCoordinateOffset().X
			wx = &wxv
			wyv := statusReport.MachinePosition.Y - rp.grbl.GetWorkCoordinateOffset().Y
			wy = &wyv
			wzv := statusReport.MachinePosition.Z - rp.grbl.GetWorkCoordinateOffset().Z
			wz = &wzv
			if statusReport.MachinePosition.A != nil && rp.grbl.GetWorkCoordinateOffset().A != nil {
				wav := *statusReport.MachinePosition.A - *rp.grbl.GetWorkCoordinateOffset().A
				wa = &wav
			}
		}
	}
	if statusReport.WorkPosition != nil {
		wx = &statusReport.WorkPosition.X
		wy = &statusReport.WorkPosition.Y
		wz = &statusReport.WorkPosition.Z
		wa = statusReport.WorkPosition.A
		if rp.grbl.GetWorkCoordinateOffset() != nil {
			mxv := statusReport.WorkPosition.X - rp.grbl.GetWorkCoordinateOffset().X
			mx = &mxv
			myv := statusReport.WorkPosition.Y - rp.grbl.GetWorkCoordinateOffset().Y
			my = &myv
			mzv := statusReport.WorkPosition.Z - rp.grbl.GetWorkCoordinateOffset().Z
			mz = &mzv
			if statusReport.WorkPosition.A != nil && rp.grbl.GetWorkCoordinateOffset().A != nil {
				mav := *statusReport.WorkPosition.A - *rp.grbl.GetWorkCoordinateOffset().A
				ma = &mav
			}
		}
	}
	var nl bool
	if wx != nil || wy != nil || wz != nil || wa != nil {
		fmt.Fprintf(w, "Work\n")
		nl = true
	}
	if wx != nil {
		fmt.Fprintf(w, "X:%.3f\n", *wx)
	}
	if wy != nil {
		fmt.Fprintf(w, "Y:%.3f\n", *wy)
	}
	if wz != nil {
		fmt.Fprintf(w, "Z:%.3f\n", *wz)
	}
	if wa != nil {
		fmt.Fprintf(w, "A:%.3f\n", *wa)
	}
	if mx != nil || my != nil || mz != nil || ma != nil {
		if nl {
			fmt.Fprintf(w, "\n")
		}
		fmt.Fprintf(w, "Machine\n")
	}
	if mx != nil {
		fmt.Fprintf(w, "X:%.3f\n", *mx)
	}
	if my != nil {
		fmt.Fprintf(w, "Y:%.3f\n", *my)
	}
	if mz != nil {
		fmt.Fprintf(w, "Z:%.3f\n", *mz)
	}
	if ma != nil {
		fmt.Fprintf(w, "A:%.3f\n", *ma)
	}
}

//gocyclo:ignore
func (rp *RootPrimitive) updateStatusTextView(
	statusReport *grblMod.MessagePushStatusReport,
) {
	rp.app.QueueUpdateDraw(func() {
		rp.statusTextView.Clear()
	})

	rp.writePositionStatus(rp.statusTextView, statusReport)

	if statusReport.BufferState != nil {
		fmt.Fprint(rp.statusTextView, "\nBuffer\n")
		fmt.Fprintf(rp.statusTextView, "Blocks:%d\n", statusReport.BufferState.AvailableBlocks)
		fmt.Fprintf(rp.statusTextView, "Bytes:%d\n", statusReport.BufferState.AvailableBytes)
	}

	if statusReport.LineNumber != nil {
		fmt.Fprintf(rp.statusTextView, "\nLine:%d\n", *statusReport.LineNumber)
	}

	if statusReport.Feed != nil {
		fmt.Fprintf(rp.statusTextView, "\nFeed:%.1f\n", *statusReport.Feed)
	}

	if statusReport.FeedSpindle != nil {
		fmt.Fprintf(rp.statusTextView, "\nFeed:%.0f\n", statusReport.FeedSpindle.Feed)
		fmt.Fprintf(rp.statusTextView, "Speed:%.0f\n", statusReport.FeedSpindle.Speed)
	}

	if statusReport.PinState != nil {
		fmt.Fprintf(rp.statusTextView, "\nPin:%s\n", statusReport.PinState)
	}

	if rp.grbl.GetOverrideValues() != nil {
		fmt.Fprint(rp.statusTextView, "\nOverrides\n")
		fmt.Fprintf(rp.statusTextView, "Feed:%.0f%%\n", rp.grbl.GetOverrideValues().Feed)
		fmt.Fprintf(rp.statusTextView, "Rapids:%.0f%%\n", rp.grbl.GetOverrideValues().Rapids)
		fmt.Fprintf(rp.statusTextView, "Spindle:%.0f%%\n", rp.grbl.GetOverrideValues().Spindle)
	}

	if statusReport.AccessoryState != nil {
		fmt.Fprint(rp.statusTextView, "\nAccessory\n")
		if statusReport.AccessoryState.SpindleCW != nil && *statusReport.AccessoryState.SpindleCW {
			fmt.Fprint(rp.statusTextView, "Spindle: CW")
		}
		if statusReport.AccessoryState.SpindleCCW != nil && *statusReport.AccessoryState.SpindleCCW {
			fmt.Fprint(rp.statusTextView, "Spindle: CCW")
		}
		if statusReport.AccessoryState.FloodCoolant != nil && *statusReport.AccessoryState.FloodCoolant {
			fmt.Fprint(rp.statusTextView, "Flood Coolant")
		}
		if statusReport.AccessoryState.MistCoolant != nil && *statusReport.AccessoryState.MistCoolant {
			fmt.Fprint(rp.statusTextView, "Mist Coolant")
		}
	}
}

func (rp *RootPrimitive) processMessagePushStatusReport(
	statusReport *grblMod.MessagePushStatusReport,
) {
	rp.updateStateTextView(statusReport.MachineState.State, statusReport.MachineState.SubStateString())
	rp.machineState = &statusReport.MachineState
	rp.updateDisabled()
	rp.updateStatusTextView(statusReport)
}

func (rp *RootPrimitive) processMessagePushFeedback(
	messagePushFeedback *grblMod.MessagePushFeedback,
) {
	rp.feedbackTextView.SetText(messagePushFeedback.Text())
}

func (rp *RootPrimitive) ProcessMessage(message grblMod.Message) {
	if _, ok := message.(*grblMod.MessagePushWelcome); ok {
		rp.processMessagePushWelcome()
		return
	}

	if _, ok := message.(*grblMod.MessagePushAlarm); ok {
		rp.processMessagePushAlarm()
		return
	}

	if messagePushStatusReport, ok := message.(*grblMod.MessagePushStatusReport); ok {
		rp.processMessagePushStatusReport(messagePushStatusReport)
		return
	}

	if messagePushFeedback, ok := message.(*grblMod.MessagePushFeedback); ok {
		rp.processMessagePushFeedback(messagePushFeedback)
		return
	}
}
