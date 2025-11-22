package control

// Like (G1)
// Args:
// 	One of:
// 		[X$x | Y$y | Z$z]
//  Required:
// 		F$f
//  Optional:
//  	[G20|G21] Inches/Millimeters
// 		[G90|G91] Absolute/Incremental
// 		G53 Machine Coordinates

import (
	"bytes"
	"context"
	"fmt"
	"slices"
	"strconv"

	"github.com/rivo/tview"

	grblMod "github.com/fornellas/cgs/grbl"
)

type JoggingPrimitive struct {
	*tview.Flex
	app                        *tview.Application
	controlPrimitive           *ControlPrimitive
	xInputField                *tview.InputField
	yInputField                *tview.InputField
	zInputField                *tview.InputField
	unitOptions                []string
	unitDropDown               *tview.DropDown
	distanceModeOptions        []string
	distanceModeDropDown       *tview.DropDown
	feedRateInputField         *tview.InputField
	machineCoordinatesCheckbox *tview.Checkbox
	jogParametersButton        *tview.Button
	cancelButton               *tview.Button
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

	acceptFloatFn := func(textToCheck string, lastChar rune) bool {
		if len(textToCheck) > 0 && textToCheck[0] == '-' {
			return true
		}
		_, err := strconv.ParseFloat(textToCheck, 64)
		return err == nil
	}

	//   A    A
	// < V >  V
	// Feed rate
	// Distance
	// Unit
	// Cancel
	joystickFlex := tview.NewFlex()
	joystickFlex.SetBorder(true)
	joystickFlex.SetTitle("Joystick")
	// joystickFlex.AddItem(joystickGrid, 0, 1, false)

	parametersForm := tview.NewForm()
	const width = len("100.0000")
	parametersForm.AddInputField("X", "", width, acceptFloatFn, nil)
	jp.xInputField = parametersForm.
		GetFormItem(parametersForm.GetFormItemCount() - 1).(*tview.InputField)
	parametersForm.AddInputField("Y", "", width, acceptFloatFn, nil)
	jp.yInputField = parametersForm.
		GetFormItem(parametersForm.GetFormItemCount() - 1).(*tview.InputField)
	parametersForm.AddInputField("Z", "", width, acceptFloatFn, nil)
	jp.zInputField = parametersForm.
		GetFormItem(parametersForm.GetFormItemCount() - 1).(*tview.InputField)
	parametersForm.AddDropDown("Unit", jp.unitOptions, -1, nil)
	jp.unitDropDown = parametersForm.
		GetFormItem(parametersForm.GetFormItemCount() - 1).(*tview.DropDown)
	parametersForm.AddDropDown("Distance mode", jp.distanceModeOptions, -1, nil)
	jp.distanceModeDropDown = parametersForm.
		GetFormItem(parametersForm.GetFormItemCount() - 1).(*tview.DropDown)
	parametersForm.AddInputField("Feed rate", "", width, acceptFloatFn, nil)
	jp.feedRateInputField = parametersForm.
		GetFormItem(parametersForm.GetFormItemCount() - 1).(*tview.InputField)
	parametersForm.AddCheckbox("Machine Coordinates", false, nil)
	jp.machineCoordinatesCheckbox = parametersForm.
		GetFormItem(parametersForm.GetFormItemCount() - 1).(*tview.Checkbox)
	parametersForm.AddButton("Jog", jp.jog)
	jp.jogParametersButton = parametersForm.GetButton(parametersForm.GetButtonCount() - 1)
	parametersForm.AddButton("Cancel", func() {
		controlPrimitive.QueueRealTimeCommand(grblMod.RealTimeCommandJogCancel)
	})
	jp.cancelButton = parametersForm.GetButton(parametersForm.GetButtonCount() - 1)

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

func (jp *JoggingPrimitive) jog() {
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

	var err error
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
		err = fmt.Errorf("missing at least one of X, Y or Z")
	}

	var unitWord string
	_, unit := jp.unitDropDown.GetCurrentOption()
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

	if !printWord(jp.feedRateInputField.GetText(), "F") {
		err = fmt.Errorf("missing feed rate")
	}

	if jp.machineCoordinatesCheckbox.IsChecked() {
		fmt.Fprint(&buf, "G53")
	}

	if err != nil {
		// TODO report error
		return
	}

	jp.controlPrimitive.QueueCommand(buf.String())
}

