package control

import (
	"bytes"
	"context"
	"fmt"
	"sync"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	grblMod "github.com/fornellas/cgs/grbl"
)

type SettingsPrimitive struct {
	*tview.Flex
	app              *tview.Application
	controlPrimitive *ControlPrimitive
	// Settings
	stepPulse           *tview.InputField
	stepIdleDelay       *tview.InputField
	stepPortInvert      *tview.InputField
	directionPortInvert *tview.InputField
	stepEnableInvert    *tview.Checkbox
	limitPinsInvert     *tview.Checkbox
	probePinInvert      *tview.Checkbox
	statusReport        *tview.InputField
	junctionDeviation   *tview.InputField
	arcTolerance        *tview.InputField
	reportInches        *tview.Checkbox
	softLimits          *tview.Checkbox
	hardLimits          *tview.Checkbox
	homingCycle         *tview.Checkbox
	homingDirInvert     *tview.InputField
	homingFeed          *tview.InputField
	homingSeek          *tview.InputField
	homingDebounce      *tview.InputField
	homingPullOff       *tview.InputField
	maxSpindleSpeed     *tview.InputField
	minSpindleSpeed     *tview.InputField
	laserMode           *tview.Checkbox
	xSteps              *tview.InputField
	ySteps              *tview.InputField
	zSteps              *tview.InputField
	xMaxRate            *tview.InputField
	yMaxRate            *tview.InputField
	zMaxRate            *tview.InputField
	xAcceleration       *tview.InputField
	yAcceleration       *tview.InputField
	zAcceleration       *tview.InputField
	xMaxTravel          *tview.InputField
	yMaxTravel          *tview.InputField
	zMaxTravel          *tview.InputField
	// Startup Lines
	startupLine0InputField *tview.InputField
	startupLine1InputField *tview.InputField
	// Build Info
	versionTextView            *tview.TextView
	infoInputField             *tview.InputField
	compileTimeOptionsTextView *tview.TextView
	// Restore Defaults
	restoreSettingsButton        *tview.Button
	restoreGcodeParametersButton *tview.Button
	restoreAllButton             *tview.Button
	// Messages
	machineState grblMod.StatusReportMachineState

	mu sync.Mutex
}

