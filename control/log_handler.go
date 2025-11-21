package control

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"math"

	"github.com/fornellas/slogxt/log"
)

// ViewLogHandler implements slog.Handler, and proxies calls to either a pre-existing log handler,
// or to a gocui.View.
type ViewLogHandler struct {
	originalHandler slog.Handler
	viewHandler     slog.Handler
}

func NewViewLogHandler(
	originalHandler slog.Handler,
	w io.Writer,
) *ViewLogHandler {
	return &ViewLogHandler{
		originalHandler: originalHandler,
		// TODO try to fetch TerminalHandlerOptions parameters from given handler
		viewHandler: log.NewTerminalTreeHandler(
			w,
			// tview.ANSIWriter(w),
			&log.TerminalHandlerOptions{
				HandlerOptions: slog.HandlerOptions{
					// AddSource: ,
					Level: slog.Level(math.MinInt),
					// ReplaceAttr: ,
				},
				DisableGroupEmoji: true,
				// TimeLayout: ,
				// NoColor: true,
				ForceColor: true,
				// ColorScheme: ,
			},
		),
	}
}

func (h *ViewLogHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.originalHandler.Enabled(ctx, level)
}

func (h *ViewLogHandler) Handle(ctx context.Context, record slog.Record) error {
	return errors.Join(
		h.originalHandler.Handle(ctx, record),
		h.viewHandler.Handle(ctx, record),
	)
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
