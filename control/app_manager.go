package control

import (
	"fmt"
	"io"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	grblMod "github.com/fornellas/cgs/grbl"
)

type AppManager struct {
	App                *tview.Application
	grbl               *grblMod.Grbl
	controlPrimitive   *ControlPrimitive
	overridesPrimitive *OverridesPrimitive
	// Common
	FeedbackTextView *tview.TextView
	StateTextView    *tview.TextView
	StatusTextView   *tview.TextView
	HomingButton     *tview.Button
	UnlockButton     *tview.Button
	ResetButton      *tview.Button
	CheckButton      *tview.Button
	DoorButton       *tview.Button
	SleepButton      *tview.Button
	HelpButton       *tview.Button
	HoldButton       *tview.Button
	ResumeButton     *tview.Button
	RootFlex         *tview.Flex

	machineState *grblMod.StatusReportMachineState
}

func NewAppManager(
	grbl *grblMod.Grbl,
	controlPrimitive *ControlPrimitive,
	overridesPrimitive *OverridesPrimitive,
) *AppManager {
	am := &AppManager{
		grbl:               grbl,
		controlPrimitive:   controlPrimitive,
		overridesPrimitive: overridesPrimitive,
	}

	tviewApp := tview.NewApplication()
	tviewApp.EnableMouse(true)
	tviewApp.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyCtrlX {
			am.controlPrimitive.QueueRealTimeCommand(grblMod.RealTimeCommandSoftReset)
			return nil
		}
		return event
	})
	am.App = tviewApp

	am.newFeedbackTextView()
	am.newStateTextView()
	am.newStatusTextView()
	am.HomingButton = tview.NewButton("Homing").
		SetSelectedFunc(func() { am.controlPrimitive.QueueCommand("$H") })
	am.UnlockButton = tview.NewButton("Unlock").
		SetSelectedFunc(func() { am.controlPrimitive.QueueCommand("$X") })
	am.ResetButton = tview.NewButton("Reset").
		SetSelectedFunc(func() { am.controlPrimitive.QueueRealTimeCommand(grblMod.RealTimeCommandSoftReset) })
	am.CheckButton = tview.NewButton("Check").
		SetSelectedFunc(func() { am.controlPrimitive.QueueCommand("$C") })
	am.DoorButton = tview.NewButton("Door").
		SetSelectedFunc(func() { am.controlPrimitive.QueueRealTimeCommand(grblMod.RealTimeCommandSafetyDoor) })
	am.SleepButton = tview.NewButton("Sleep").
		SetSelectedFunc(func() { am.controlPrimitive.QueueCommand("$SLP") })
	am.HelpButton = tview.NewButton("Help").
		SetSelectedFunc(func() { am.controlPrimitive.QueueCommand("$") })
	am.HoldButton = tview.NewButton("Hold").
		SetSelectedFunc(func() { am.controlPrimitive.QueueRealTimeCommand(grblMod.RealTimeCommandFeedHold) })
	am.ResumeButton = tview.NewButton("Resume").
		SetSelectedFunc(func() {
			am.controlPrimitive.QueueRealTimeCommand(grblMod.RealTimeCommandCycleStartResume)
		})

	am.newRootFlex()

	am.App.SetRoot(am.RootFlex, true)

	return am
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
	stream := am.getStreamPrimitive()
	script := am.getScriptPrimitive()
	settings := am.getSettingsPrimitive()

	page := tview.NewPages()
	page.AddPage("Control", am.controlPrimitive, true, true)
	page.AddPage("Jogging", jogging, true, true)
	page.AddPage("Overrides", am.overridesPrimitive, true, true)
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

