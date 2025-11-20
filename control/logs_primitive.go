package control

import (
	"github.com/rivo/tview"
)

type LogsPrimitive struct {
	*tview.TextView
	app *tview.Application
}

func NewLogsPrimitive(
	app *tview.Application,
) *LogsPrimitive {
	lp := &LogsPrimitive{
		app: app,
	}

	logsTextView := tview.NewTextView()
	logsTextView.SetBorder(true)
	logsTextView.SetTitle("Logs")
	logsTextView.SetDynamicColors(true)
	logsTextView.SetScrollable(true)
	logsTextView.SetWrap(true)
	logsTextView.SetChangedFunc(func() {
		lp.app.Draw()
	})
	lp.TextView = logsTextView

	return lp
}