func NewSettingsPrimitive(
	ctx context.Context,
	app *tview.Application,
	controlPrimitive *ControlPrimitive,
) *SettingsPrimitive {
	sp := &SettingsPrimitive{
		app:              app,
		controlPrimitive: controlPrimitive,
	}

	// Settings: InputFields
	// $0 Step pulse, microseconds
	stepPulse := tview.NewInputField()
	stepPulse.SetLabel("Step pulse (us)")
	sp.stepPulse = stepPulse
	// $1 Step idle delay, milliseconds
	stepIdleDelay := tview.NewInputField()
	stepIdleDelay.SetLabel("Step idle delay (ms)")
	sp.stepIdleDelay = stepIdleDelay
	// $2 Step port invert, mask
	stepPortInvert := tview.NewInputField()
	stepPortInvert.SetLabel("Step port invert (mask)")
	sp.stepPortInvert = stepPortInvert
	// $3 Direction port invert, mask
	directionPortInvert := tview.NewInputField()
	directionPortInvert.SetLabel("Direction port invert (mask)")
	sp.directionPortInvert = directionPortInvert
	// $4 Step enable invert, boolean
	stepEnableInvert := tview.NewCheckbox()
	sp.stepEnableInvert = stepEnableInvert
	stepEnableInvert.SetLabel("Step enable invert")
	// $5  Limit pins invert, boolean
	limitPinsInvert := tview.NewCheckbox()
	sp.limitPinsInvert = limitPinsInvert
	limitPinsInvert.SetLabel("Limit pins invert")
	// $6  Probe pin invert, boolean
	probePinInvert := tview.NewCheckbox()
	sp.probePinInvert = probePinInvert
	probePinInvert.SetLabel("Probe pin invert")
	// $10 Status report, mask
	statusReport := tview.NewInputField()
	statusReport.SetLabel("Status report (mask)")
	sp.statusReport = statusReport
	// $11 Junction deviation, mm
	junctionDeviation := tview.NewInputField()
	junctionDeviation.SetLabel("Junction deviation (mm)")
	sp.junctionDeviation = junctionDeviation
	// $12 Arc tolerance, mm
	arcTolerance := tview.NewInputField()
	arcTolerance.SetLabel("Arc tolerance (mm)")
	sp.arcTolerance = arcTolerance
	// $13 Report inches, boolean
	reportInches := tview.NewCheckbox()
	sp.reportInches = reportInches
	reportInches.SetLabel("Report inches")
	// $20 Soft limits, boolean
	softLimits := tview.NewCheckbox()
	sp.softLimits = softLimits
	softLimits.SetLabel("Soft limits")
	// $21 Hard limits, boolean
	hardLimits := tview.NewCheckbox()
	sp.hardLimits = hardLimits
	hardLimits.SetLabel("Hard limits")
	// $22 Homing cycle, boolean
	homingCycle := tview.NewCheckbox()
	sp.homingCycle = homingCycle
	homingCycle.SetLabel("Homing cycle")
	// $23 Homing dir invert, mask
	homingDirInvert := tview.NewInputField()
	homingDirInvert.SetLabel("Homing dir invert (mask)")
	sp.homingDirInvert = homingDirInvert
	// $24 Homing feed, mm/min
	homingFeed := tview.NewInputField()
	homingFeed.SetLabel("Homing feed (mm/min)")
	sp.homingFeed = homingFeed
	// $25 Homing seek, mm/min
	homingSeek := tview.NewInputField()
	homingSeek.SetLabel("Homing seek (mm/min)")
	sp.homingSeek = homingSeek
	// $26 Homing debounce, milliseconds
	homingDebounce := tview.NewInputField()
	homingDebounce.SetLabel("Homing debounce (ms)")
	sp.homingDebounce = homingDebounce
	// $27 Homing pull-off, mm
	homingPullOff := tview.NewInputField()
	homingPullOff.SetLabel("Homing pull-off (mm)")
	sp.homingPullOff = homingPullOff
	// $30 Max spindle speed, RPM
	maxSpindleSpeed := tview.NewInputField()
	maxSpindleSpeed.SetLabel("Max spindle speed (RPM)")
	sp.maxSpindleSpeed = maxSpindleSpeed
	// $31 Min spindle speed, RPM
	minSpindleSpeed := tview.NewInputField()
	minSpindleSpeed.SetLabel("Min spindle speed (RPM)")
	sp.minSpindleSpeed = minSpindleSpeed
	// $32 Laser mode, boolean
	laserMode := tview.NewCheckbox()
	sp.laserMode = laserMode
	laserMode.SetLabel("Laser mode")
	// $100 X steps/mm
	xSteps := tview.NewInputField()
	xSteps.SetLabel("X (steps/mm)")
	sp.xSteps = xSteps
	// $101 Y steps/mm
	ySteps := tview.NewInputField()
	ySteps.SetLabel("Y (steps/mm)")
	sp.ySteps = ySteps
	// $102 Z steps/mm
	zSteps := tview.NewInputField()
	zSteps.SetLabel("Z (steps/mm)")
	sp.zSteps = zSteps
	// $110 X Max rate, mm/min
	xMaxRate := tview.NewInputField()
	xMaxRate.SetLabel("X Max rate (mm/min)")
	sp.xMaxRate = xMaxRate
	// $111 Y Max rate, mm/min
	yMaxRate := tview.NewInputField()
	yMaxRate.SetLabel("Y Max rate (mm/min)")
	sp.yMaxRate = yMaxRate
	// $112 Z Max rate, mm/min
	zMaxRate := tview.NewInputField()
	zMaxRate.SetLabel("Z Max rate (mm/min)")
	sp.zMaxRate = zMaxRate
	// $120 X Acceleration, mm/sec^2
	xAcceleration := tview.NewInputField()
	xAcceleration.SetLabel("X Acceleration (mm/sec^2)")
	sp.xAcceleration = xAcceleration
	// $121 Y Acceleration, mm/sec^2
	yAcceleration := tview.NewInputField()
	yAcceleration.SetLabel("Y Acceleration (mm/sec^2)")
	sp.yAcceleration = yAcceleration
	// $122 Z Acceleration, mm/sec^2
	zAcceleration := tview.NewInputField()
	zAcceleration.SetLabel("Z Acceleration (mm/sec^2)")
	sp.zAcceleration = zAcceleration
	// $130 X Max travel, mm
	xMaxTravel := tview.NewInputField()
	xMaxTravel.SetLabel("X Max travel (mm)")
	sp.xMaxTravel = xMaxTravel
	// $131 Y Max travel, mm
	yMaxTravel := tview.NewInputField()
	yMaxTravel.SetLabel("Y Max travel (mm)")
	sp.yMaxTravel = yMaxTravel
	// $132 Z Max travel, mm
	zMaxTravel := tview.NewInputField()
	zMaxTravel.SetLabel("Z Max travel (mm)")
	sp.zMaxTravel = zMaxTravel

	// Settings
	mainSettings := NewScrollContainer()
	mainSettings.SetBorder(true)
	mainSettings.SetTitle("Settings")
	mainSettings.AddPrimitive(stepPulse, 1)
	mainSettings.AddPrimitive(stepIdleDelay, 1)
	mainSettings.AddPrimitive(stepPortInvert, 1)
	mainSettings.AddPrimitive(directionPortInvert, 1)
	mainSettings.AddPrimitive(stepEnableInvert, 1)
	mainSettings.AddPrimitive(limitPinsInvert, 1)
	mainSettings.AddPrimitive(probePinInvert, 1)
	mainSettings.AddPrimitive(statusReport, 1)
	mainSettings.AddPrimitive(junctionDeviation, 1)
	mainSettings.AddPrimitive(arcTolerance, 1)
	mainSettings.AddPrimitive(reportInches, 1)
	mainSettings.AddPrimitive(softLimits, 1)
	mainSettings.AddPrimitive(hardLimits, 1)
	mainSettings.AddPrimitive(homingCycle, 1)
	mainSettings.AddPrimitive(homingDirInvert, 1)
	mainSettings.AddPrimitive(homingFeed, 1)
	mainSettings.AddPrimitive(homingSeek, 1)
	mainSettings.AddPrimitive(homingDebounce, 1)
	mainSettings.AddPrimitive(homingPullOff, 1)
	mainSettings.AddPrimitive(maxSpindleSpeed, 1)
	mainSettings.AddPrimitive(minSpindleSpeed, 1)
	mainSettings.AddPrimitive(laserMode, 1)
	mainSettings.AddPrimitive(xSteps, 1)
	mainSettings.AddPrimitive(ySteps, 1)
	mainSettings.AddPrimitive(zSteps, 1)
	mainSettings.AddPrimitive(xMaxRate, 1)
	mainSettings.AddPrimitive(yMaxRate, 1)
	mainSettings.AddPrimitive(zMaxRate, 1)
	mainSettings.AddPrimitive(xAcceleration, 1)
	mainSettings.AddPrimitive(yAcceleration, 1)
	mainSettings.AddPrimitive(zAcceleration, 1)
	mainSettings.AddPrimitive(xMaxTravel, 1)
	mainSettings.AddPrimitive(yMaxTravel, 1)
	mainSettings.AddPrimitive(zMaxTravel, 1)

	// Startup Lines: Input Fields
	startupLine0InputField := tview.NewInputField()
	startupLine0InputField.SetLabel("0")
	startupLine0InputField.SetDoneFunc(func(tcell.Key) {
		sp.controlPrimitive.QueueCommand(fmt.Sprintf("$N0=%s", startupLine0InputField.GetText()))
	})
	sp.startupLine0InputField = startupLine0InputField
	startupLine1InputField := tview.NewInputField()
	startupLine1InputField.SetLabel("1")
	startupLine1InputField.SetDoneFunc(func(tcell.Key) {
		sp.controlPrimitive.QueueCommand(fmt.Sprintf("$N1=%s", startupLine1InputField.GetText()))
	})
	sp.startupLine1InputField = startupLine1InputField

	// Startup Lines
	startupLinesFlex := tview.NewFlex()
	startupLinesFlex.SetDirection(tview.FlexRow)
	startupLinesFlex.SetBorder(true)
	startupLinesFlex.SetTitle("Startup Lines")
	startupLinesFlex.AddItem(startupLine0InputField, 0, 1, false)
	startupLinesFlex.AddItem(startupLine1InputField, 0, 1, false)

	// Build Info: Primitives
	versionTextView := tview.NewTextView()
	versionTextView.SetLabel("Version")
	sp.versionTextView = versionTextView
	infoInputField := tview.NewInputField()
	infoInputField.SetLabel("Info")
	infoInputField.SetDoneFunc(func(tcell.Key) {
		sp.controlPrimitive.QueueCommand(fmt.Sprintf("$I=%s", infoInputField.GetText()))
	})
	sp.infoInputField = infoInputField
	compileTimeOptionsTextView := tview.NewTextView()
	compileTimeOptionsTextView.SetDynamicColors(true)
	sp.compileTimeOptionsTextView = compileTimeOptionsTextView

	// Build Info
	buildInfoFlex := NewScrollContainer()
	buildInfoFlex.SetBorder(true)
	buildInfoFlex.SetTitle("Build Info")
	buildInfoFlex.AddPrimitive(sp.versionTextView, 1)
	buildInfoFlex.AddPrimitive(sp.infoInputField, 1)
	buildInfoFlex.AddPrimitive(sp.compileTimeOptionsTextView, 10)

	// Restore Defaults: Buttons
	restoreSettingsButton := tview.NewButton("Settings")
	restoreSettingsButton.SetSelectedFunc(func() {
		sp.controlPrimitive.QueueCommand("$RST=$")
	})
	sp.restoreSettingsButton = restoreSettingsButton
	restoreGcodeParametersButton := tview.NewButton("G-Code Parameters")
	restoreGcodeParametersButton.SetSelectedFunc(func() {
		sp.controlPrimitive.QueueCommand("$RST=#")
	})
	sp.restoreGcodeParametersButton = restoreGcodeParametersButton
	restoreAllButton := tview.NewButton("All")
	restoreAllButton.SetSelectedFunc(func() {
		sp.controlPrimitive.QueueCommand("$RST=*")
	})
	sp.restoreAllButton = restoreAllButton

	// Restore Defaults
	restoreDefaultsFlex := tview.NewFlex()
	restoreDefaultsFlex.SetBorderColor(tcell.ColorRed)
	restoreDefaultsFlex.SetBorder(true)
	restoreDefaultsFlex.SetTitle("Restore Defaults")
	restoreDefaultsFlex.SetDirection(tview.FlexRow)
	restoreDefaultsFlex.AddItem(sp.restoreSettingsButton, 0, 1, false)
	restoreDefaultsFlex.AddItem(sp.restoreGcodeParametersButton, 0, 1, false)
	restoreDefaultsFlex.AddItem(sp.restoreAllButton, 0, 1, false)

	otherSettingsFlex := tview.NewFlex()
	otherSettingsFlex.SetDirection(tview.FlexRow)
	otherSettingsFlex.AddItem(startupLinesFlex, 4, 0, false)
	otherSettingsFlex.AddItem(buildInfoFlex, 0, 1, false)
	otherSettingsFlex.AddItem(restoreDefaultsFlex, 5, 0, false)

	settingsRootFlex := tview.NewFlex()
	settingsRootFlex.SetBorder(true)
	settingsRootFlex.SetTitle("Settings")
	settingsRootFlex.SetDirection(tview.FlexColumn)
	settingsRootFlex.AddItem(mainSettings, 0, 1, false)
	settingsRootFlex.AddItem(otherSettingsFlex, 0, 1, false)
	sp.Flex = settingsRootFlex

	return sp
}

