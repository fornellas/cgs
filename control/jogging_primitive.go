package control

import (
	"bytes"
	"context"
	"fmt"
	"slices"
	"strconv"
	"sync"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	grblMod "github.com/fornellas/cgs/grbl"
)

type JoggingPrimitive struct {
	*tview.Flex
	app              *tview.Application
	controlPrimitive *ControlPrimitive
	// Joystick
	xMinusButton               *tview.Button
	xPlusButton                *tview.Button
	yMinusButton               *tview.Button
	yPlusButton                *tview.Button
	zMinusButton               *tview.Button
	zPlusButton                *tview.Button
	joystickFeedRateInputField *tview.InputField
	distanceInputField         *tview.InputField
	joystickUnitDropDown       *tview.DropDown
	joystickCancelButton       *tview.Button
	joystickJogOk              bool
	// Parameters
	xInputField                *tview.InputField
	yInputField                *tview.InputField
	zInputField                *tview.InputField
	unitOptions                []string
	paramsUnitDropDown         *tview.DropDown
	distanceModeOptions        []string
	distanceModeDropDown       *tview.DropDown
	paramsFeedRateInputField   *tview.InputField
	machineCoordinatesCheckbox *tview.Checkbox
	paramErrTextView           *tview.TextView
	jogParametersButton        *tview.Button
	paramsJogBlock             string
	paramsCancelButton         *tview.Button
	// Messages
	machineState grblMod.StatusReportMachineState

	mu sync.Mutex
}

func NewJoggingPrimitive(
	ctx context.Context,
	app *tview.Application,
	controlPrimitive *ControlPrimitive,
) *JoggingPrimitive {
	jp := &JoggingPrimitive{
		app:                 app,
		controlPrimitive:    controlPrimitive,
		unitOptions:         []string{"Inches", "Millimeters"},
		distanceModeOptions: []string{"Absolute", "Incremental"},
	}

	joystickFlex := jp.newJoystickFlex()

	parametersForm := jp.newParametersForm()

	parametersFlex := tview.NewFlex()
	parametersFlex.SetBorder(true)
	parametersFlex.SetTitle("Parameters")
	parametersFlex.AddItem(parametersForm, 0, 1, false)

	joggingFlex := tview.NewFlex()
	joggingFlex.SetBorder(true)
	joggingFlex.SetTitle("Jogging")
	joggingFlex.SetDirection(tview.FlexColumn)
	joggingFlex.AddItem(joystickFlex, 0, 1, false)
	joggingFlex.AddItem(parametersFlex, 0, 1, false)
	jp.Flex = joggingFlex

	return jp
}

func (jp *JoggingPrimitive) setJoystickJogOk() {
	jp.mu.Lock()
	jp.joystickJogOk = true
	if jp.joystickFeedRateInputField.GetText() == "" {
		jp.joystickJogOk = false
	}
	if jp.distanceInputField.GetText() == "" {
		jp.joystickJogOk = false
	}
	jp.mu.Unlock()

	jp.updateDisabled()
}

