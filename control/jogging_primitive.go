package control

// Like (G1)
// Args:
// 	One of:
// 		[X$x | Y$y | Z$z]
//  Required:
// 		F$f
//  Optional:
//  	[G20|G21]
// 		[G90|G91]
// 		G53

import (
	"fmt"

	"github.com/rivo/tview"

	grblMod "github.com/fornellas/cgs/grbl"
)

type JoggingPrimitive struct {
	*tview.Flex
	controlPrimitive *ControlPrimitive
}

func NewJoggingPrimitive(
	controlPrimitive *ControlPrimitive,
) *JoggingPrimitive {
	joggingPrimitive := &JoggingPrimitive{
		controlPrimitive: controlPrimitive,
	}

	joggingFlex := tview.NewFlex()
	joggingFlex.SetBorder(true)
	joggingFlex.SetTitle("Jogging")
	joggingFlex.SetDirection(tview.FlexRow)
	// TODO add item
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