func (am *AppManager) updateDisabled() {
	if am.machineState == nil {
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

func (am *AppManager) processMessagePushWelcome() {
	am.StateTextView.SetBackgroundColor(tview.Styles.PrimitiveBackgroundColor)
	am.StateTextView.Clear()
	am.StatusTextView.Clear()
	am.FeedbackTextView.SetText("")
}

func getMachineStateColor(state string) tcell.Color {
	switch state {
	case "Idle":
		return tcell.ColorGreen
	case "Run":
		return tcell.ColorLightCyan
	case "Hold":
		return tcell.ColorYellow
	case "Jog":
		return tcell.ColorBlue
	case "Alarm":
		return tcell.ColorRed
	case "Door":
		return tcell.ColorOrange
	case "Check":
		return tcell.ColorBlue
	case "Home":
		return tcell.ColorLime
	case "Sleep":
		return tcell.ColorSilver
	default:
		return tcell.ColorWhite
	}
}

func (am *AppManager) updateStateTextView(state string, subState string) {
	stateColor := getMachineStateColor(state)

	am.StateTextView.Clear()
	am.StateTextView.SetBackgroundColor(stateColor)
	fmt.Fprintf(
		am.StateTextView, "%s\n",
		tview.Escape(state),
	)
	if len(subState) > 0 {
		fmt.Fprintf(
			am.StateTextView, "(%s)\n",
			tview.Escape(subState),
		)
	}
}

func (am *AppManager) processMessagePushAlarm(messagePushAlarm *grblMod.MessagePushAlarm) {
	am.updateStateTextView("Alarm", "")
}

//gocyclo:ignore
func (am *AppManager) writePositionStatus(w io.Writer, statusReport *grblMod.MessagePushStatusReport) {
	var mx, my, mz, ma, wx, wy, wz, wa *float64
	if statusReport.MachinePosition != nil {
		mx = &statusReport.MachinePosition.X
		my = &statusReport.MachinePosition.Y
		mz = &statusReport.MachinePosition.Z
		ma = statusReport.MachinePosition.A
		if am.grbl.GetWorkCoordinateOffset() != nil {
			wxv := statusReport.MachinePosition.X - am.grbl.GetWorkCoordinateOffset().X
			wx = &wxv
			wyv := statusReport.MachinePosition.Y - am.grbl.GetWorkCoordinateOffset().Y
			wy = &wyv
			wzv := statusReport.MachinePosition.Z - am.grbl.GetWorkCoordinateOffset().Z
			wz = &wzv
			if statusReport.MachinePosition.A != nil && am.grbl.GetWorkCoordinateOffset().A != nil {
				wav := *statusReport.MachinePosition.A - *am.grbl.GetWorkCoordinateOffset().A
				wa = &wav
			}
		}
	}
	if statusReport.WorkPosition != nil {
		wx = &statusReport.WorkPosition.X
		wy = &statusReport.WorkPosition.Y
		wz = &statusReport.WorkPosition.Z
		wa = statusReport.WorkPosition.A
		if am.grbl.GetWorkCoordinateOffset() != nil {
			mxv := statusReport.WorkPosition.X - am.grbl.GetWorkCoordinateOffset().X
			mx = &mxv
			myv := statusReport.WorkPosition.Y - am.grbl.GetWorkCoordinateOffset().Y
			my = &myv
			mzv := statusReport.WorkPosition.Z - am.grbl.GetWorkCoordinateOffset().Z
			mz = &mzv
			if statusReport.WorkPosition.A != nil && am.grbl.GetWorkCoordinateOffset().A != nil {
				mav := *statusReport.WorkPosition.A - *am.grbl.GetWorkCoordinateOffset().A
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
func (am *AppManager) updateStatusTextView(
	statusReport *grblMod.MessagePushStatusReport,
) {
	am.StatusTextView.Clear()

	am.writePositionStatus(am.StatusTextView, statusReport)

	if statusReport.BufferState != nil {
		fmt.Fprint(am.StatusTextView, "\nBuffer\n")
		fmt.Fprintf(am.StatusTextView, "Blocks:%d\n", statusReport.BufferState.AvailableBlocks)
		fmt.Fprintf(am.StatusTextView, "Bytes:%d\n", statusReport.BufferState.AvailableBytes)
	}

	if statusReport.LineNumber != nil {
		fmt.Fprintf(am.StatusTextView, "\nLine:%d\n", *statusReport.LineNumber)
	}

	if statusReport.Feed != nil {
		fmt.Fprintf(am.StatusTextView, "\nFeed:%.1f\n", *statusReport.Feed)
	}

	if statusReport.FeedSpindle != nil {
		fmt.Fprintf(am.StatusTextView, "\nFeed:%.0f\n", statusReport.FeedSpindle.Feed)
		fmt.Fprintf(am.StatusTextView, "Speed:%.0f\n", statusReport.FeedSpindle.Speed)
	}

	if statusReport.PinState != nil {
		fmt.Fprintf(am.StatusTextView, "\nPin:%s\n", statusReport.PinState)
	}

	if am.grbl.GetOverrideValues() != nil {
		fmt.Fprint(am.StatusTextView, "\nOverrides\n")
		fmt.Fprintf(am.StatusTextView, "Feed:%.0f%%\n", am.grbl.GetOverrideValues().Feed)
		fmt.Fprintf(am.StatusTextView, "Rapids:%.0f%%\n", am.grbl.GetOverrideValues().Rapids)
		fmt.Fprintf(am.StatusTextView, "Spindle:%.0f%%\n", am.grbl.GetOverrideValues().Spindle)
	}

	if statusReport.AccessoryState != nil {
		fmt.Fprint(am.StatusTextView, "\nAccessory\n")
		if statusReport.AccessoryState.SpindleCW != nil && *statusReport.AccessoryState.SpindleCW {
			fmt.Fprint(am.StatusTextView, "Spindle: CW")
		}
		if statusReport.AccessoryState.SpindleCCW != nil && *statusReport.AccessoryState.SpindleCCW {
			fmt.Fprint(am.StatusTextView, "Spindle: CCW")
		}
		if statusReport.AccessoryState.FloodCoolant != nil && *statusReport.AccessoryState.FloodCoolant {
			fmt.Fprint(am.StatusTextView, "Flood Coolant")
		}
		if statusReport.AccessoryState.MistCoolant != nil && *statusReport.AccessoryState.MistCoolant {
			fmt.Fprint(am.StatusTextView, "Mist Coolant")
		}
	}
}

func (am *AppManager) processMessagePushStatusReport(
	statusReport *grblMod.MessagePushStatusReport,
) {
	am.updateStateTextView(statusReport.MachineState.State, statusReport.MachineState.SubStateString())
	am.machineState = &statusReport.MachineState
	am.updateDisabled()
	am.updateStatusTextView(statusReport)
}

func (am *AppManager) processMessagePushFeedback(
	messagePushFeedback *grblMod.MessagePushFeedback,
) {
	am.FeedbackTextView.SetText(messagePushFeedback.Text())
}

func (am *AppManager) ProcessMessage(message grblMod.Message) {
	if _, ok := message.(*grblMod.MessagePushWelcome); ok {
		am.processMessagePushWelcome()
		return
	}

	if messagePushAlarm, ok := message.(*grblMod.MessagePushAlarm); ok {
		am.processMessagePushAlarm(messagePushAlarm)
		return
	}

	if messagePushStatusReport, ok := message.(*grblMod.MessagePushStatusReport); ok {
		am.processMessagePushStatusReport(messagePushStatusReport)
		return
	}

	if messagePushFeedback, ok := message.(*grblMod.MessagePushFeedback); ok {
		am.processMessagePushFeedback(messagePushFeedback)
		return
	}
}
