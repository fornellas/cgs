package control

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	grblMod "github.com/fornellas/cgs/grbl"
)

type StatusPrimitive struct {
	*tview.Flex
	grbl           *grblMod.Grbl
	app            *tview.Application
	stateTextView  *tview.TextView
	statusTextView *tview.TextView
	machineState   grblMod.MachineState
}

func NewStatusPrimitive(
	ctx context.Context,
	grbl *grblMod.Grbl,
	app *tview.Application,
) *StatusPrimitive {
	sp := &StatusPrimitive{
		grbl: grbl,
		app:  app,
	}

	sp.newStateTextView()
	sp.newStatusTextView()

	statusFlex := tview.NewFlex()
	statusFlex.SetDirection(tview.FlexRow)
	statusFlex.AddItem(sp.stateTextView, 4, 0, false)
	statusFlex.AddItem(sp.statusTextView, 0, 1, false)
	sp.Flex = statusFlex

	return sp
}

func (sp *StatusPrimitive) FixedSize() int {
	return 14
}

func (sp *StatusPrimitive) newStateTextView() {
	textView := tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true).
		SetTextAlign(tview.AlignCenter).
		SetWrap(true)
	textView.SetBorder(true).SetTitle("State")
	textView.SetChangedFunc(func() {})
	sp.stateTextView = textView
}

func (sp *StatusPrimitive) newStatusTextView() {
	textView := tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true).
		SetWrap(true)
	textView.SetBorder(true).SetTitle("Status")
	textView.SetChangedFunc(func() {
		textView.ScrollToBeginning()
	})
	textView.SetTextAlign(tview.AlignCenter)
	sp.statusTextView = textView
}

func (sp *StatusPrimitive) processMessagePushWelcome() {
	sp.app.QueueUpdateDraw(func() {
		sp.machineState = grblMod.MachineState{}
		sp.stateTextView.SetBackgroundColor(tview.Styles.PrimitiveBackgroundColor)
		sp.stateTextView.Clear()
		sp.statusTextView.Clear()
	})
}

func (sp *StatusPrimitive) updateStateTextView(machineState grblMod.MachineState) {
	if sp.machineState == machineState {
		return
	}
	sp.machineState = machineState

	stateColor := getMachineStateColor(sp.machineState.State)

	sp.app.QueueUpdateDraw(func() {
		sp.stateTextView.Clear()
		sp.stateTextView.SetBackgroundColor(stateColor)
	})
	fmt.Fprintf(sp.stateTextView, "%s\n", tview.Escape(string(sp.machineState.State)))
	subState := sp.machineState.SubStateString()
	if len(subState) > 0 {
		fmt.Fprintf(sp.stateTextView, "(%s)\n", tview.Escape(subState))
	}
}