func (jp *JoggingPrimitive) newJoystickFlex() *tview.Flex {
	jog := func(axis string) {
		var buf bytes.Buffer
		feed, err := strconv.ParseFloat(jp.joystickFeedRateInputField.GetText(), 64)
		if err != nil {
			panic(err)
		}
		distance, err := strconv.ParseFloat(jp.distanceInputField.GetText(), 64)
		if err != nil {
			panic(err)
		}
		var unitWord string
		_, unit := jp.joystickUnitDropDown.GetCurrentOption()
		switch unit {
		case "Inches":
			unitWord = "G20"
		case "Millimeters":
			unitWord = "G21"
		default:
			panic(fmt.Sprintf("bug: bad unit option: %#v", unit))
		}
		fmt.Fprintf(&buf, "$J=F%.4f%s%.4f%sG91", feed, axis, distance, unitWord)
		jp.controlPrimitive.QueueCommand(buf.String())
	}

	xMinusButton := tview.NewButton("-X")
	xMinusButton.SetSelectedFunc(func() { jog("X-") })
	jp.xMinusButton = xMinusButton

	xPlusButton := tview.NewButton("+X")
	xPlusButton.SetSelectedFunc(func() { jog("X") })
	jp.xPlusButton = xPlusButton

	yMinusButton := tview.NewButton("-Y")
	yMinusButton.SetSelectedFunc(func() { jog("Y-") })
	jp.yMinusButton = yMinusButton

	yPlusButton := tview.NewButton("+Y")
	yPlusButton.SetSelectedFunc(func() { jog("Y") })
	jp.yPlusButton = yPlusButton

	zMinusButton := tview.NewButton("-Z")
	zMinusButton.SetSelectedFunc(func() { jog("Z-") })
	jp.zMinusButton = zMinusButton

	zPlusButton := tview.NewButton("+Z")
	zPlusButton.SetSelectedFunc(func() { jog("Z") })
	jp.zPlusButton = zPlusButton

	joystickGrid := tview.NewGrid()
	joystickGrid.SetColumns(0, 0, 0, 0)
	joystickGrid.SetRows(0, 0)
	joystickGrid.SetGap(1, 1)
	joystickGrid.AddItem(jp.xMinusButton, 1, 0, 1, 1, 0, 0, false)
	joystickGrid.AddItem(jp.xPlusButton, 1, 2, 1, 1, 0, 0, false)
	joystickGrid.AddItem(jp.yMinusButton, 1, 1, 1, 1, 0, 0, false)
	joystickGrid.AddItem(jp.yPlusButton, 0, 1, 1, 1, 0, 0, false)
	joystickGrid.AddItem(jp.zMinusButton, 1, 3, 1, 1, 0, 0, false)
	joystickGrid.AddItem(jp.zPlusButton, 0, 3, 1, 1, 0, 0, false)

	parametersForm := tview.NewForm()
	parametersForm.SetButtonsAlign(tview.AlignCenter)

	parametersForm.AddInputField("Feed rate", "", 0, acceptUFloat, nil)
	jp.joystickFeedRateInputField = parametersForm.
		GetFormItem(parametersForm.GetFormItemCount() - 1).(*tview.InputField)
	jp.joystickFeedRateInputField.SetChangedFunc(func(string) { jp.setJoystickJogOk() })

	parametersForm.AddInputField("Distance", "", 0, acceptUFloat, nil)
	jp.distanceInputField = parametersForm.
		GetFormItem(parametersForm.GetFormItemCount() - 1).(*tview.InputField)
	jp.distanceInputField.SetChangedFunc(func(string) { jp.setJoystickJogOk() })

	parametersForm.AddDropDown("Unit", jp.unitOptions, -1, nil)
	jp.joystickUnitDropDown = parametersForm.
		GetFormItem(parametersForm.GetFormItemCount() - 1).(*tview.DropDown)
	jp.joystickUnitDropDown.SetSelectedFunc(func(string, int) {
		// TODO
	})

	parametersForm.AddButton("Cancel", func() {
		// TODO
	})
	jp.joystickCancelButton = parametersForm.GetButton(parametersForm.GetButtonCount() - 1)

	joystickFlex := tview.NewFlex()
	joystickFlex.SetBorder(true)
	joystickFlex.SetDirection(tview.FlexRow)
	joystickFlex.SetTitle("Joystick")
	joystickFlex.AddItem(joystickGrid, 0, 1, false)
	joystickFlex.AddItem(parametersForm, 0, 1, false)
	return joystickFlex
}