func (sp *SettingsPrimitive) processMessagePushWelcome() {
	sp.app.QueueUpdateDraw(func() {
		// Settings
		sp.stepPulse.SetText("")
		sp.stepIdleDelay.SetText("")
		sp.stepPortInvert.SetText("")
		sp.directionPortInvert.SetText("")
		sp.stepEnableInvert.SetChecked(false)
		sp.limitPinsInvert.SetChecked(false)
		sp.probePinInvert.SetChecked(false)
		sp.statusReport.SetText("")
		sp.junctionDeviation.SetText("")
		sp.arcTolerance.SetText("")
		sp.reportInches.SetChecked(false)
		sp.softLimits.SetChecked(false)
		sp.hardLimits.SetChecked(false)
		sp.homingCycle.SetChecked(false)
		sp.homingDirInvert.SetText("")
		sp.homingFeed.SetText("")
		sp.homingSeek.SetText("")
		sp.homingDebounce.SetText("")
		sp.homingPullOff.SetText("")
		sp.maxSpindleSpeed.SetText("")
		sp.minSpindleSpeed.SetText("")
		sp.laserMode.SetChecked(false)
		sp.xSteps.SetText("")
		sp.ySteps.SetText("")
		sp.zSteps.SetText("")
		sp.xMaxRate.SetText("")
		sp.yMaxRate.SetText("")
		sp.zMaxRate.SetText("")
		sp.xAcceleration.SetText("")
		sp.yAcceleration.SetText("")
		sp.zAcceleration.SetText("")
		sp.xMaxTravel.SetText("")
		sp.yMaxTravel.SetText("")
		sp.zMaxTravel.SetText("")
		// Startup Lines
		sp.startupLine0InputField.SetText("")
		sp.startupLine1InputField.SetText("")
		// Build Info
		sp.infoInputField.SetText("")
	})
}

