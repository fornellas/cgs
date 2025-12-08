package tui

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

	sp.updateStateTextView(UnknownTrackedState)

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

func (sp *StatusPrimitive) clearAll() {
	sp.app.QueueUpdateDraw(func() {
		sp.stateTextView.SetBackgroundColor(tview.Styles.PrimitiveBackgroundColor)
		sp.stateTextView.Clear()
		sp.statusTextView.Clear()
	})
}

func (sp *StatusPrimitive) updateStateTextView(trackedState *TrackedState) {

	stateColor := getMachineStateColor(trackedState.State)

	sp.stateTextView.Clear()
	sp.stateTextView.SetBackgroundColor(stateColor)

	fmt.Fprintf(sp.stateTextView, "%s\n", tview.Escape(string(trackedState.State)))

	if trackedState.SubState != nil {
		fmt.Fprintf(sp.stateTextView, "(%s)\n", tview.Escape(*trackedState.SubState))
	}
}

//gocyclo:ignore
func (sp *StatusPrimitive) writePositionStatus(w io.Writer, statusReportPushMessage *grblMod.StatusReportPushMessage) {
	machineCoordinates := statusReportPushMessage.GetMachineCoordinates(sp.grbl)
	workCoordinates := statusReportPushMessage.GetWorkCoordinates(sp.grbl)
	var nl bool
	if workCoordinates != nil {
		fmt.Fprintf(w, "Work\n")
		nl = true
		fmt.Fprintf(w, "X:%s\n", sprintCoordinate(workCoordinates.X))
		fmt.Fprintf(w, "Y:%s\n", sprintCoordinate(workCoordinates.Y))
		fmt.Fprintf(w, "Z:%s\n", sprintCoordinate(workCoordinates.Z))
		if workCoordinates.A != nil {
			fmt.Fprintf(w, "A:%s\n", sprintCoordinate(*workCoordinates.A))
		}
	}
	if machineCoordinates != nil {
		if nl {
			fmt.Fprintf(w, "\n")
		}
		fmt.Fprintf(w, "Machine\n")
		fmt.Fprintf(w, "X:%s\n", sprintCoordinate(machineCoordinates.X))
		fmt.Fprintf(w, "Y:%s\n", sprintCoordinate(machineCoordinates.Y))
		fmt.Fprintf(w, "Z:%s\n", sprintCoordinate(machineCoordinates.Z))
		if machineCoordinates.A != nil {
			fmt.Fprintf(w, "A:%s\n", sprintCoordinate(*machineCoordinates.A))
		}
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

func (sp *StatusPrimitive) Worker(
	ctx context.Context,
	pushMessageCh <-chan grblMod.PushMessage,
	trackedStateCh <-chan *TrackedState,
) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case pushMessage, ok := <-pushMessageCh:
			if !ok {
				return fmt.Errorf("push message channel closed")
			}
			if _, ok := pushMessage.(*grblMod.WelcomePushMessage); ok {
				sp.clearAll()
			}
			if statusReportPushMessage, ok := pushMessage.(*grblMod.StatusReportPushMessage); ok {
				sp.updateStatusTextView(statusReportPushMessage)
			}
		case trackedState, ok := <-trackedStateCh:
			if !ok {
				return fmt.Errorf("tracked state channel closed")
			}
			sp.app.QueueUpdateDraw(func() {
				sp.updateStateTextView(trackedState)
			})
		}
	}
}
