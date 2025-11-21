package control

import (
	"context"
	"log/slog"
)

// EnabledOverrideHandler is a slog.Handler wrapper that delegates Enabled checks
// to a separate handler while using the embedded Handler for all other operations.
type EnabledOverrideHandler struct {
	slog.Handler
	// EnabledHandler is the handler used for Enabled level checks.
	EnabledHandler slog.Handler
}

func (h *EnabledOverrideHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.EnabledHandler.Enabled(ctx, level)
}

func (h *EnabledOverrideHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &EnabledOverrideHandler{
		Handler:        h.Handler.WithAttrs(attrs),
		EnabledHandler: h.EnabledHandler.WithAttrs(attrs),
	}
}

func (h *EnabledOverrideHandler) WithGroup(name string) slog.Handler {
	return &EnabledOverrideHandler{
		Handler:        h.Handler.WithGroup(name),
		EnabledHandler: h.EnabledHandler.WithGroup(name),
	}
}