func (sp *SettingsPrimitive) processMessagePushVersion(messagePushVersion *grblMod.MessagePushVersion) {
	sp.app.QueueUpdateDraw(func() {
		versionText := tview.Escape(messagePushVersion.Version)
		if versionText != sp.versionTextView.GetText(false) {
			sp.versionTextView.SetText(versionText)
		}
		infoText := tview.Escape(messagePushVersion.Info)
		if infoText != sp.infoInputField.GetText() {
			sp.infoInputField.SetText(infoText)
		}
	})
}

func (sp *SettingsPrimitive) processMessagePushCompileTimeOptions(messagePushCompileTimeOptions *grblMod.MessagePushCompileTimeOptions) {
	var buf bytes.Buffer

	fmt.Fprintf(&buf, "[%s]Compile Time Options[-]\n", tcell.ColorYellow)
	for _, opt := range messagePushCompileTimeOptions.CompileTimeOptions {
		fmt.Fprintf(&buf, "[%s]%s[-]\n", tcell.ColorWhite, tview.Escape(opt))
	}
	fmt.Fprintf(&buf, "[%s]Planner Blocks[-]%d\n", tcell.ColorYellow, messagePushCompileTimeOptions.PlannerBlocks)
	fmt.Fprintf(&buf, "[%s]Serial RX buffer bytes[-]%d\n", tcell.ColorYellow, messagePushCompileTimeOptions.SerialRxBufferBytes)

	sp.app.QueueUpdateDraw(func() {
		if buf.String() == sp.compileTimeOptionsTextView.GetText(false) {
			return
		}
		sp.compileTimeOptionsTextView.SetText(buf.String())
	})
}