func (jp *JoggingPrimitive) processMessagePushGcodeState(
	ctx context.Context,
	messagePushGcodeState *grblMod.MessagePushGcodeState,
) {
	jp.app.QueueUpdateDraw(func() {
		modalGroup := messagePushGcodeState.ModalGroup
		if modalGroup != nil {
			units := modalGroup.Units
			if units != nil {
				if n, _ := jp.unitDropDown.GetCurrentOption(); n < 0 {
					switch units.NormalizedString() {
					case "G20":
						jp.unitDropDown.SetCurrentOption(slices.Index(jp.unitOptions, "Inches"))
					case "G21":
						jp.unitDropDown.SetCurrentOption(slices.Index(jp.unitOptions, "Millimeters"))
					}
				}
			}
			// distanceModeDropDown
			distanceMode := modalGroup.DistanceMode
			if distanceMode != nil {
				if n, _ := jp.distanceModeDropDown.GetCurrentOption(); n < 0 {
					switch distanceMode.NormalizedString() {
					case "G90":
						jp.distanceModeDropDown.SetCurrentOption(slices.Index(jp.distanceModeOptions, "Absolute"))
					case "G91":
						jp.distanceModeDropDown.SetCurrentOption(slices.Index(jp.distanceModeOptions, "Incremental"))
					}
				}
			}
		}
		// TODO initial feedRate to min of
		// $110, $111 and $112 – [X,Y,Z] Max rate, mm/min
		// jp.feedRateInputField.SetText(fmt.Sprintf("%.4f", feedRate))
	})
}

func (jp *JoggingPrimitive) processMessagePushStatusReport(
	messagePushStatusReport *grblMod.MessagePushStatusReport,
) {
	jp.app.QueueUpdateDraw(func() {
		switch messagePushStatusReport.MachineState.State {
		case "Idle":
			jp.xInputField.SetDisabled(false)
			jp.yInputField.SetDisabled(false)
			jp.zInputField.SetDisabled(false)
			jp.unitDropDown.SetDisabled(false)
			jp.distanceModeDropDown.SetDisabled(false)
			jp.feedRateInputField.SetDisabled(false)
			if _, option := jp.distanceModeDropDown.GetCurrentOption(); option == "Incremental" {
				jp.machineCoordinatesCheckbox.SetDisabled(true)
			} else {
				jp.machineCoordinatesCheckbox.SetDisabled(false)
			}
			jp.jogParametersButton.SetDisabled(false)
			jp.cancelButton.SetDisabled(true)
		case "Jog":
			jp.xInputField.SetDisabled(true)
			jp.yInputField.SetDisabled(true)
			jp.zInputField.SetDisabled(true)
			jp.unitDropDown.SetDisabled(true)
			jp.distanceModeDropDown.SetDisabled(true)
			jp.feedRateInputField.SetDisabled(true)
			jp.machineCoordinatesCheckbox.SetDisabled(true)
			jp.jogParametersButton.SetDisabled(true)
			jp.cancelButton.SetDisabled(false)
		case "Run", "Hold", "Alarm", "Door", "Check", "Home", "Sleep":
			jp.xInputField.SetDisabled(true)
			jp.yInputField.SetDisabled(true)
			jp.zInputField.SetDisabled(true)
			jp.unitDropDown.SetDisabled(true)
			jp.distanceModeDropDown.SetDisabled(true)
			jp.feedRateInputField.SetDisabled(true)
			jp.machineCoordinatesCheckbox.SetDisabled(true)
			jp.jogParametersButton.SetDisabled(true)
			jp.cancelButton.SetDisabled(true)
		default:
			panic(fmt.Sprintf("unknown machine state: %#v", messagePushStatusReport.MachineState.State))
		}
	})
}

func (jp *JoggingPrimitive) ProcessMessage(ctx context.Context, message grblMod.Message) {
	if messagePushGcodeState, ok := message.(*grblMod.MessagePushGcodeState); ok {
		jp.processMessagePushGcodeState(ctx, messagePushGcodeState)
		return
	}
	if messagePushStatusReport, ok := message.(*grblMod.MessagePushStatusReport); ok {
		jp.processMessagePushStatusReport(messagePushStatusReport)
		return
	}
}
