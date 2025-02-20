package theo

import (
	"context"
	"log/slog"
	"time"

	"github.com/grid-stream-org/go-commons/pkg/bqclient"
	"github.com/grid-stream-org/theo/internal/config"
	"github.com/grid-stream-org/theo/internal/event"
	"github.com/grid-stream-org/theo/internal/handler"
	"github.com/grid-stream-org/theo/internal/scheduler"
	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type Theo struct {
	cfg       *config.Config
	store     event.Store
	scheduler scheduler.Scheduler
	log       *slog.Logger
}

func New(ctx context.Context, cfg *config.Config, log *slog.Logger) (*Theo, error) {
	bq, err := bqclient.New(ctx, cfg.Database)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	kcfg, err := rest.InClusterConfig()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	k8s, err := kubernetes.NewForConfig(kcfg)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	handler := handler.NewK8sHandler(cfg.K8s, k8s, log)

	return &Theo{
		cfg:       cfg,
		store:     event.NewBigQueryStore(bq, cfg.Database.DatasetID),
		scheduler: scheduler.NewScheduler(handler, log),
		log:       log.With("component", "theo"),
	}, nil
}

func (theo *Theo) Run(ctx context.Context) error {
	theo.log.Info("stating theo")
	if err := theo.refresh(ctx); err != nil {
		return errors.WithStack(err)
	}

	ticker := time.NewTicker(theo.cfg.Theo.PollInterval)
	defer ticker.Stop()

	theo.log.Info("theo started")

	for {
		select {
		case <-ctx.Done():
			return theo.Stop()
		case <-ticker.C:
			if err := theo.refresh(ctx); err != nil {
				theo.log.Error("failed to refresh events", "error", err)
			}
		}
	}
}

func (theo *Theo) refresh(ctx context.Context) error {
	events, err := theo.store.GetUpcoming(ctx)
	if err != nil {
		return errors.WithStack(err)
	}

	if err := theo.scheduler.Schedule(ctx, events); err != nil {
		return errors.WithStack(err)
	}

	return nil
}

func (theo *Theo) Stop() error {
	theo.log.Info("stopping theo")

	theo.scheduler.Stop()

	if err := theo.store.Close(); err != nil {
		return errors.WithStack(err)
	}

	theo.log.Info("theo stopped")
	return nil
}
