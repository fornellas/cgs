package grbl

import (
	"context"
	"fmt"
	"log/slog"
	"math"

	"github.com/fornellas/slogxt/log"
	"github.com/jroimartin/gocui"
)

// ViewHandler implements slog.Handler, and proxies calls to either a pre-existing log handler,
// or to a gocui.View.
type ViewHandler struct {
	originalHandler slog.Handler
	viewHandler     slog.Handler
	gui             **gocui.Gui
	viewName        string
}

func NewViewHandler(
	originalHandler slog.Handler,
	gui **gocui.Gui,
	viewName string,
) *ViewHandler {
	h := &ViewHandler{
		originalHandler: originalHandler,
		gui:             gui,
		viewName:        viewName,
	}

	// TODO try to fetch TerminalHandlerOptions parameters from given handler
	h.viewHandler = log.NewTerminalLineHandler(h, &log.TerminalHandlerOptions{
		HandlerOptions: slog.HandlerOptions{
			// AddSource: ,
			Level: slog.Level(math.MinInt),
			// ReplaceAttr: ,
		},
		// TimeLayout: ,
		ForceColor: true,
		// ColorScheme: ,
	})

	return h
}

func (h *ViewHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.originalHandler.Enabled(ctx, level)
}

func (h *ViewHandler) Handle(ctx context.Context, record slog.Record) error {
	gui := (*h.gui)
	if gui == nil {
		return h.originalHandler.Handle(ctx, record)
	}
	return h.viewHandler.Handle(ctx, record)
}

func (h *ViewHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &ViewHandler{
		originalHandler: h.originalHandler.WithAttrs(attrs),
		viewHandler:     h.viewHandler.WithAttrs(attrs),
		gui:             h.gui,
		viewName:        h.viewName,
	}
}

func (h *ViewHandler) WithGroup(name string) slog.Handler {
	return &ViewHandler{
		originalHandler: h.originalHandler.WithGroup(name),
		viewHandler:     h.viewHandler.WithGroup(name),
		gui:             h.gui,
		viewName:        h.viewName,
	}
}

func (h *ViewHandler) Write(p []byte) (n int, err error) {
	gui := (*h.gui)
	view, err := gui.View(h.viewName)
	if err != nil {
		return 0, fmt.Errorf("ViewHandler.Write: failed to get Grbl view: %w", err)
	}
	return view.Write(p)
}