func (jp *JoggingPrimitive) getParamsJogBlock() (string, error) {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "$J=")

	printWord := func(value, letter string) bool {
		if len(value) == 0 {
			return false
		}
		v, err := strconv.ParseFloat(value, 64)
		if err != nil {
			panic(fmt.Sprintf("bug: parsing not expected to fail: %s", err))
		}
		fmt.Fprintf(&buf, "%s%.4f", letter, v)
		return true
	}

	var hasCoordinate bool

	if printWord(jp.xInputField.GetText(), "X") {
		hasCoordinate = true
	}
	if printWord(jp.yInputField.GetText(), "Y") {
		hasCoordinate = true
	}
	if printWord(jp.zInputField.GetText(), "Z") {
		hasCoordinate = true
	}
	if !hasCoordinate {
		return "", fmt.Errorf("missing X, Y or Z")
	}

	var unitWord string
	_, unit := jp.paramsUnitDropDown.GetCurrentOption()
	switch unit {
	case "Inches":
		unitWord = "G20"
	case "Millimeters":
		unitWord = "G21"
	default:
		panic(fmt.Sprintf("bug: bad unit option: %#v", unit))
	}
	fmt.Fprintf(&buf, "%s", unitWord)

	var distanceModeWord string
	_, distanceMode := jp.distanceModeDropDown.GetCurrentOption()
	switch distanceMode {
	case "Absolute":
		distanceModeWord = "G90"
	case "Incremental":
		distanceModeWord = "G91"
	default:
		panic(fmt.Sprintf("bug: bad distanceMode option: %#v", distanceMode))
	}
	fmt.Fprintf(&buf, "%s", distanceModeWord)

	if !printWord(jp.paramsFeedRateInputField.GetText(), "F") {
		return "", fmt.Errorf("missing feed rate")
	}

	if jp.machineCoordinatesCheckbox.IsChecked() {
		fmt.Fprint(&buf, "G53")
	}

	return buf.String(), nil
}

func (jp *JoggingPrimitive) updateDisabled() {
	jp.mu.Lock()
	switch jp.machineState.State {
	case "Idle":
		// Joystick
		jogDisabled := !jp.joystickJogOk
		jp.xMinusButton.SetDisabled(jogDisabled)
		jp.xPlusButton.SetDisabled(jogDisabled)
		jp.yMinusButton.SetDisabled(jogDisabled)
		jp.yPlusButton.SetDisabled(jogDisabled)
		jp.zMinusButton.SetDisabled(jogDisabled)
		jp.zPlusButton.SetDisabled(jogDisabled)
		jp.joystickFeedRateInputField.SetDisabled(false)
		jp.distanceInputField.SetDisabled(false)
		jp.joystickUnitDropDown.SetDisabled(false)
		jp.joystickCancelButton.SetDisabled(true)
		// Parameters
		jp.xInputField.SetDisabled(false)
		jp.yInputField.SetDisabled(false)
		jp.zInputField.SetDisabled(false)
		jp.paramsUnitDropDown.SetDisabled(false)
		jp.distanceModeDropDown.SetDisabled(false)
		jp.paramsFeedRateInputField.SetDisabled(false)
		if _, option := jp.distanceModeDropDown.GetCurrentOption(); option == "Incremental" {
			jp.machineCoordinatesCheckbox.SetDisabled(true)
		} else {
			jp.machineCoordinatesCheckbox.SetDisabled(false)
		}
		if jp.paramsJogBlock != "" {
			jp.jogParametersButton.SetDisabled(false)
		} else {
			jp.jogParametersButton.SetDisabled(true)
		}
		jp.paramsCancelButton.SetDisabled(true)
	case "Jog":
		// Joystick
		jp.xMinusButton.SetDisabled(true)
		jp.xPlusButton.SetDisabled(true)
		jp.yMinusButton.SetDisabled(true)
		jp.yPlusButton.SetDisabled(true)
		jp.zMinusButton.SetDisabled(true)
		jp.zPlusButton.SetDisabled(true)
		jp.joystickFeedRateInputField.SetDisabled(true)
		jp.distanceInputField.SetDisabled(true)
		jp.joystickUnitDropDown.SetDisabled(true)
		jp.joystickCancelButton.SetDisabled(false)
		// Parameters
		jp.xInputField.SetDisabled(true)
		jp.yInputField.SetDisabled(true)
		jp.zInputField.SetDisabled(true)
		jp.paramsUnitDropDown.SetDisabled(true)
		jp.distanceModeDropDown.SetDisabled(true)
		jp.paramsFeedRateInputField.SetDisabled(true)
		jp.machineCoordinatesCheckbox.SetDisabled(true)
		jp.jogParametersButton.SetDisabled(true)
		jp.paramsCancelButton.SetDisabled(false)
	case "Run", "Hold", "Alarm", "Door", "Check", "Home", "Sleep":
		// Joystick
		jp.xMinusButton.SetDisabled(true)
		jp.xPlusButton.SetDisabled(true)
		jp.yMinusButton.SetDisabled(true)
		jp.yPlusButton.SetDisabled(true)
		jp.zMinusButton.SetDisabled(true)
		jp.zPlusButton.SetDisabled(true)
		jp.joystickFeedRateInputField.SetDisabled(true)
		jp.distanceInputField.SetDisabled(true)
		jp.joystickUnitDropDown.SetDisabled(true)
		jp.joystickCancelButton.SetDisabled(true)
		// Parameters
		jp.xInputField.SetDisabled(true)
		jp.yInputField.SetDisabled(true)
		jp.zInputField.SetDisabled(true)
		jp.paramsUnitDropDown.SetDisabled(true)
		jp.distanceModeDropDown.SetDisabled(true)
		jp.paramsFeedRateInputField.SetDisabled(true)
		jp.machineCoordinatesCheckbox.SetDisabled(true)
		jp.jogParametersButton.SetDisabled(true)
		jp.paramsCancelButton.SetDisabled(true)
	default:
		panic(fmt.Sprintf("unknown machine state: %#v", jp.machineState.State))
	}
	jp.mu.Unlock()
}