//gocyclo:ignore
func (sp *SettingsPrimitive) processMessagePushSetting(messagePushSetting *grblMod.MessagePushSetting) {
	sp.app.QueueUpdateDraw(func() {
		switch messagePushSetting.Key {
		// Settings
		case "0":
			sp.stepPulse.SetText(messagePushSetting.Value)
		case "1":
			sp.stepIdleDelay.SetText(messagePushSetting.Value)
		case "2":
			sp.stepPortInvert.SetText(messagePushSetting.Value)
		case "3":
			sp.directionPortInvert.SetText(messagePushSetting.Value)
		case "4":
			sp.stepEnableInvert.SetChecked(messagePushSetting.Value != "0")
		case "5":
			sp.limitPinsInvert.SetChecked(messagePushSetting.Value != "0")
		case "6":
			sp.probePinInvert.SetChecked(messagePushSetting.Value != "0")
		case "10":
			sp.statusReport.SetText(messagePushSetting.Value)
		case "11":
			sp.junctionDeviation.SetText(messagePushSetting.Value)
		case "12":
			sp.arcTolerance.SetText(messagePushSetting.Value)
		case "13":
			sp.reportInches.SetChecked(messagePushSetting.Value != "0")
		case "20":
			sp.softLimits.SetChecked(messagePushSetting.Value != "0")
		case "21":
			sp.hardLimits.SetChecked(messagePushSetting.Value != "0")
		case "22":
			sp.homingCycle.SetChecked(messagePushSetting.Value != "0")
		case "23":
			sp.homingDirInvert.SetText(messagePushSetting.Value)
		case "24":
			sp.homingFeed.SetText(messagePushSetting.Value)
		case "25":
			sp.homingSeek.SetText(messagePushSetting.Value)
		case "26":
			sp.homingDebounce.SetText(messagePushSetting.Value)
		case "27":
			sp.homingPullOff.SetText(messagePushSetting.Value)
		case "30":
			sp.maxSpindleSpeed.SetText(messagePushSetting.Value)
		case "31":
			sp.minSpindleSpeed.SetText(messagePushSetting.Value)
		case "32":
			sp.laserMode.SetChecked(messagePushSetting.Value != "0")
		case "100":
			sp.xSteps.SetText(messagePushSetting.Value)
		case "101":
			sp.ySteps.SetText(messagePushSetting.Value)
		case "102":
			sp.zSteps.SetText(messagePushSetting.Value)
		case "110":
			sp.xMaxRate.SetText(messagePushSetting.Value)
		case "111":
			sp.yMaxRate.SetText(messagePushSetting.Value)
		case "112":
			sp.zMaxRate.SetText(messagePushSetting.Value)
		case "120":
			sp.xAcceleration.SetText(messagePushSetting.Value)
		case "121":
			sp.yAcceleration.SetText(messagePushSetting.Value)
		case "122":
			sp.zAcceleration.SetText(messagePushSetting.Value)
		case "130":
			sp.xMaxTravel.SetText(messagePushSetting.Value)
		case "131":
			sp.yMaxTravel.SetText(messagePushSetting.Value)
		case "132":
			sp.zMaxTravel.SetText(messagePushSetting.Value)
		// Startup Lines
		case "N0":
			sp.startupLine0InputField.SetText(messagePushSetting.Value)
		case "N1":
			sp.startupLine1InputField.SetText(messagePushSetting.Value)
		}
	})
}

