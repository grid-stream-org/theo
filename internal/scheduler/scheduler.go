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
	Schedule(ctx context.Context, events []event.Event)
	Stop()
}

type scheduledEvent struct {
	event      event.Event
	startTimer *time.Timer
	endTimer   *time.Timer
	active     bool
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

func (s *eventScheduler) Schedule(ctx context.Context, events []event.Event) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.stopAllTimers()

	s.events = make(map[string]scheduledEvent)

	for _, e := range events {
		if err := s.scheduleEvent(ctx, e); err != nil {
			s.log.Error("failed to schedule event", errLogFields(e, err)...)
			continue // continue scheduling other events even if one fails
		}
	}
}

func (s *eventScheduler) scheduleEvent(ctx context.Context, e event.Event) error {
	now := time.Now().UTC()

	scheduled := scheduledEvent{
		event:  e,
		active: false,
	}

	if e.StartTime.IsZero() || e.EndTime.IsZero() {
		return errors.New("event start or end time cannot be zero")
	}
	if e.EndTime.Before(e.StartTime) {
		return errors.New("event end time cannot be before start time")
	}
	if e.EndTime.Before(now) {
		return nil // skip expired events
	}

	bufferStart := e.StartTime.Add(-1 * time.Minute)

	if now.Before(bufferStart) {
		// we have time to schedule the 5-minute warning
		startDelay := time.Until(bufferStart)
		scheduled.startTimer = time.AfterFunc(startDelay, func() {
			s.mu.Lock()
			defer s.mu.Unlock()

			if event, exists := s.events[e.ID]; exists && !event.active {
				if err := s.handler.OnStart(ctx, e); err != nil {
					s.log.Error("failed to handle event start", errLogFields(e, err)...)
				} else {
					event.active = true
					s.log.Info("event started", e.LogFields()...)
				}
			}
		})
	} else if now.Before(e.StartTime) {
		// within 5 minutes of start, execute OnStart immediately
		if err := s.handler.OnStart(ctx, e); err != nil {
			s.log.Error("failed to handle immediate event start", errLogFields(e, err)...)
		} else {
			scheduled.active = true
			s.log.Info("event started immediately", e.LogFields()...)
		}
	}

	if e.EndTime.After(now) {
		endDelay := time.Until(e.EndTime)
		scheduled.endTimer = time.AfterFunc(endDelay, func() {
			s.mu.Lock()
			defer s.mu.Unlock()

			if err := s.handler.OnEnd(ctx, e); err != nil {
				s.log.Error("failed to handle event end", errLogFields(e, err)...)
			} else {
				s.log.Info("event ended", e.LogFields()...)
				delete(s.events, e.ID)
			}
		})
	}

	s.events[e.ID] = scheduled
	return nil
}

func (s *eventScheduler) stopAllTimers() {
	for _, e := range s.events {
		if e.startTimer != nil {
			e.startTimer.Stop()
		}
		if e.endTimer != nil {
			e.endTimer.Stop()
		}
	}
}

func errLogFields(e event.Event, err error) []any {
	return append([]any{"error", err}, e.LogFields()...)
}

func (s *eventScheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.stopAllTimers()
	s.events = make(map[string]scheduledEvent)
	s.log.Info("scheduler stopped")
}