func (jp *JoggingPrimitive) setParamsJogBlock() {
	jogBlock, err := jp.getParamsJogBlock()
	if err != nil {
		jp.paramErrTextView.SetText(fmt.Sprintf("[%s]%s[-]", tcell.ColorRed, tview.Escape(err.Error())))
		jp.paramsJogBlock = ""
		return
	}
	if jogBlock == jp.paramsJogBlock {
		return
	}

	jp.paramErrTextView.SetText(fmt.Sprintf("[%s]None[-]", tcell.ColorGreen))

	jp.mu.Lock()
	jp.paramsJogBlock = jogBlock
	jp.mu.Unlock()

	jp.updateDisabled()
}

func (jp *JoggingPrimitive) newParametersForm() *tview.Form {
	parametersForm := tview.NewForm()
	parametersForm.SetButtonsAlign(tview.AlignCenter)

	parametersForm.AddInputField("X", "", 0, acceptFloat, nil)
	jp.xInputField = parametersForm.
		GetFormItem(parametersForm.GetFormItemCount() - 1).(*tview.InputField)
	jp.xInputField.SetChangedFunc(func(string) { jp.setParamsJogBlock() })

	parametersForm.AddInputField("Y", "", 0, acceptFloat, nil)
	jp.yInputField = parametersForm.
		GetFormItem(parametersForm.GetFormItemCount() - 1).(*tview.InputField)
	jp.yInputField.SetChangedFunc(func(string) { jp.setParamsJogBlock() })

	parametersForm.AddInputField("Z", "", 0, acceptFloat, nil)
	jp.zInputField = parametersForm.
		GetFormItem(parametersForm.GetFormItemCount() - 1).(*tview.InputField)
	jp.zInputField.SetChangedFunc(func(string) { jp.setParamsJogBlock() })

	parametersForm.AddDropDown("Unit", jp.unitOptions, -1, nil)
	jp.paramsUnitDropDown = parametersForm.
		GetFormItem(parametersForm.GetFormItemCount() - 1).(*tview.DropDown)
	jp.paramsUnitDropDown.SetSelectedFunc(func(string, int) { jp.setParamsJogBlock() })

	parametersForm.AddDropDown("Distance mode", jp.distanceModeOptions, -1, nil)
	jp.distanceModeDropDown = parametersForm.
		GetFormItem(parametersForm.GetFormItemCount() - 1).(*tview.DropDown)
	jp.distanceModeDropDown.SetSelectedFunc(func(string, int) { jp.setParamsJogBlock() })

	parametersForm.AddInputField("Feed rate", "", 0, acceptUFloat, nil)
	jp.paramsFeedRateInputField = parametersForm.
		GetFormItem(parametersForm.GetFormItemCount() - 1).(*tview.InputField)
	jp.paramsFeedRateInputField.SetChangedFunc(func(string) { jp.setParamsJogBlock() })

	parametersForm.AddCheckbox("Machine Coordinates", false, nil)
	jp.machineCoordinatesCheckbox = parametersForm.
		GetFormItem(parametersForm.GetFormItemCount() - 1).(*tview.Checkbox)
	jp.machineCoordinatesCheckbox.SetChangedFunc(func(bool) { jp.setParamsJogBlock() })
	jp.distanceModeDropDown.SetSelectedFunc(func(option string, optionIndex int) {
		if option == "Incremental" {
			jp.machineCoordinatesCheckbox.SetDisabled(true)
		} else {
			jp.machineCoordinatesCheckbox.SetDisabled(false)
		}
	})

	parametersForm.AddTextView("Error", "", 0, 2, true, true)
	jp.paramErrTextView = parametersForm.
		GetFormItem(parametersForm.GetFormItemCount() - 1).(*tview.TextView)

	parametersForm.AddButton("Jog", func() {
		jp.mu.Lock()
		jp.controlPrimitive.QueueCommand(jp.paramsJogBlock)
		jp.mu.Unlock()
	})
	jp.jogParametersButton = parametersForm.GetButton(parametersForm.GetButtonCount() - 1)

	parametersForm.AddButton("Cancel", func() {
		jp.controlPrimitive.QueueRealTimeCommand(grblMod.RealTimeCommandJogCancel)
	})
	jp.paramsCancelButton = parametersForm.GetButton(parametersForm.GetButtonCount() - 1)

	return parametersForm
}

