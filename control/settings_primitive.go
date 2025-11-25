package control

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
	"sync"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	grblMod "github.com/fornellas/cgs/grbl"
)

func maskToCheckboxes(mask int) (x, y, z bool) {
	return mask&1 != 0, mask&2 != 0, mask&4 != 0
}

func checkboxesToMask(x, y, z bool) int {
	mask := 0
	if x {
		mask |= 1
	}
	if y {
		mask |= 2
	}
	if z {
		mask |= 4
	}
	return mask
}

type SettingsPrimitive struct {
	*tview.Flex
	app              *tview.Application
	controlPrimitive *ControlPrimitive
	// Settings
	stepPulse            *tview.InputField
	stepIdleDelay        *tview.InputField
	stepPortInvertX      *tview.Checkbox
	stepPortInvertY      *tview.Checkbox
	stepPortInvertZ      *tview.Checkbox
	directionPortInvertX *tview.Checkbox
	directionPortInvertY *tview.Checkbox
	directionPortInvertZ *tview.Checkbox
	stepEnableInvert     *tview.Checkbox
	limitPinsInvert      *tview.Checkbox
	probePinInvert       *tview.Checkbox
	statusReport         *tview.InputField
	junctionDeviation    *tview.InputField
	arcTolerance         *tview.InputField
	reportInches         *tview.Checkbox
	softLimits           *tview.Checkbox
	hardLimits           *tview.Checkbox
	homingCycle          *tview.Checkbox
	homingDirInvertX     *tview.Checkbox
	homingDirInvertY     *tview.Checkbox
	homingDirInvertZ     *tview.Checkbox
	homingFeed           *tview.InputField
	homingSeek           *tview.InputField
	homingDebounce       *tview.InputField
	homingPullOff        *tview.InputField
	maxSpindleSpeed      *tview.InputField
	minSpindleSpeed      *tview.InputField
	laserMode            *tview.Checkbox
	xSteps               *tview.InputField
	ySteps               *tview.InputField
	zSteps               *tview.InputField
	xMaxRate             *tview.InputField
	yMaxRate             *tview.InputField
	zMaxRate             *tview.InputField
	xAcceleration        *tview.InputField
	yAcceleration        *tview.InputField
	zAcceleration        *tview.InputField
	xMaxTravel           *tview.InputField
	yMaxTravel           *tview.InputField
	zMaxTravel           *tview.InputField
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

	mu               sync.Mutex
	skipQueueCommand bool
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

	newSettingInputField := func(key, label string, width int) *tview.InputField {
		field := tview.NewInputField()
		field.SetLabel(label)
		if width > 0 {
			field.SetFieldWidth(width)
		}
		field.SetDoneFunc(func(tcell.Key) {
			if sp.skipQueueCommand {
				return
			}
			sp.controlPrimitive.QueueCommand(fmt.Sprintf("$%s=%s", key, field.GetText()))
		})
		return field
	}

	newSettingCheckbox := func(key, label string) *tview.Checkbox {
		cb := tview.NewCheckbox()
		cb.SetLabel(label)
		cb.SetChangedFunc(func(checked bool) {
			if sp.skipQueueCommand {
				return
			}
			value := "0"
			if checked {
				value = "1"
			}
			sp.controlPrimitive.QueueCommand(fmt.Sprintf("$%s=%s", key, value))
		})
		return cb
	}

	newSettingMask := func(key string) (*tview.Checkbox, *tview.Checkbox, *tview.Checkbox) {
		x := tview.NewCheckbox()
		x.SetLabel("X")
		y := tview.NewCheckbox()
		y.SetLabel("Y")
		z := tview.NewCheckbox()
		z.SetLabel("Z")

		updateMask := func() {
			if sp.skipQueueCommand {
				return
			}
			mask := checkboxesToMask(x.IsChecked(), y.IsChecked(), z.IsChecked())
			sp.controlPrimitive.QueueCommand(fmt.Sprintf("$%s=%d", key, mask))
		}

		x.SetChangedFunc(func(bool) { updateMask() })
		y.SetChangedFunc(func(bool) { updateMask() })
		z.SetChangedFunc(func(bool) { updateMask() })

		return x, y, z
	}

	newSettingMaskContainer := func(label string, x, y, z *tview.Checkbox) tview.Primitive {
		flex := tview.NewFlex()
		flex.SetDirection(tview.FlexColumn)
		labelView := tview.NewTextView()
		labelView.SetText(label)
		flex.AddItem(labelView, len(label)+1, 0, false)
		flex.AddItem(x, 3, 0, false)
		flex.AddItem(y, 3, 0, false)
		flex.AddItem(z, 3, 0, false)
		return flex
	}

	const widthUs = len("1000000 ")
	const widthMs = len("1000 ")
	const widthMm = len("2600.000 ")
	const widthMmMin = len("1000.000 ")
	const widthRpm = len("10000 ")
	const widthStepsMm = len("1000.000 ")
	const widthMmSec2 = len("1000.000 ")

	// Settings: InputFields
	sp.stepPulse = newSettingInputField("0", "Step pulse(us)", widthUs)
	sp.stepIdleDelay = newSettingInputField("1", "Step idle delay(ms)", widthMs)
	sp.stepPortInvertX, sp.stepPortInvertY, sp.stepPortInvertZ = newSettingMask("2")
	sp.directionPortInvertX, sp.directionPortInvertY, sp.directionPortInvertZ = newSettingMask("3")
	sp.stepEnableInvert = newSettingCheckbox("4", "Step enable invert")
	sp.limitPinsInvert = newSettingCheckbox("5", "Limit pins invert")
	sp.probePinInvert = newSettingCheckbox("6", "Probe pin invert")
	sp.statusReport = newSettingInputField("10", "Status report(mask)", 2)
	sp.junctionDeviation = newSettingInputField("11", "Junction deviation(mm)", widthMm)
	sp.arcTolerance = newSettingInputField("12", "Arc tolerance(mm)", widthMm)
	sp.reportInches = newSettingCheckbox("13", "Report inches")
	sp.softLimits = newSettingCheckbox("20", "Soft limits")
	sp.hardLimits = newSettingCheckbox("21", "Hard limits")
	sp.homingCycle = newSettingCheckbox("22", "Homing cycle")
	sp.homingDirInvertX, sp.homingDirInvertY, sp.homingDirInvertZ = newSettingMask("23")
	sp.homingFeed = newSettingInputField("24", "Homing feed(mm/min)", widthMmMin)
	sp.homingSeek = newSettingInputField("25", "Homing seek(mm/min)", widthMmMin)
	sp.homingDebounce = newSettingInputField("26", "Homing debounce(ms)", widthMs)
	sp.homingPullOff = newSettingInputField("27", "Homing pull-off(mm)", widthMm)
	sp.maxSpindleSpeed = newSettingInputField("30", "Max spindle speed(RPM)", widthRpm)
	sp.minSpindleSpeed = newSettingInputField("31", "Min spindle speed(RPM)", widthRpm)
	sp.laserMode = newSettingCheckbox("32", "Laser mode")
	sp.xSteps = newSettingInputField("100", "X(steps/mm)", widthStepsMm)
	sp.ySteps = newSettingInputField("101", "Y(steps/mm)", widthStepsMm)
	sp.zSteps = newSettingInputField("102", "Z(steps/mm)", widthStepsMm)
	sp.xMaxRate = newSettingInputField("110", "X Max rate(mm/min)", widthMmMin)
	sp.yMaxRate = newSettingInputField("111", "Y Max rate(mm/min)", widthMmMin)
	sp.zMaxRate = newSettingInputField("112", "Z Max rate(mm/min)", widthMmMin)
	sp.xAcceleration = newSettingInputField("120", "X Acceleration(mm/sec^2)", widthMmSec2)
	sp.yAcceleration = newSettingInputField("121", "Y Acceleration(mm/sec^2)", widthMmSec2)
	sp.zAcceleration = newSettingInputField("122", "Z Acceleration(mm/sec^2)", widthMmSec2)
	sp.xMaxTravel = newSettingInputField("130", "X Max travel(mm)", widthMm)
	sp.yMaxTravel = newSettingInputField("131", "Y Max travel(mm)", widthMm)
	sp.zMaxTravel = newSettingInputField("132", "Z Max travel(mm)", widthMm)

	// Settings
	mainSettings := NewScrollContainer()
	mainSettings.SetBorder(true)
	mainSettings.SetTitle("Settings")
	mainSettings.AddPrimitive(sp.stepPulse, 1)
	mainSettings.AddPrimitive(sp.stepIdleDelay, 1)
	mainSettings.AddPrimitive(newSettingMaskContainer("Step port invert", sp.stepPortInvertX, sp.stepPortInvertY, sp.stepPortInvertZ), 1)
	mainSettings.AddPrimitive(newSettingMaskContainer("Direction port invert", sp.directionPortInvertX, sp.directionPortInvertY, sp.directionPortInvertZ), 1)
	mainSettings.AddPrimitive(sp.stepEnableInvert, 1)
	mainSettings.AddPrimitive(sp.limitPinsInvert, 1)
	mainSettings.AddPrimitive(sp.probePinInvert, 1)
	mainSettings.AddPrimitive(sp.statusReport, 1)
	mainSettings.AddPrimitive(sp.junctionDeviation, 1)
	mainSettings.AddPrimitive(sp.arcTolerance, 1)
	mainSettings.AddPrimitive(sp.reportInches, 1)
	mainSettings.AddPrimitive(sp.softLimits, 1)
	mainSettings.AddPrimitive(sp.hardLimits, 1)
	mainSettings.AddPrimitive(sp.homingCycle, 1)
	mainSettings.AddPrimitive(newSettingMaskContainer("Homing dir invert", sp.homingDirInvertX, sp.homingDirInvertY, sp.homingDirInvertZ), 1)
	mainSettings.AddPrimitive(sp.homingFeed, 1)
	mainSettings.AddPrimitive(sp.homingSeek, 1)
	mainSettings.AddPrimitive(sp.homingDebounce, 1)
	mainSettings.AddPrimitive(sp.homingPullOff, 1)
	mainSettings.AddPrimitive(sp.maxSpindleSpeed, 1)
	mainSettings.AddPrimitive(sp.minSpindleSpeed, 1)
	mainSettings.AddPrimitive(sp.laserMode, 1)
	mainSettings.AddPrimitive(sp.xSteps, 1)
	mainSettings.AddPrimitive(sp.ySteps, 1)
	mainSettings.AddPrimitive(sp.zSteps, 1)
	mainSettings.AddPrimitive(sp.xMaxRate, 1)
	mainSettings.AddPrimitive(sp.yMaxRate, 1)
	mainSettings.AddPrimitive(sp.zMaxRate, 1)
	mainSettings.AddPrimitive(sp.xAcceleration, 1)
	mainSettings.AddPrimitive(sp.yAcceleration, 1)
	mainSettings.AddPrimitive(sp.zAcceleration, 1)
	mainSettings.AddPrimitive(sp.xMaxTravel, 1)
	mainSettings.AddPrimitive(sp.yMaxTravel, 1)
	mainSettings.AddPrimitive(sp.zMaxTravel, 1)

	// Startup Lines: Input Fields
	sp.startupLine0InputField = newSettingInputField("N0", "0", 0)
	sp.startupLine1InputField = newSettingInputField("N1", "1", 0)

	// Startup Lines
	startupLinesFlex := tview.NewFlex()
	startupLinesFlex.SetDirection(tview.FlexRow)
	startupLinesFlex.SetBorder(true)
	startupLinesFlex.SetTitle("Startup Lines")
	startupLinesFlex.AddItem(sp.startupLine0InputField, 0, 1, false)
	startupLinesFlex.AddItem(sp.startupLine1InputField, 0, 1, false)

	// Build Info: Primitives
	versionTextView := tview.NewTextView()
	versionTextView.SetLabel("Version")
	sp.versionTextView = versionTextView
	sp.infoInputField = newSettingInputField("I", "Info", 0)
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

	sp.updateDisabled()

	return sp
}

