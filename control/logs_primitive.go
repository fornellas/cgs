package control

import (
	"context"

	"github.com/fornellas/slogxt/log"

	"github.com/rivo/tview"
)

type LogsPrimitive struct {
	*tview.TextView
	app *tview.Application
}

func NewLogsPrimitive(
	ctx context.Context,
	app *tview.Application,
) *LogsPrimitive {
	lp := &LogsPrimitive{
		app: app,
	}
	_, logger := log.MustWithGroup(ctx, "LogsPrimitive")

	logsTextView := tview.NewTextView()
	logsTextView.SetBorder(true)
	logsTextView.SetTitle("Logs")
	logsTextView.SetDynamicColors(true)
	logsTextView.SetScrollable(true)
	logsTextView.SetWrap(true)
	logsTextView.SetChangedFunc(func() {
		logger.Debug("SetChangedFunc")
		lp.app.Draw()
	})
	lp.TextView = logsTextView

	return lp
}
