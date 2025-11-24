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
	// Parameters
	xInputField                *tview.InputField
	yInputField                *tview.InputField
	zInputField                *tview.InputField
	unitOptions                []string
	unitDropDown               *tview.DropDown
	distanceModeOptions        []string
	distanceModeDropDown       *tview.DropDown
	feedRateInputField         *tview.InputField
	machineCoordinatesCheckbox *tview.Checkbox
	paramErrTextView           *tview.TextView
	jogParametersButton        *tview.Button
	jogBlock                   string
	cancelButton               *tview.Button
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

func (jp *JoggingPrimitive) newParametersForm() *tview.Form {
	parametersForm := tview.NewForm()
	parametersForm.SetButtonsAlign(tview.AlignCenter)
	const width = len("100.0000")

	parametersForm.AddInputField("X", "", width, acceptFloat, nil)
	jp.xInputField = parametersForm.
		GetFormItem(parametersForm.GetFormItemCount() - 1).(*tview.InputField)
	jp.xInputField.SetChangedFunc(func(string) { jp.setJogBlock() })

	parametersForm.AddInputField("Y", "", width, acceptFloat, nil)
	jp.yInputField = parametersForm.
		GetFormItem(parametersForm.GetFormItemCount() - 1).(*tview.InputField)
	jp.yInputField.SetChangedFunc(func(string) { jp.setJogBlock() })

	parametersForm.AddInputField("Z", "", width, acceptFloat, nil)
	jp.zInputField = parametersForm.
		GetFormItem(parametersForm.GetFormItemCount() - 1).(*tview.InputField)
	jp.zInputField.SetChangedFunc(func(string) { jp.setJogBlock() })

	parametersForm.AddDropDown("Unit", jp.unitOptions, -1, nil)
	jp.unitDropDown = parametersForm.
		GetFormItem(parametersForm.GetFormItemCount() - 1).(*tview.DropDown)
	jp.unitDropDown.SetSelectedFunc(func(string, int) { jp.setJogBlock() })

	parametersForm.AddDropDown("Distance mode", jp.distanceModeOptions, -1, nil)
	jp.distanceModeDropDown = parametersForm.
		GetFormItem(parametersForm.GetFormItemCount() - 1).(*tview.DropDown)
	jp.distanceModeDropDown.SetSelectedFunc(func(string, int) { jp.setJogBlock() })

	parametersForm.AddInputField("Feed rate", "", width, acceptUFloat, nil)
	jp.feedRateInputField = parametersForm.
		GetFormItem(parametersForm.GetFormItemCount() - 1).(*tview.InputField)
	jp.feedRateInputField.SetChangedFunc(func(string) { jp.setJogBlock() })

	parametersForm.AddCheckbox("Machine Coordinates", false, nil)
	jp.machineCoordinatesCheckbox = parametersForm.
		GetFormItem(parametersForm.GetFormItemCount() - 1).(*tview.Checkbox)
	jp.machineCoordinatesCheckbox.SetChangedFunc(func(bool) { jp.setJogBlock() })
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

	parametersForm.AddButton("Jog", jp.jog)
	jp.jogParametersButton = parametersForm.GetButton(parametersForm.GetButtonCount() - 1)

	parametersForm.AddButton("Cancel", func() {
		jp.controlPrimitive.QueueRealTimeCommand(grblMod.RealTimeCommandJogCancel)
	})
	jp.cancelButton = parametersForm.GetButton(parametersForm.GetButtonCount() - 1)

	return parametersForm
}

func (jp *JoggingPrimitive) getJogBlock() (string, error) {
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
		return "", fmt.Errorf("missing feed rate")
	}

	if jp.machineCoordinatesCheckbox.IsChecked() {
		fmt.Fprint(&buf, "G53")
	}

	return buf.String(), nil
}

func (jp *JoggingPrimitive) setJogBlock() {
	jogBlock, err := jp.getJogBlock()
	if err != nil {
		jp.paramErrTextView.SetText(fmt.Sprintf("[%s]%s[-]", tcell.ColorRed, tview.Escape(err.Error())))
		jp.jogBlock = ""
		return
	}
	if jogBlock == jp.jogBlock {
		return
	}

	jp.paramErrTextView.SetText(fmt.Sprintf("[%s]None[-]", tcell.ColorGreen))

	jp.mu.Lock()
	jp.jogBlock = jogBlock
	jp.mu.Unlock()

	jp.updateDisabled()
}

func (jp *JoggingPrimitive) jog() {
	jp.mu.Lock()
	jp.controlPrimitive.QueueCommand(jp.jogBlock)
	jp.mu.Unlock()
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

func (jp *JoggingPrimitive) processMessagePushGcodeState(
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

func (jp *JoggingPrimitive) updateDisabled() {
	jp.mu.Lock()
	switch jp.machineState.State {
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
		if jp.jogBlock != "" {
			jp.jogParametersButton.SetDisabled(false)
		} else {
			jp.jogParametersButton.SetDisabled(true)
		}
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
		panic(fmt.Sprintf("unknown machine state: %#v", jp.machineState.State))
	}
	jp.mu.Unlock()
}

func (jp *JoggingPrimitive) processMessagePushStatusReport(
	messagePushStatusReport *grblMod.MessagePushStatusReport,
) {
	jp.setMachineState(messagePushStatusReport.MachineState)
}

func (jp *JoggingPrimitive) ProcessMessage(ctx context.Context, message grblMod.Message) {
	if messagePushGcodeState, ok := message.(*grblMod.MessagePushGcodeState); ok {
		jp.processMessagePushGcodeState(messagePushGcodeState)
		return
	}
	if messagePushStatusReport, ok := message.(*grblMod.MessagePushStatusReport); ok {
		jp.processMessagePushStatusReport(messagePushStatusReport)
		return
	}
}
