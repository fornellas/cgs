package control

import (
	"context"
	"log/slog"
)

// EnabledOverrideHandler is a slog.Handler wrapper that delegates Enabled checks
// to a separate handler while using the embedded Handler for all other operations.
type EnabledOverrideHandler struct {
	slog.Handler
	enabledHandler slog.Handler
}

// NewEnabledOverrideHandler creates a new EnabledOverrideHandler that uses handler
// for record handling and enabledHandler for level checks.
func NewEnabledOverrideHandler(handler, enabledHandler slog.Handler) *EnabledOverrideHandler {
	return &EnabledOverrideHandler{
		Handler:        handler,
		enabledHandler: enabledHandler,
	}
}

func (h *EnabledOverrideHandler) Handle(ctx context.Context, r slog.Record) error {
	return h.Handler.Handle(ctx, r)
}

func (h *EnabledOverrideHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.enabledHandler.Enabled(ctx, level)
}

func (h *EnabledOverrideHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &EnabledOverrideHandler{
		Handler:        h.Handler.WithAttrs(attrs),
		enabledHandler: h.enabledHandler.WithAttrs(attrs),
	}
}

func (h *EnabledOverrideHandler) WithGroup(name string) slog.Handler {
	return &EnabledOverrideHandler{
		Handler:        h.Handler.WithGroup(name),
		enabledHandler: h.enabledHandler.WithGroup(name),
	}
}
