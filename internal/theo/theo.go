package theo

import (
	"context"
	"log/slog"

	"github.com/grid-stream-org/theo/internal/config"
)

type Theo struct{}

func New(ctx context.Context, cfg *config.Config, log *slog.Logger) (*Theo, error) {
	return &Theo{}, nil
}

func (t *Theo) Run(ctx context.Context) error {
	return nil
}
