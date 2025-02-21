package handler

import (
	"context"
	"log/slog"

	"github.com/grid-stream-org/theo/internal/event"
)

// this guy is just for debugging
type textHandler struct {
	log *slog.Logger
}

func NewTextHandler(log *slog.Logger) event.Handler {
	return &textHandler{
		log: log.With("component", "text_handler"),
	}
}

func (h *textHandler) OnStart(ctx context.Context, e event.Event) error {
	h.log.Info("starting event", e.LogFields()...)
	return nil
}

func (h *textHandler) OnEnd(ctx context.Context, e event.Event) error {
	h.log.Info("ending event", e.LogFields()...)
	return nil
}