func (jp *JoggingPrimitive) processMessagePushWelcome() {
	jp.app.QueueUpdateDraw(func() {
		// Joystick
		jp.joystickUnitDropDown.SetCurrentOption(-1)
		// Parameters
		jp.paramsUnitDropDown.SetCurrentOption(-1)
		jp.distanceModeDropDown.SetCurrentOption(-1)
	})
}

func (jp *JoggingPrimitive) processMessagePushGcodeState(
	messagePushGcodeState *grblMod.MessagePushGcodeState,
) {
	jp.app.QueueUpdateDraw(func() {
		modalGroup := messagePushGcodeState.ModalGroup
		if modalGroup != nil {
			units := modalGroup.Units
			if units != nil {
				switch units.NormalizedString() {
				case "G20":
					jp.joystickUnitDropDown.SetCurrentOption(slices.Index(jp.unitOptions, "Inches"))
					jp.paramsUnitDropDown.SetCurrentOption(slices.Index(jp.unitOptions, "Inches"))
				case "G21":
					jp.joystickUnitDropDown.SetCurrentOption(slices.Index(jp.unitOptions, "Millimeters"))
					jp.paramsUnitDropDown.SetCurrentOption(slices.Index(jp.unitOptions, "Millimeters"))
				}
			}
			distanceMode := modalGroup.DistanceMode
			if distanceMode != nil {
				switch distanceMode.NormalizedString() {
				case "G90":
					jp.distanceModeDropDown.SetCurrentOption(slices.Index(jp.distanceModeOptions, "Absolute"))
				case "G91":
					jp.distanceModeDropDown.SetCurrentOption(slices.Index(jp.distanceModeOptions, "Incremental"))
				}
			}
		}
	})
}

func (jp *JoggingPrimitive) setMachineState(machineState grblMod.StatusReportMachineState) {
	if jp.machineState == machineState {
		return
	}

	jp.mu.Lock()
	jp.machineState = machineState
	jp.mu.Unlock()

	jp.app.QueueUpdateDraw(func() {
		jp.updateDisabled()
	})
}

func (jp *JoggingPrimitive) processMessagePushStatusReport(
	messagePushStatusReport *grblMod.MessagePushStatusReport,
) {
	jp.setMachineState(messagePushStatusReport.MachineState)
}

func (jp *JoggingPrimitive) ProcessMessage(ctx context.Context, message grblMod.Message) {
	if _, ok := message.(*grblMod.MessagePushWelcome); ok {
		jp.processMessagePushWelcome()
		return
	}

	if messagePushGcodeState, ok := message.(*grblMod.MessagePushGcodeState); ok {
		jp.processMessagePushGcodeState(messagePushGcodeState)
		return
	}
	if messagePushStatusReport, ok := message.(*grblMod.MessagePushStatusReport); ok {
		jp.processMessagePushStatusReport(messagePushStatusReport)
		return
	}
}