func (sp *SettingsPrimitive) updateDisabled() {
	sp.mu.Lock()
	disabled := sp.machineState.State != "Idle"

	// Settings
	sp.stepPulse.SetDisabled(disabled)
	sp.stepIdleDelay.SetDisabled(disabled)
	sp.stepPortInvert.SetDisabled(disabled)
	sp.directionPortInvert.SetDisabled(disabled)
	sp.stepEnableInvert.SetDisabled(disabled)
	sp.limitPinsInvert.SetDisabled(disabled)
	sp.probePinInvert.SetDisabled(disabled)
	sp.statusReport.SetDisabled(disabled)
	sp.junctionDeviation.SetDisabled(disabled)
	sp.arcTolerance.SetDisabled(disabled)
	sp.reportInches.SetDisabled(disabled)
	sp.softLimits.SetDisabled(disabled)
	sp.hardLimits.SetDisabled(disabled)
	sp.homingCycle.SetDisabled(disabled)
	sp.homingDirInvert.SetDisabled(disabled)
	sp.homingFeed.SetDisabled(disabled)
	sp.homingSeek.SetDisabled(disabled)
	sp.homingDebounce.SetDisabled(disabled)
	sp.homingPullOff.SetDisabled(disabled)
	sp.maxSpindleSpeed.SetDisabled(disabled)
	sp.minSpindleSpeed.SetDisabled(disabled)
	sp.laserMode.SetDisabled(disabled)
	sp.xSteps.SetDisabled(disabled)
	sp.ySteps.SetDisabled(disabled)
	sp.zSteps.SetDisabled(disabled)
	sp.xMaxRate.SetDisabled(disabled)
	sp.yMaxRate.SetDisabled(disabled)
	sp.zMaxRate.SetDisabled(disabled)
	sp.xAcceleration.SetDisabled(disabled)
	sp.yAcceleration.SetDisabled(disabled)
	sp.zAcceleration.SetDisabled(disabled)
	sp.xMaxTravel.SetDisabled(disabled)
	sp.yMaxTravel.SetDisabled(disabled)
	sp.zMaxTravel.SetDisabled(disabled)
	// Startup Lines
	sp.startupLine0InputField.SetDisabled(disabled)
	sp.startupLine1InputField.SetDisabled(disabled)
	// Build Info
	sp.infoInputField.SetDisabled(disabled)
	// Restore Defaults
	sp.restoreSettingsButton.SetDisabled(disabled)
	sp.restoreGcodeParametersButton.SetDisabled(disabled)
	sp.restoreAllButton.SetDisabled(disabled)

	sp.mu.Unlock()
}

