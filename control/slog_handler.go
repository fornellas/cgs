package control

import (
	"context"
	"log/slog"
	"sync/atomic"
)

// EnabledOverrideHandler is a slog.Handler wrapper that delegates Enabled checks
// to a separate handler while using the embedded Handler for all other operations.
// It can be disabled to prevent all log records from being handled.
type EnabledOverrideHandler struct {
	slog.Handler
	enabledHandler slog.Handler
	disabled       *atomic.Bool
}

// NewEnabledOverrideHandler creates a new EnabledOverrideHandler that uses handler
// for record handling and enabledHandler for level checks.
func NewEnabledOverrideHandler(handler, enabledHandler slog.Handler) *EnabledOverrideHandler {
	return &EnabledOverrideHandler{
		Handler:        handler,
		enabledHandler: enabledHandler,
		disabled:       &atomic.Bool{},
	}
}

// Disable prevents the handler from processing any log records.
func (h *EnabledOverrideHandler) Disable() {
	h.disabled.Store(true)
}

// Handle processes a log record if the handler is not disabled.
func (h *EnabledOverrideHandler) Handle(ctx context.Context, r slog.Record) error {
	if h.disabled.Load() {
		return nil
	}
	return h.Handler.Handle(ctx, r)
}

// Enabled checks if logging is enabled for the given level using the enabledHandler,
// unless the handler has been disabled.
func (h *EnabledOverrideHandler) Enabled(ctx context.Context, level slog.Level) bool {
	if h.disabled.Load() {
		return false
	}
	return h.enabledHandler.Enabled(ctx, level)
}

// WithAttrs returns a new EnabledOverrideHandler with the given attributes added.
func (h *EnabledOverrideHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &EnabledOverrideHandler{
		Handler:        h.Handler.WithAttrs(attrs),
		enabledHandler: h.enabledHandler.WithAttrs(attrs),
		disabled:       h.disabled,
	}
}

// WithGroup returns a new EnabledOverrideHandler with the given group added.
func (h *EnabledOverrideHandler) WithGroup(name string) slog.Handler {
	return &EnabledOverrideHandler{
		Handler:        h.Handler.WithGroup(name),
		enabledHandler: h.enabledHandler.WithGroup(name),
		disabled:       h.disabled,
	}
}
