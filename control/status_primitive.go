package control

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/rivo/tview"

	grblMod "github.com/fornellas/cgs/grbl"
)

type StatusPrimitive struct {
	*tview.Flex
	grbl           *grblMod.Grbl
	app            *tview.Application
	stateTextView  *tview.TextView
	statusTextView *tview.TextView
	machineState   grblMod.StatusReportMachineState
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
	sp.statusTextView = textView
}

func (sp *StatusPrimitive) processMessagePushWelcome() {
	sp.app.QueueUpdateDraw(func() {
		sp.stateTextView.SetBackgroundColor(tview.Styles.PrimitiveBackgroundColor)
		sp.stateTextView.Clear()
		sp.statusTextView.Clear()
	})
}

func (sp *StatusPrimitive) updateStateTextView(machineState grblMod.StatusReportMachineState) {
	if sp.machineState == machineState {
		return
	}
	sp.machineState = machineState

	stateColor := getMachineStateColor(sp.machineState.State)

	sp.app.QueueUpdateDraw(func() {
		sp.stateTextView.Clear()
		sp.stateTextView.SetBackgroundColor(stateColor)
	})
	fmt.Fprintf(sp.stateTextView, "%s\n", tview.Escape(sp.machineState.State))
	subState := sp.machineState.SubStateString()
	if len(subState) > 0 {
		fmt.Fprintf(sp.stateTextView, "(%s)\n", tview.Escape(subState))
	}
}

//gocyclo:ignore
func (sp *StatusPrimitive) writePositionStatus(w io.Writer, statusReport *grblMod.MessagePushStatusReport) {
	var mx, my, mz, ma, wx, wy, wz, wa *float64
	if statusReport.MachinePosition != nil {
		mx = &statusReport.MachinePosition.X
		my = &statusReport.MachinePosition.Y
		mz = &statusReport.MachinePosition.Z
		ma = statusReport.MachinePosition.A
		if sp.grbl.GetWorkCoordinateOffset() != nil {
			wxv := statusReport.MachinePosition.X - sp.grbl.GetWorkCoordinateOffset().X
			wx = &wxv
			wyv := statusReport.MachinePosition.Y - sp.grbl.GetWorkCoordinateOffset().Y
			wy = &wyv
			wzv := statusReport.MachinePosition.Z - sp.grbl.GetWorkCoordinateOffset().Z
			wz = &wzv
			if statusReport.MachinePosition.A != nil && sp.grbl.GetWorkCoordinateOffset().A != nil {
				wav := *statusReport.MachinePosition.A - *sp.grbl.GetWorkCoordinateOffset().A
				wa = &wav
			}
		}
	}
	if statusReport.WorkPosition != nil {
		wx = &statusReport.WorkPosition.X
		wy = &statusReport.WorkPosition.Y
		wz = &statusReport.WorkPosition.Z
		wa = statusReport.WorkPosition.A
		if sp.grbl.GetWorkCoordinateOffset() != nil {
			mxv := statusReport.WorkPosition.X - sp.grbl.GetWorkCoordinateOffset().X
			mx = &mxv
			myv := statusReport.WorkPosition.Y - sp.grbl.GetWorkCoordinateOffset().Y
			my = &myv
			mzv := statusReport.WorkPosition.Z - sp.grbl.GetWorkCoordinateOffset().Z
			mz = &mzv
			if statusReport.WorkPosition.A != nil && sp.grbl.GetWorkCoordinateOffset().A != nil {
				mav := *statusReport.WorkPosition.A - *sp.grbl.GetWorkCoordinateOffset().A
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
func (sp *StatusPrimitive) updateStatusTextView(statusReport *grblMod.MessagePushStatusReport) {
	var buf bytes.Buffer

	sp.writePositionStatus(&buf, statusReport)

	if statusReport.BufferState != nil {
		fmt.Fprint(&buf, "\nBuffer\n")
		fmt.Fprintf(&buf, "Blocks:%d\n", statusReport.BufferState.AvailableBlocks)
		fmt.Fprintf(&buf, "Bytes:%d\n", statusReport.BufferState.AvailableBytes)
	}

	if statusReport.LineNumber != nil {
		fmt.Fprintf(&buf, "\n\nLine:%d\n", *statusReport.LineNumber)
	}

	if statusReport.Feed != nil {
		fmt.Fprintf(&buf, "\nFeed:%.1f\n", *statusReport.Feed)
	}

	if statusReport.FeedSpindle != nil {
		if statusReport.FeedSpindle.Feed != 0 {
			fmt.Fprintf(&buf, "\nFeed:%.0f\n", statusReport.FeedSpindle.Feed)
		}
		if statusReport.FeedSpindle.Speed != 0 {
			fmt.Fprintf(&buf, "\nSpeed:%.0f\n", statusReport.FeedSpindle.Speed)
		}
	}

	if statusReport.PinState != nil {
		fmt.Fprintf(&buf, "\nPin:%s\n", statusReport.PinState)
	}

	overrideValues := sp.grbl.GetOverrideValues()
	if overrideValues != nil && overrideValues.HasOverride() {
		fmt.Fprint(&buf, "\nOverrides\n")
		if overrideValues.Feed != 100.0 {
			fmt.Fprintf(&buf, "Feed:%.0f%%\n", overrideValues.Feed)
		}
		if overrideValues.Rapids != 100.0 {
			fmt.Fprintf(&buf, "Rapids:%.0f%%\n", overrideValues.Rapids)
		}
		if overrideValues.Spindle != 100.0 {
			fmt.Fprintf(&buf, "Spindle:%.0f%%\n", overrideValues.Spindle)
		}
	}

	accessoryState := sp.grbl.AccessoryState()
	if accessoryState != nil {
		fmt.Fprint(&buf, "\nAccessory\n")
		if accessoryState.SpindleCW != nil && *accessoryState.SpindleCW {
			fmt.Fprint(&buf, "Spindle: CW")
		}
		if accessoryState.SpindleCCW != nil && *accessoryState.SpindleCCW {
			fmt.Fprint(&buf, "Spindle: CCW")
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

func (sp *StatusPrimitive) processMessagePushStatusReport(statusReport *grblMod.MessagePushStatusReport) {
	sp.updateStateTextView(statusReport.MachineState)
	sp.updateStatusTextView(statusReport)
}

func (sp *StatusPrimitive) ProcessMessage(ctx context.Context, message grblMod.Message) {
	if _, ok := message.(*grblMod.MessagePushWelcome); ok {
		sp.processMessagePushWelcome()
		return
	}
	if messagePushStatusReport, ok := message.(*grblMod.MessagePushStatusReport); ok {
		sp.processMessagePushStatusReport(messagePushStatusReport)
		return
	}
}