func (sp *SettingsPrimitive) setMachineState(machineState grblMod.StatusReportMachineState) {
	if sp.machineState == machineState {
		return
	}

	sp.mu.Lock()
	sp.machineState = machineState
	sp.mu.Unlock()

	sp.app.QueueUpdateDraw(func() {
		sp.updateDisabled()
	})
}

func (sp *SettingsPrimitive) processMessagePushStatusReport(
	messagePushStatusReport *grblMod.MessagePushStatusReport,
) {
	sp.setMachineState(messagePushStatusReport.MachineState)
}

func (sp *SettingsPrimitive) ProcessMessage(ctx context.Context, message grblMod.Message) {
	if _, ok := message.(*grblMod.MessagePushWelcome); ok {
		sp.processMessagePushWelcome()
		return
	}
	if messagePushVersion, ok := message.(*grblMod.MessagePushVersion); ok {
		sp.processMessagePushVersion(messagePushVersion)
		return
	}
	if messagePushCompileTimeOptions, ok := message.(*grblMod.MessagePushCompileTimeOptions); ok {
		sp.processMessagePushCompileTimeOptions(messagePushCompileTimeOptions)
		return
	}
	if messagePushSetting, ok := message.(*grblMod.MessagePushSetting); ok {
		sp.processMessagePushSetting(messagePushSetting)
		return
	}
	if messagePushStatusReport, ok := message.(*grblMod.MessagePushStatusReport); ok {
		sp.processMessagePushStatusReport(messagePushStatusReport)
		return
	}
}