//gocyclo:ignore
func (sp *StatusPrimitive) writePositionStatus(w io.Writer, statusReportPushMessage *grblMod.StatusReportPushMessage) {
	var mx, my, mz, ma, wx, wy, wz, wa *float64
	if statusReportPushMessage.MachinePosition != nil {
		mx = &statusReportPushMessage.MachinePosition.X
		my = &statusReportPushMessage.MachinePosition.Y
		mz = &statusReportPushMessage.MachinePosition.Z
		ma = statusReportPushMessage.MachinePosition.A
		if sp.grbl.GetLastWorkCoordinateOffset() != nil {
			wxv := statusReportPushMessage.MachinePosition.X - sp.grbl.GetLastWorkCoordinateOffset().X
			wx = &wxv
			wyv := statusReportPushMessage.MachinePosition.Y - sp.grbl.GetLastWorkCoordinateOffset().Y
			wy = &wyv
			wzv := statusReportPushMessage.MachinePosition.Z - sp.grbl.GetLastWorkCoordinateOffset().Z
			wz = &wzv
			if statusReportPushMessage.MachinePosition.A != nil && sp.grbl.GetLastWorkCoordinateOffset().A != nil {
				wav := *statusReportPushMessage.MachinePosition.A - *sp.grbl.GetLastWorkCoordinateOffset().A
				wa = &wav
			}
		}
	}
	if statusReportPushMessage.WorkPosition != nil {
		wx = &statusReportPushMessage.WorkPosition.X
		wy = &statusReportPushMessage.WorkPosition.Y
		wz = &statusReportPushMessage.WorkPosition.Z
		wa = statusReportPushMessage.WorkPosition.A
		if sp.grbl.GetLastWorkCoordinateOffset() != nil {
			mxv := statusReportPushMessage.WorkPosition.X - sp.grbl.GetLastWorkCoordinateOffset().X
			mx = &mxv
			myv := statusReportPushMessage.WorkPosition.Y - sp.grbl.GetLastWorkCoordinateOffset().Y
			my = &myv
			mzv := statusReportPushMessage.WorkPosition.Z - sp.grbl.GetLastWorkCoordinateOffset().Z
			mz = &mzv
			if statusReportPushMessage.WorkPosition.A != nil && sp.grbl.GetLastWorkCoordinateOffset().A != nil {
				mav := *statusReportPushMessage.WorkPosition.A - *sp.grbl.GetLastWorkCoordinateOffset().A
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
		fmt.Fprintf(w, "X:%s\n", sprintCoordinate(*wx))
	}
	if wy != nil {
		fmt.Fprintf(w, "Y:%s\n", sprintCoordinate(*wy))
	}
	if wz != nil {
		fmt.Fprintf(w, "Z:%s\n", sprintCoordinate(*wz))
	}
	if wa != nil {
		fmt.Fprintf(w, "A:%s\n", sprintCoordinate(*wa))
	}
	if mx != nil || my != nil || mz != nil || ma != nil {
		if nl {
			fmt.Fprintf(w, "\n")
		}
		fmt.Fprintf(w, "Machine\n")
	}
	if mx != nil {
		fmt.Fprintf(w, "X:%s\n", sprintCoordinate(*mx))
	}
	if my != nil {
		fmt.Fprintf(w, "Y:%s\n", sprintCoordinate(*my))
	}
	if mz != nil {
		fmt.Fprintf(w, "Z:%s\n", sprintCoordinate(*mz))
	}
	if ma != nil {
		fmt.Fprintf(w, "A:%s\n", sprintCoordinate(*ma))
	}
}

//gocyclo:ignore
func (sp *StatusPrimitive) updateStatusTextView(statusReportPushMessage *grblMod.StatusReportPushMessage) {
	var buf bytes.Buffer

	sp.writePositionStatus(&buf, statusReportPushMessage)

	if statusReportPushMessage.BufferState != nil {
		fmt.Fprint(&buf, "\nBuffer\n")
		fmt.Fprintf(&buf, "Blocks:%s\n", sprintBlocks(statusReportPushMessage.BufferState.AvailableBlocks))
		fmt.Fprintf(&buf, "Bytes:%s\n", sprintBytes(statusReportPushMessage.BufferState.AvailableBytes))
	}

	if statusReportPushMessage.LineNumber != nil {
		fmt.Fprintf(&buf, "\n\nLine:%s\n", sprintLine(int(*statusReportPushMessage.LineNumber)))
	}

	if statusReportPushMessage.Feed != nil {
		fmt.Fprintf(&buf, "\nFeed:%s\n", sprintFeed(float64(*statusReportPushMessage.Feed)))
	}

	if statusReportPushMessage.FeedSpindle != nil {
		if statusReportPushMessage.FeedSpindle.Feed != 0 {
			fmt.Fprintf(&buf, "\nFeed:%s\n", sprintFeed(statusReportPushMessage.FeedSpindle.Feed))
		}
		if statusReportPushMessage.FeedSpindle.Speed != 0 {
			fmt.Fprintf(&buf, "\nSpeed:%s\n", sprintSpeed(statusReportPushMessage.FeedSpindle.Speed))
		}
	}

	if statusReportPushMessage.PinState != nil {
		fmt.Fprintf(&buf, "\nPin:[%s]%s[-]\n", tcell.ColorOrange, statusReportPushMessage.PinState)
	}

	overrideValues := sp.grbl.GetLastOverrideValues()
	if overrideValues != nil && overrideValues.HasOverride() {
		fmt.Fprint(&buf, "\nOverrides\n")
		if overrideValues.Feed != 100.0 {
			fmt.Fprintf(&buf, "Feed:%s%%\n", sprintFeed(overrideValues.Feed))
		}
		if overrideValues.Rapids != 100.0 {
			fmt.Fprintf(&buf, "Rapids:%s%%\n", sprintFeed(overrideValues.Rapids))
		}
		if overrideValues.Spindle != 100.0 {
			fmt.Fprintf(&buf, "Spindle:%s%%\n", sprintSpindle(overrideValues.Spindle))
		}
	}

	accessoryState := sp.grbl.GetLastAccessoryState()
	if accessoryState != nil {
		fmt.Fprint(&buf, "\nAccessory\n")
		if accessoryState.SpindleCW != nil && *accessoryState.SpindleCW {
			fmt.Fprintf(&buf, "Spindle: [%s]CW[-]", tcell.ColorOrange)
		}
		if accessoryState.SpindleCCW != nil && *accessoryState.SpindleCCW {
			fmt.Fprintf(&buf, "Spindle: [%s]CCW[-]", tcell.ColorOrange)
		}
		if accessoryState.FloodCoolant != nil && *accessoryState.FloodCoolant {
			fmt.Fprint(&buf, "Flood Coolant")
		}
		if accessoryState.MistCoolant != nil && *accessoryState.MistCoolant {
			fmt.Fprint(&buf, "Mist Coolant")
		}
	}

	sp.app.QueueUpdateDraw(func() {
		if buf.String() == sp.statusTextView.GetText(false) {
			return
		}
		sp.statusTextView.SetText(buf.String())
	})
}

func (sp *StatusPrimitive) processStatusReportPushMessage(statusReportPushMessage *grblMod.StatusReportPushMessage) {
	sp.updateStateTextView(statusReportPushMessage.MachineState)
	sp.updateStatusTextView(statusReportPushMessage)
}

func (sp *StatusPrimitive) ProcessPushMessage(ctx context.Context, pushMessage grblMod.PushMessage) {
	if _, ok := pushMessage.(*grblMod.WelcomePushMessage); ok {
		sp.processMessagePushWelcome()
		return
	}
	if statusReportPushMessage, ok := pushMessage.(*grblMod.StatusReportPushMessage); ok {
		sp.processStatusReportPushMessage(statusReportPushMessage)
		return
	}
}
