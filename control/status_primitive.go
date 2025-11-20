package control

import (
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
}

func NewStatusPrimitive(
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
	textView.SetChangedFunc(func() {
		sp.app.Draw()
	})
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
		sp.app.Draw()
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

func (sp *StatusPrimitive) setMachineState(machineState *grblMod.StatusReportMachineState) {
	stateColor := getMachineStateColor(machineState.State)

	sp.app.QueueUpdateDraw(func() {
		sp.stateTextView.Clear()
		sp.stateTextView.SetBackgroundColor(stateColor)
		sp.statusTextView.Clear()
	})
	fmt.Fprintf(
		sp.stateTextView, "%s\n",
		tview.Escape(machineState.State),
	)
	subState := machineState.SubStateString()
	if len(subState) > 0 {
		fmt.Fprintf(
			sp.stateTextView, "(%s)\n",
			tview.Escape(subState),
		)
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
func (sp *StatusPrimitive) updateStatusTextView(
	statusReport *grblMod.MessagePushStatusReport,
) {
	sp.app.QueueUpdateDraw(func() {
		sp.statusTextView.Clear()
	})

	sp.writePositionStatus(sp.statusTextView, statusReport)

	if statusReport.BufferState != nil {
		fmt.Fprint(sp.statusTextView, "\nBuffer\n")
		fmt.Fprintf(sp.statusTextView, "Blocks:%d\n", statusReport.BufferState.AvailableBlocks)
		fmt.Fprintf(sp.statusTextView, "Bytes:%d\n", statusReport.BufferState.AvailableBytes)
	}

	if statusReport.LineNumber != nil {
		fmt.Fprintf(sp.statusTextView, "\nLine:%d\n", *statusReport.LineNumber)
	}

	if statusReport.Feed != nil {
		fmt.Fprintf(sp.statusTextView, "\nFeed:%.1f\n", *statusReport.Feed)
	}

	if statusReport.FeedSpindle != nil {
		if statusReport.FeedSpindle.Feed != 0 || statusReport.FeedSpindle.Speed != 0 {
			fmt.Fprint(sp.statusTextView, "\n")
		}
		if statusReport.FeedSpindle.Feed != 0 {
			fmt.Fprintf(sp.statusTextView, "\nFeed:%.0f\n", statusReport.FeedSpindle.Feed)
		}
		if statusReport.FeedSpindle.Speed != 0 {
			fmt.Fprintf(sp.statusTextView, "\nSpeed:%.0f\n", statusReport.FeedSpindle.Speed)
		}
	}

	if statusReport.PinState != nil {
		fmt.Fprintf(sp.statusTextView, "\nPin:%s\n", statusReport.PinState)
	}

	if sp.grbl.GetOverrideValues() != nil {
		fmt.Fprint(sp.statusTextView, "\nOverrides\n")
		fmt.Fprintf(sp.statusTextView, "Feed:%.0f%%\n", sp.grbl.GetOverrideValues().Feed)
		fmt.Fprintf(sp.statusTextView, "Rapids:%.0f%%\n", sp.grbl.GetOverrideValues().Rapids)
		fmt.Fprintf(sp.statusTextView, "Spindle:%.0f%%\n", sp.grbl.GetOverrideValues().Spindle)
	}

	if statusReport.AccessoryState != nil {
		fmt.Fprint(sp.statusTextView, "\nAccessory\n")
		if statusReport.AccessoryState.SpindleCW != nil && *statusReport.AccessoryState.SpindleCW {
			fmt.Fprint(sp.statusTextView, "Spindle: CW")
		}
		if statusReport.AccessoryState.SpindleCCW != nil && *statusReport.AccessoryState.SpindleCCW {
			fmt.Fprint(sp.statusTextView, "Spindle: CCW")
		}
		if statusReport.AccessoryState.FloodCoolant != nil && *statusReport.AccessoryState.FloodCoolant {
			fmt.Fprint(sp.statusTextView, "Flood Coolant")
		}
		if statusReport.AccessoryState.MistCoolant != nil && *statusReport.AccessoryState.MistCoolant {
			fmt.Fprint(sp.statusTextView, "Mist Coolant")
		}
	}
}

func (sp *StatusPrimitive) processMessagePushStatusReport(
	statusReport *grblMod.MessagePushStatusReport,
) {
	sp.setMachineState(&statusReport.MachineState)
	sp.updateStatusTextView(statusReport)
}

func (sp *StatusPrimitive) ProcessMessage(message grblMod.Message) {
	if _, ok := message.(*grblMod.MessagePushWelcome); ok {
		sp.processMessagePushWelcome()
		return
	}
	if messagePushStatusReport, ok := message.(*grblMod.MessagePushStatusReport); ok {
		sp.processMessagePushStatusReport(messagePushStatusReport)
		return
	}
}
