package control

// Like (G1)
// Args:
// 	One of:
// 		[X$x | Y$y | Z$z]
//  Required:
// 		F$f
//  Optional:
//  	[G20|G21] Inches/Milimiters
// 		[G90|G91] Absolute/Incremental
// 		G53 Machine Coordinates

import (
	"bytes"
	"fmt"
	"math"
	"strconv"

	"github.com/rivo/tview"

	grblMod "github.com/fornellas/cgs/grbl"
)

type JoggingPrimitive struct {
	*tview.Flex
	app              *tview.Application
	controlPrimitive *ControlPrimitive
}

func NewJoggingPrimitive(
	app *tview.Application,
	controlPrimitive *ControlPrimitive,
) *JoggingPrimitive {
	joggingPrimitive := &JoggingPrimitive{
		app:              app,
		controlPrimitive: controlPrimitive,
	}

	acceptFloatFn := func(textToCheck string, lastChar rune) bool {
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
	width := len("100.0000")
	var err error
	var x, y, z float64
	parametersForm.AddInputField("X", "", width, acceptFloatFn, func(text string) {
		x, err = strconv.ParseFloat(text, 64)
		if err != nil {
			x = math.NaN()
		}
	})
	parametersForm.AddInputField("Y", "", width, acceptFloatFn, func(text string) {
		y, err = strconv.ParseFloat(text, 64)
		if err != nil {
			y = math.NaN()
		}
	})
	parametersForm.AddInputField("Z", "", width, acceptFloatFn, func(text string) {
		z, err = strconv.ParseFloat(text, 64)
		if err != nil {
			z = math.NaN()
		}
	})
	// TODO fetch from current gcode parser
	var unitWord string
	parametersForm.AddDropDown("Unit", []string{"Inches", "Milimiters"}, 0, func(option string, optionIndex int) {
		switch option {
		case "Inches":
			unitWord = "G20"
		case "Milimiters":
			unitWord = "G21"
		default:
			panic(fmt.Sprintf("bug: bad unit option: %#v", option))
		}
	})
	// TODO fetch from current gcode parser
	var distanceModeWord string
	parametersForm.AddDropDown("Distance mode", []string{"Absolute", "Incremental"}, 0, func(option string, optionIndex int) {
		switch option {
		case "Absolute":
			unitWord = "G90"
		case "Incremental":
			unitWord = "G91"
		default:
			panic(fmt.Sprintf("bug: bad unit option: %#v", option))
		}
	})
	// TODO fetch from current gcode parser
	var feedRate float64
	parametersForm.AddInputField("Feed rate", "", width, acceptFloatFn, func(text string) {
		feedRate, err = strconv.ParseFloat(text, 64)
		if err != nil {
			feedRate = math.NaN()
		}
	})
	// TODO disable if distance mode is incremental
	var machineCoordinatesWord string
	parametersForm.AddCheckbox("Machine Coordinates", false, func(checked bool) {
		if checked {
			machineCoordinatesWord = "G53"
		} else {
			machineCoordinatesWord = ""
		}
	})
	parametersForm.AddButton("Jog", func() {
		var buf bytes.Buffer
		fmt.Fprintf(&buf, "$J=")
		var hasCoordinate bool
		if !math.IsNaN(x) {
			fmt.Fprintf(&buf, "X%.4f", x)
			hasCoordinate = true
		}
		if !math.IsNaN(y) {
			fmt.Fprintf(&buf, "Y%.4f", y)
			hasCoordinate = true
		}
		if !math.IsNaN(z) {
			fmt.Fprintf(&buf, "Z%.4f", z)
			hasCoordinate = true
		}
		if !hasCoordinate {
			return
		}
		fmt.Fprintf(&buf, "%s%s", unitWord, distanceModeWord)
		if math.IsNaN(feedRate) {
			return
		}
		fmt.Fprintf(&buf, "F%.4f%s", feedRate, machineCoordinatesWord)
		controlPrimitive.QueueCommand(buf.String())
	})
	parametersForm.AddButton("Cancel", func() {
		controlPrimitive.QueueRealTimeCommand(grblMod.RealTimeCommandJogCancel)
	})

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
	joggingPrimitive.Flex = joggingFlex

	return joggingPrimitive
}

func (op *JoggingPrimitive) processMessagePushStatusReport(
	messagePushStatusReport *grblMod.MessagePushStatusReport,
) {
	switch messagePushStatusReport.MachineState.State {
	case "Idle":
	// TODO disable/enable
	case "Run":
	// TODO disable/enable
	case "Hold":
	// TODO disable/enable
	case "Jog":
	// TODO disable/enable
	case "Alarm":
	// TODO disable/enable
	case "Door":
	// TODO disable/enable
	case "Check":
	// TODO disable/enable
	case "Home":
	// TODO disable/enable
	case "Sleep":
	// TODO disable/enable
	default:
		panic(fmt.Sprintf("unknown machine state: %#v", messagePushStatusReport.MachineState.State))
	}
}

func (op *JoggingPrimitive) ProcessMessage(message grblMod.Message) {
	if messagePushStatusReport, ok := message.(*grblMod.MessagePushStatusReport); ok {
		op.processMessagePushStatusReport(messagePushStatusReport)
		return
	}
}
