package control

import (
	"context"
	"fmt"
	"io"

	"github.com/rivo/tview"

	"github.com/fornellas/slogxt/log"

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
	ctx context.Context,
	grbl *grblMod.Grbl,
	app *tview.Application,
) *StatusPrimitive {
	sp := &StatusPrimitive{
		grbl: grbl,
		app:  app,
	}

	ctx, _ = log.MustWithGroup(ctx, "StatusPrimitive")

	sp.newStateTextView(ctx)
	sp.newStatusTextView(ctx)

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

func (sp *StatusPrimitive) newStateTextView(ctx context.Context) {
	_, logger := log.MustWithGroup(ctx, "StateTextView")
	textView := tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true).
		SetTextAlign(tview.AlignCenter).
		SetWrap(true)
	textView.SetBorder(true).SetTitle("State")
	textView.SetChangedFunc(func() {
		logger.Debug("SetChangedFunc")
	})
	sp.stateTextView = textView
}

func (sp *StatusPrimitive) newStatusTextView(ctx context.Context) {
	_, logger := log.MustWithGroup(ctx, "StatusTextView")
	textView := tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true).
		SetWrap(true)
	textView.SetBorder(true).SetTitle("Status")
	textView.SetChangedFunc(func() {
		logger.Debug("SetChangedFunc")
		textView.ScrollToBeginning()
	})
	sp.statusTextView = textView
}

func (sp *StatusPrimitive) processMessagePushWelcome(ctx context.Context) {
	_, logger := log.MustWithGroup(ctx, "StatusPrimitive.processMessagePushWelcome")
	logger.Debug("Before QueueUpdateDraw")
	sp.app.QueueUpdateDraw(func() {
		logger.Debug("Inside QueueUpdateDraw")
		sp.stateTextView.SetBackgroundColor(tview.Styles.PrimitiveBackgroundColor)
		sp.stateTextView.Clear()
		sp.statusTextView.Clear()
	})
	logger.Debug("After QueueUpdateDraw")
}

func (sp *StatusPrimitive) setMachineState(
	ctx context.Context,
	machineState *grblMod.StatusReportMachineState,
) {
	_, logger := log.MustWithGroup(ctx, "setMachineState")

	stateColor := getMachineStateColor(machineState.State)

	logger.Debug("Before QueueUpdateDraw")
	sp.app.QueueUpdateDraw(func() {
		logger.Debug("Inside QueueUpdateDraw")
		sp.stateTextView.Clear()
		sp.stateTextView.SetBackgroundColor(stateColor)
		sp.statusTextView.Clear()
	})
	logger.Debug("After QueueUpdateDraw")
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
	ctx context.Context,
	statusReport *grblMod.MessagePushStatusReport,
) {
	_, logger := log.MustWithGroup(ctx, "StatusPrimitive.updateStatusTextView")

	logger.Debug("Before QueueUpdateDraw")
	sp.app.QueueUpdateDraw(func() {
		logger.Debug("Inside QueueUpdateDraw")
		sp.statusTextView.Clear()
	})
	logger.Debug("After QueueUpdateDraw")

	sp.writePositionStatus(sp.statusTextView, statusReport)

	if statusReport.BufferState != nil {
		fmt.Fprint(sp.statusTextView, "\nBuffer\n")
		fmt.Fprintf(sp.statusTextView, "Blocks:%d\n", statusReport.BufferState.AvailableBlocks)
		fmt.Fprintf(sp.statusTextView, "Bytes:%d\n", statusReport.BufferState.AvailableBytes)
	}

	if statusReport.LineNumber != nil {
		fmt.Fprintf(sp.statusTextView, "\n\nLine:%d\n", *statusReport.LineNumber)
	}

	if statusReport.Feed != nil {
		fmt.Fprintf(sp.statusTextView, "\nFeed:%.1f\n", *statusReport.Feed)
	}

	if statusReport.FeedSpindle != nil {
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

	overrideValues := sp.grbl.GetOverrideValues()
	if overrideValues != nil && overrideValues.HasOverride() {
		fmt.Fprint(sp.statusTextView, "\nOverrides\n")
		if overrideValues.Feed != 100.0 {
			fmt.Fprintf(sp.statusTextView, "Feed:%.0f%%\n", overrideValues.Feed)
		}
		if overrideValues.Rapids != 100.0 {
			fmt.Fprintf(sp.statusTextView, "Rapids:%.0f%%\n", overrideValues.Rapids)
		}
		if overrideValues.Spindle != 100.0 {
			fmt.Fprintf(sp.statusTextView, "Spindle:%.0f%%\n", overrideValues.Spindle)
		}
	}

	accessoryState := sp.grbl.AccessoryState()
	if accessoryState != nil {
		fmt.Fprint(sp.statusTextView, "\nAccessory\n")
		if accessoryState.SpindleCW != nil && *accessoryState.SpindleCW {
			fmt.Fprint(sp.statusTextView, "Spindle: CW")
		}
		if accessoryState.SpindleCCW != nil && *accessoryState.SpindleCCW {
			fmt.Fprint(sp.statusTextView, "Spindle: CCW")
		}
		if accessoryState.FloodCoolant != nil && *accessoryState.FloodCoolant {
			fmt.Fprint(sp.statusTextView, "Flood Coolant")
		}
		if accessoryState.MistCoolant != nil && *accessoryState.MistCoolant {
			fmt.Fprint(sp.statusTextView, "Mist Coolant")
		}
	}
}

func (sp *StatusPrimitive) processMessagePushStatusReport(
	ctx context.Context,
	statusReport *grblMod.MessagePushStatusReport,
) {
	_, logger := log.MustWithGroup(ctx, "StatusPrimitive.processMessagePushStatusReport")
	logger.Debug("Run")
	sp.setMachineState(ctx, &statusReport.MachineState)
	sp.updateStatusTextView(ctx, statusReport)
}

func (sp *StatusPrimitive) ProcessMessage(ctx context.Context, message grblMod.Message) {
	ctx, logger := log.MustWithGroup(ctx, "StatusPrimitive.ProcessMessage")
	logger.Debug("Start")
	if _, ok := message.(*grblMod.MessagePushWelcome); ok {
		sp.processMessagePushWelcome(ctx)
		return
	}
	if messagePushStatusReport, ok := message.(*grblMod.MessagePushStatusReport); ok {
		sp.processMessagePushStatusReport(ctx, messagePushStatusReport)
		return
	}
	logger.Debug("End")
}