func (sp *SettingsPrimitive) processMessagePushWelcome() {
	sp.app.QueueUpdateDraw(func() {
		sp.skipQueueCommand = true
		defer func() { sp.skipQueueCommand = false }()
		// Settings
		sp.stepPulse.SetText("")
		sp.stepIdleDelay.SetText("")
		sp.stepPortInvertX.SetChecked(false)
		sp.stepPortInvertY.SetChecked(false)
		sp.stepPortInvertZ.SetChecked(false)
		sp.directionPortInvertX.SetChecked(false)
		sp.directionPortInvertY.SetChecked(false)
		sp.directionPortInvertZ.SetChecked(false)
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
		sp.homingDirInvertX.SetChecked(false)
		sp.homingDirInvertY.SetChecked(false)
		sp.homingDirInvertZ.SetChecked(false)
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
			sp.skipQueueCommand = true
			sp.versionTextView.SetText(versionText)
			sp.skipQueueCommand = false
		}
		infoText := tview.Escape(messagePushVersion.Info)
		if infoText != sp.infoInputField.GetText() {
			sp.skipQueueCommand = true
			sp.infoInputField.SetText(infoText)
			sp.skipQueueCommand = false
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
		sp.skipQueueCommand = true
		sp.compileTimeOptionsTextView.SetText(buf.String())
		sp.skipQueueCommand = false
	})
}

