package scheduler

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/grid-stream-org/theo/internal/event"
	"github.com/pkg/errors"
)

type Scheduler interface {
	Schedule(ctx context.Context, events []event.Event) error
	Stop()
}

type scheduledEvent struct {
	event      event.Event
	startTimer *time.Timer
	endTimer   *time.Timer
}

type eventScheduler struct {
	handler event.Handler
	events  map[string]scheduledEvent
	mu      sync.RWMutex
	log     *slog.Logger
}

func NewScheduler(handler event.Handler, log *slog.Logger) Scheduler {
	return &eventScheduler{
		handler: handler,
		events:  make(map[string]scheduledEvent),
		log:     log.With("component", "scheduler"),
	}
}

func (s *eventScheduler) Schedule(ctx context.Context, events []event.Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Stop existing timers
	for _, e := range s.events {
		if e.startTimer != nil {
			e.startTimer.Stop()
		}
		if e.endTimer != nil {
			e.endTimer.Stop()
		}
	}

	// Clear and reschedule
	s.events = make(map[string]scheduledEvent)
	for _, e := range events {
		if err := s.scheduleEvent(ctx, e); err != nil {
			return errors.WithStack(err)
		}
	}

	return nil
}

func (s *eventScheduler) scheduleEvent(ctx context.Context, e event.Event) error {
	now := time.Now()

	// Only schedule if the event hasn't ended
	if e.EndTime.Before(now) {
		return nil
	}

	scheduled := scheduledEvent{
		event: e,
	}

	// Schedule start (if not already started)
	if e.StartTime.Add(-5 * time.Minute).After(now) {
		startDelay := time.Until(e.StartTime.Add(-5 * time.Minute))
		scheduled.startTimer = time.AfterFunc(startDelay, func() {
			if err := s.handler.OnStart(ctx, e); err != nil {
				fields := append([]any{"error", err}, e.LogFields()...)
				s.log.Error("failed to handle event start", fields...)
			}
		})
	}

	// Schedule end (if not already ended)
	if e.EndTime.After(now) {
		endDelay := time.Until(e.EndTime)
		scheduled.endTimer = time.AfterFunc(endDelay, func() {
			if err := s.handler.OnEnd(ctx, e); err != nil {
				fields := append([]any{"error", err}, e.LogFields()...)
				s.log.Error("failed to handle event end", fields...)
			}
		})
	}

	s.events[e.ID] = scheduled
	return nil
}

func (s *eventScheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for id, e := range s.events {
		if e.startTimer != nil {
			e.startTimer.Stop()
		}
		if e.endTimer != nil {
			e.endTimer.Stop()
		}
		delete(s.events, id)
	}

}
