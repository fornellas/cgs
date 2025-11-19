package control

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	grblMod "github.com/fornellas/cgs/grbl"
)

type MessageProcessor struct {
	grbl                       *grblMod.Grbl
	appManager                 *AppManager
	pushMessageCh              chan grblMod.Message
	commandDispatcher          *CommandDispatcher
	quietGcodeParserStateComms bool
	quietGcodeParamStateComms  bool
	quietStatusComms           bool
}

func NewMessageProcessor(
	grbl *grblMod.Grbl,
	appManager *AppManager,
	pushMessageCh chan grblMod.Message,
	commandDispatcher *CommandDispatcher,
	quietGcodeParserStateComms bool,
	quietGcodeParamStateComms bool,
	quietStatusComms bool,
) *MessageProcessor {
	return &MessageProcessor{
		grbl:                       grbl,
		appManager:                 appManager,
		pushMessageCh:              pushMessageCh,
		commandDispatcher:          commandDispatcher,
		quietGcodeParserStateComms: quietGcodeParserStateComms,
		quietGcodeParamStateComms:  quietGcodeParamStateComms,
		quietStatusComms:           quietStatusComms,
	}
}

//gocyclo:ignore
func (mp *MessageProcessor) writePositionStatus(w io.Writer, statusReport *grblMod.MessagePushStatusReport) {
	var mx, my, mz, ma, wx, wy, wz, wa *float64
	if statusReport.MachinePosition != nil {
		mx = &statusReport.MachinePosition.X
		my = &statusReport.MachinePosition.Y
		mz = &statusReport.MachinePosition.Z
		ma = statusReport.MachinePosition.A
		if mp.grbl.GetWorkCoordinateOffset() != nil {
			wxv := statusReport.MachinePosition.X - mp.grbl.GetWorkCoordinateOffset().X
			wx = &wxv
			wyv := statusReport.MachinePosition.Y - mp.grbl.GetWorkCoordinateOffset().Y
			wy = &wyv
			wzv := statusReport.MachinePosition.Z - mp.grbl.GetWorkCoordinateOffset().Z
			wz = &wzv
			if statusReport.MachinePosition.A != nil && mp.grbl.GetWorkCoordinateOffset().A != nil {
				wav := *statusReport.MachinePosition.A - *mp.grbl.GetWorkCoordinateOffset().A
				wa = &wav
			}
		}
	}
	if statusReport.WorkPosition != nil {
		wx = &statusReport.WorkPosition.X
		wy = &statusReport.WorkPosition.Y
		wz = &statusReport.WorkPosition.Z
		wa = statusReport.WorkPosition.A
		if mp.grbl.GetWorkCoordinateOffset() != nil {
			mxv := statusReport.WorkPosition.X - mp.grbl.GetWorkCoordinateOffset().X
			mx = &mxv
			myv := statusReport.WorkPosition.Y - mp.grbl.GetWorkCoordinateOffset().Y
			my = &myv
			mzv := statusReport.WorkPosition.Z - mp.grbl.GetWorkCoordinateOffset().Z
			mz = &mzv
			if statusReport.WorkPosition.A != nil && mp.grbl.GetWorkCoordinateOffset().A != nil {
				mav := *statusReport.WorkPosition.A - *mp.grbl.GetWorkCoordinateOffset().A
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

func (mp *MessageProcessor) updateState(
	state string,
	subState string,
) {
	stateColor := getMachineStateColor(state)

	mp.appManager.StateTextView.Clear()
	mp.appManager.StateTextView.SetBackgroundColor(stateColor)
	fmt.Fprintf(
		mp.appManager.StateTextView, "%s\n",
		tview.Escape(state),
	)
	if len(subState) > 0 {
		fmt.Fprintf(
			mp.appManager.StateTextView, "(%s)\n",
			tview.Escape(subState),
		)
	}
}

//gocyclo:ignore
func (mp *MessageProcessor) updateStatusReport(
	statusReport *grblMod.MessagePushStatusReport,
) {
	mp.updateState(statusReport.MachineState.State, statusReport.MachineState.SubStateString())

	mp.appManager.StatusTextView.Clear()

	mp.writePositionStatus(mp.appManager.StatusTextView, statusReport)

	if statusReport.BufferState != nil {
		fmt.Fprint(mp.appManager.StatusTextView, "\nBuffer\n")
		fmt.Fprintf(mp.appManager.StatusTextView, "Blocks:%d\n", statusReport.BufferState.AvailableBlocks)
		fmt.Fprintf(mp.appManager.StatusTextView, "Bytes:%d\n", statusReport.BufferState.AvailableBytes)
	}

	if statusReport.LineNumber != nil {
		fmt.Fprintf(mp.appManager.StatusTextView, "\nLine:%d\n", *statusReport.LineNumber)
	}

	if statusReport.Feed != nil {
		fmt.Fprintf(mp.appManager.StatusTextView, "\nFeed:%.1f\n", *statusReport.Feed)
	}

	if statusReport.FeedSpindle != nil {
		fmt.Fprintf(mp.appManager.StatusTextView, "\nFeed:%.0f\n", statusReport.FeedSpindle.Feed)
		fmt.Fprintf(mp.appManager.StatusTextView, "Speed:%.0f\n", statusReport.FeedSpindle.Speed)
	}

	if statusReport.PinState != nil {
		fmt.Fprintf(mp.appManager.StatusTextView, "\nPin:%s\n", statusReport.PinState)
	}

	if mp.grbl.GetOverrideValues() != nil {
		fmt.Fprint(mp.appManager.StatusTextView, "\nOverrides\n")
		fmt.Fprintf(mp.appManager.StatusTextView, "Feed:%.0f%%\n", mp.grbl.GetOverrideValues().Feed)
		fmt.Fprintf(mp.appManager.StatusTextView, "Rapids:%.0f%%\n", mp.grbl.GetOverrideValues().Rapids)
		fmt.Fprintf(mp.appManager.StatusTextView, "Spindle:%.0f%%\n", mp.grbl.GetOverrideValues().Spindle)
	}

	if statusReport.AccessoryState != nil {
		fmt.Fprint(mp.appManager.StatusTextView, "\nAccessory\n")
		if statusReport.AccessoryState.SpindleCW != nil && *statusReport.AccessoryState.SpindleCW {
			fmt.Fprint(mp.appManager.StatusTextView, "Spindle: CW")
		}
		if statusReport.AccessoryState.SpindleCCW != nil && *statusReport.AccessoryState.SpindleCCW {
			fmt.Fprint(mp.appManager.StatusTextView, "Spindle: CCW")
		}
		if statusReport.AccessoryState.FloodCoolant != nil && *statusReport.AccessoryState.FloodCoolant {
			fmt.Fprint(mp.appManager.StatusTextView, "Flood Coolant")
		}
		if statusReport.AccessoryState.MistCoolant != nil && *statusReport.AccessoryState.MistCoolant {
			fmt.Fprint(mp.appManager.StatusTextView, "Mist Coolant")
		}
	}
}

//gocyclo:ignore
func (mp *MessageProcessor) processMessagePushGcodeState(
	messagePushGcodeState *grblMod.MessagePushGcodeState,
) (func(), tcell.Color) {
	var buf bytes.Buffer

	if modalGroup := messagePushGcodeState.ModalGroup; modalGroup != nil {
		if modalGroup.Motion != nil {
			fmt.Fprintf(&buf, "%s:%s\n", modalGroup.Motion.NormalizedString(), modalGroup.Motion.Name())
		}
		if modalGroup.PlaneSelection != nil {
			fmt.Fprintf(&buf, "%s:%s\n", modalGroup.PlaneSelection.NormalizedString(), modalGroup.PlaneSelection.Name())
		}
		if modalGroup.DistanceMode != nil {
			fmt.Fprintf(&buf, "%s:%s\n", modalGroup.DistanceMode.NormalizedString(), modalGroup.DistanceMode.Name())
		}
		if modalGroup.FeedRateMode != nil {
			fmt.Fprintf(&buf, "%s:%s\n", modalGroup.FeedRateMode.NormalizedString(), modalGroup.FeedRateMode.Name())
		}
		if modalGroup.Units != nil {
			fmt.Fprintf(&buf, "%s:%s\n", messagePushGcodeState.ModalGroup.Units.NormalizedString(), messagePushGcodeState.ModalGroup.Units.Name())
		}
		if modalGroup.CutterRadiusCompensation != nil {
			fmt.Fprintf(&buf, "%s:%s\n", modalGroup.CutterRadiusCompensation.NormalizedString(), modalGroup.CutterRadiusCompensation.Name())
		}
		if modalGroup.ToolLengthOffset != nil {
			fmt.Fprintf(&buf, "%s:%s\n", modalGroup.ToolLengthOffset.NormalizedString(), modalGroup.ToolLengthOffset.Name())
		}
		if modalGroup.CoordinateSystemSelection != nil {
			fmt.Fprintf(&buf, "%s:%s\n", modalGroup.CoordinateSystemSelection.NormalizedString(), modalGroup.CoordinateSystemSelection.Name())
		}
		if modalGroup.Stopping != nil {
			fmt.Fprintf(&buf, "%s:%s\n", modalGroup.Stopping.NormalizedString(), modalGroup.Stopping.Name())
		}
		if modalGroup.SpindleTurning != nil {
			fmt.Fprintf(&buf, "%s:%s\n", modalGroup.SpindleTurning.NormalizedString(), modalGroup.SpindleTurning.Name())
		}
		for _, word := range modalGroup.Coolant {
			fmt.Fprintf(&buf, "%s:%s\n", word.NormalizedString(), word.Name())
		}
	}

	if messagePushGcodeState.Tool != nil {
		fmt.Fprintf(&buf, "Tool: %.0f\n", *messagePushGcodeState.Tool)
	}
	if messagePushGcodeState.FeedRate != nil {
		fmt.Fprintf(&buf, "Feed Rate: %.0f\n", *messagePushGcodeState.FeedRate)
	}
	if messagePushGcodeState.SpindleSpeed != nil {
		fmt.Fprintf(&buf, "Speed: %.0f\n", *messagePushGcodeState.SpindleSpeed)
	}

	mp.appManager.GcodeParserTextView.Clear()
	fmt.Fprint(mp.appManager.GcodeParserTextView, tview.Escape(buf.String()))

	return nil, tcell.ColorGreen
}

//gocyclo:ignore
func (mp *MessageProcessor) processMessagePushGcodeParam() (func(), tcell.Color) {
	color := tcell.ColorGreen

	params := mp.grbl.GetGcodeParameters()
	if params == nil {
		return nil, color
	}

	var buf bytes.Buffer

	if params.CoordinateSystem1 != nil {
		fmt.Fprintf(&buf, "G54:Coordinate System 1\n")
		fmt.Fprintf(&buf, "X:%.4f Y:%.4f Z:%.4f", params.CoordinateSystem1.X, params.CoordinateSystem1.Y, params.CoordinateSystem1.Z)
		if params.CoordinateSystem1.A != nil {
			fmt.Fprintf(&buf, " A:%.4f", *params.CoordinateSystem1.A)
		}
		fmt.Fprintf(&buf, "\n")
	}
	if params.CoordinateSystem2 != nil {
		fmt.Fprintf(&buf, "G55:Coordinate System 2\n")
		fmt.Fprintf(&buf, "X:%.4f Y:%.4f Z:%.4f", params.CoordinateSystem2.X, params.CoordinateSystem2.Y, params.CoordinateSystem2.Z)
		if params.CoordinateSystem2.A != nil {
			fmt.Fprintf(&buf, " A:%.4f", *params.CoordinateSystem2.A)
		}
		fmt.Fprintf(&buf, "\n")
	}
	if params.CoordinateSystem3 != nil {
		fmt.Fprintf(&buf, "G56:Coordinate System 3\n")
		fmt.Fprintf(&buf, "X:%.4f Y:%.4f Z:%.4f", params.CoordinateSystem3.X, params.CoordinateSystem3.Y, params.CoordinateSystem3.Z)
		if params.CoordinateSystem3.A != nil {
			fmt.Fprintf(&buf, " A:%.4f", *params.CoordinateSystem3.A)
		}
		fmt.Fprintf(&buf, "\n")
	}
	if params.CoordinateSystem4 != nil {
		fmt.Fprintf(&buf, "G57:Coordinate System 4\n")
		fmt.Fprintf(&buf, "X:%.4f Y:%.4f Z:%.4f", params.CoordinateSystem4.X, params.CoordinateSystem4.Y, params.CoordinateSystem4.Z)
		if params.CoordinateSystem4.A != nil {
			fmt.Fprintf(&buf, " A:%.4f", *params.CoordinateSystem4.A)
		}
		fmt.Fprintf(&buf, "\n")
	}
	if params.CoordinateSystem5 != nil {
		fmt.Fprintf(&buf, "G58:Coordinate System 5\n")
		fmt.Fprintf(&buf, "X:%.4f Y:%.4f Z:%.4f", params.CoordinateSystem5.X, params.CoordinateSystem5.Y, params.CoordinateSystem5.Z)
		if params.CoordinateSystem5.A != nil {
			fmt.Fprintf(&buf, " A:%.4f", *params.CoordinateSystem5.A)
		}
		fmt.Fprintf(&buf, "\n")
	}
	if params.CoordinateSystem6 != nil {
		fmt.Fprintf(&buf, "G59:Coordinate System 6\n")
		fmt.Fprintf(&buf, "X:%.4f Y:%.4f Z:%.4f", params.CoordinateSystem6.X, params.CoordinateSystem6.Y, params.CoordinateSystem6.Z)
		if params.CoordinateSystem6.A != nil {
			fmt.Fprintf(&buf, " A:%.4f", *params.CoordinateSystem6.A)
		}
		fmt.Fprintf(&buf, "\n")
	}
	if params.PrimaryPreDefinedPosition != nil {
		fmt.Fprintf(&buf, "G28:Primary Pre-Defined Position\n")
		fmt.Fprintf(&buf, "X:%.4f Y:%.4f Z:%.4f", params.PrimaryPreDefinedPosition.X, params.PrimaryPreDefinedPosition.Y, params.PrimaryPreDefinedPosition.Z)
		if params.PrimaryPreDefinedPosition.A != nil {
			fmt.Fprintf(&buf, " A:%.4f", *params.PrimaryPreDefinedPosition.A)
		}
		fmt.Fprintf(&buf, "\n")
	}
	if params.SecondaryPreDefinedPosition != nil {
		fmt.Fprintf(&buf, "G30:Secondary Pre-Defined Position\n")
		fmt.Fprintf(&buf, "X:%.4f Y:%.4f Z:%.4f", params.SecondaryPreDefinedPosition.X, params.SecondaryPreDefinedPosition.Y, params.SecondaryPreDefinedPosition.Z)
		if params.SecondaryPreDefinedPosition.A != nil {
			fmt.Fprintf(&buf, " A:%.4f", *params.SecondaryPreDefinedPosition.A)
		}
		fmt.Fprintf(&buf, "\n")
	}
	if params.CoordinateOffset != nil {
		fmt.Fprintf(&buf, "G92:Coordinate Offset\n")
		fmt.Fprintf(&buf, "X:%.4f Y:%.4f Z:%.4f", params.CoordinateOffset.X, params.CoordinateOffset.Y, params.CoordinateOffset.Z)
		if params.CoordinateOffset.A != nil {
			fmt.Fprintf(&buf, " A:%.4f", *params.CoordinateOffset.A)
		}
		fmt.Fprintf(&buf, "\n")
	}
	if params.ToolLengthOffset != nil {
		fmt.Fprintf(&buf, "Tool Length Offset\n")
		fmt.Fprintf(&buf, "Z:%.4f\n", *params.ToolLengthOffset)
	}
	if params.Probe != nil {
		fmt.Fprintf(&buf, "Last Probing Cycle\n")
		fmt.Fprintf(&buf, "X:%.4f Y:%.4f Z:%.4f", params.Probe.Coordinates.X, params.Probe.Coordinates.Y, params.Probe.Coordinates.Z)
		if params.Probe.Coordinates.A != nil {
			fmt.Fprintf(&buf, " A:%.4f", *params.Probe.Coordinates.A)
		}
		fmt.Fprintf(&buf, "\n")
		fmt.Fprintf(&buf, "Successful: %v\n", params.Probe.Successful)
	}

	mp.appManager.GcodeParamsTextView.Clear()
	fmt.Fprint(mp.appManager.GcodeParamsTextView, tview.Escape(buf.String()))

	return nil, color
}

func (mp *MessageProcessor) processMessagePushWelcome(
	_ *grblMod.MessagePushWelcome,
) (func(), tcell.Color) {
	color := tcell.ColorYellow
	detailsFn := func() {
		fmt.Fprintf(mp.appManager.PushMessagesLogsTextView, "[%s]Soft-Reset detected[-]\n", color)
	}
	mp.appManager.GcodeParserTextView.Clear()
	mp.appManager.GcodeParamsTextView.Clear()
	mp.appManager.StateTextView.SetBackgroundColor(tview.Styles.PrimitiveBackgroundColor)
	mp.appManager.StateTextView.Clear()
	mp.appManager.StatusTextView.Clear()
	mp.appManager.FeedbackTextView.SetText("")
	// Sending $G enables tracking of G-Code parsing state
	mp.commandDispatcher.QueueCommand("$G")
	// Sending $G enables tracking of G-Code parameters
	mp.commandDispatcher.QueueCommand("$#")
	return detailsFn, color
}

func (mp *MessageProcessor) processMessagePushAlarm(
	messagePushAlarm *grblMod.MessagePushAlarm,
) (func(), tcell.Color) {
	color := tcell.ColorRed
	detailsFn := func() {
		fmt.Fprintf(mp.appManager.PushMessagesLogsTextView, "[%s]%s[-]\n", color, tview.Escape(messagePushAlarm.Error().Error()))
	}
	mp.updateState("Alarm", "")
	return detailsFn, color
}

func (mp *MessageProcessor) processMessagePushStatusReport(
	messagePushStatusReport *grblMod.MessagePushStatusReport,
) (func(), tcell.Color) {
	color := getMachineStateColor(messagePushStatusReport.MachineState.State)
	mp.updateStatusReport(messagePushStatusReport)
	return nil, color
}

func (mp *MessageProcessor) processMessagePushFeedback(
	messagePushFeedback *grblMod.MessagePushFeedback,
) (func(), tcell.Color) {
	mp.appManager.FeedbackTextView.SetText(messagePushFeedback.Text())
	return nil, tcell.ColorGreen
}

//gocyclo:ignore
func (mp *MessageProcessor) Run(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			err := ctx.Err()
			if errors.Is(err, context.Canceled) {
				err = nil
			}
			return err
		case message, ok := <-mp.pushMessageCh:
			if !ok {
				return fmt.Errorf("push message channel closed")
			}

			var color = tcell.ColorGreen
			var detailsFn func()
			if messagePushGcodeState, ok := message.(*grblMod.MessagePushGcodeState); ok {
				detailsFn, color = mp.processMessagePushGcodeState(messagePushGcodeState)
				if mp.quietGcodeParserStateComms {
					continue
				}
			}
			if _, ok := message.(*grblMod.MessagePushGcodeParam); ok {
				detailsFn, color = mp.processMessagePushGcodeParam()
				if mp.quietGcodeParamStateComms {
					continue
				}
			}

			if messagePushWelcome, ok := message.(*grblMod.MessagePushWelcome); ok {
				detailsFn, color = mp.processMessagePushWelcome(messagePushWelcome)
			}
			if messagePushAlarm, ok := message.(*grblMod.MessagePushAlarm); ok {
				detailsFn, color = mp.processMessagePushAlarm(messagePushAlarm)
			}
			if messagePushStatusReport, ok := message.(*grblMod.MessagePushStatusReport); ok {
				detailsFn, color = mp.processMessagePushStatusReport(messagePushStatusReport)
				if mp.quietStatusComms {
					continue
				}
			}
			if messagePushFeedback, ok := message.(*grblMod.MessagePushFeedback); ok {
				detailsFn, color = mp.processMessagePushFeedback(messagePushFeedback)
			}

			text := message.String()
			if len(text) == 0 {
				fmt.Fprintf(mp.appManager.PushMessagesLogsTextView, "\n\n")
			} else {
				fmt.Fprintf(mp.appManager.PushMessagesLogsTextView, "[%s]%s[-]\n", color, tview.Escape(text))
			}
			if detailsFn != nil {
				detailsFn()
			}
		}
	}
}