//gocyclo:ignore
func (sp *SettingsPrimitive) processMessagePushSetting(messagePushSetting *grblMod.MessagePushSetting) {
	sp.app.QueueUpdateDraw(func() {
		sp.skipQueueCommand = true
		defer func() { sp.skipQueueCommand = false }()
		switch messagePushSetting.Key {
		// Settings
		case "0":
			sp.stepPulse.SetText(messagePushSetting.Value)
		case "1":
			sp.stepIdleDelay.SetText(messagePushSetting.Value)
		case "2":
			mask, err := strconv.Atoi(messagePushSetting.Value)
			if err != nil {
				panic(fmt.Sprintf("failed to parse: %s: %s", messagePushSetting, err))
			}
			x, y, z := maskToCheckboxes(mask)
			sp.stepPortInvertX.SetChecked(x)
			sp.stepPortInvertY.SetChecked(y)
			sp.stepPortInvertZ.SetChecked(z)
		case "3":
			mask, err := strconv.Atoi(messagePushSetting.Value)
			if err != nil {
				panic(fmt.Sprintf("failed to parse: %s: %s", messagePushSetting, err))
			}
			x, y, z := maskToCheckboxes(mask)
			sp.directionPortInvertX.SetChecked(x)
			sp.directionPortInvertY.SetChecked(y)
			sp.directionPortInvertZ.SetChecked(z)
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
			mask, err := strconv.Atoi(messagePushSetting.Value)
			if err != nil {
				panic(fmt.Sprintf("failed to parse: %s: %s", messagePushSetting, err))
			}
			x, y, z := maskToCheckboxes(mask)
			sp.homingDirInvertX.SetChecked(x)
			sp.homingDirInvertY.SetChecked(y)
			sp.homingDirInvertZ.SetChecked(z)
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
	sp.stepPortInvertX.SetDisabled(disabled)
	sp.stepPortInvertY.SetDisabled(disabled)
	sp.stepPortInvertZ.SetDisabled(disabled)
	sp.directionPortInvertX.SetDisabled(disabled)
	sp.directionPortInvertY.SetDisabled(disabled)
	sp.directionPortInvertZ.SetDisabled(disabled)
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
	sp.homingDirInvertX.SetDisabled(disabled)
	sp.homingDirInvertY.SetDisabled(disabled)
	sp.homingDirInvertZ.SetDisabled(disabled)
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
