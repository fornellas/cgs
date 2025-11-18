package control

import (
	"context"
	"log/slog"
	"math"

	"github.com/fornellas/slogxt/log"
	"github.com/rivo/tview"
)

// ViewLogHandler implements slog.Handler, and proxies calls to either a pre-existing log handler,
// or to a gocui.View.
type ViewLogHandler struct {
	originalHandler slog.Handler
	viewHandler     slog.Handler
	textView        *tview.TextView
}

func NewViewLogHandler(
	originalHandler slog.Handler,
	textView *tview.TextView,
) *ViewLogHandler {
	viewLogHandler := &ViewLogHandler{
		originalHandler: originalHandler,
		textView:        textView,
	}
	// TODO try to fetch TerminalHandlerOptions parameters from given handler
	viewLogHandler.viewHandler = log.NewTerminalLineHandler(viewLogHandler, &log.TerminalHandlerOptions{
		HandlerOptions: slog.HandlerOptions{
			// AddSource: ,
			Level: slog.Level(math.MinInt),
			// ReplaceAttr: ,
		},
		// TimeLayout: ,
		// ForceColor: true,
		NoColor: true,
		// ColorScheme: ,
	})
	return viewLogHandler
}

func (h *ViewLogHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.originalHandler.Enabled(ctx, level)
}

func (h *ViewLogHandler) Handle(ctx context.Context, record slog.Record) error {
	return h.viewHandler.Handle(ctx, record)
}

func (h *ViewLogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &ViewLogHandler{
		originalHandler: h.originalHandler.WithAttrs(attrs),
		viewHandler:     h.viewHandler.WithAttrs(attrs),
	}
}

func (h *ViewLogHandler) WithGroup(name string) slog.Handler {
	return &ViewLogHandler{
		originalHandler: h.originalHandler.WithGroup(name),
		viewHandler:     h.viewHandler.WithGroup(name),
	}
}

func (h *ViewLogHandler) Write(b []byte) (int, error) {
	return h.textView.Write(b)
}
