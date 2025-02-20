package event

import (
	"context"
	"time"
)

type Event struct {
	ID        string    `json:"id" bigquery:"id"`
	StartTime time.Time `json:"start_time" bigquery:"start_time"`
	EndTime   time.Time `json:"end_time" bigquery:"end_time"`
	UtilityID string    `json:"utility_id" bigquery:"utility_id"`
}

type Handler interface {
	OnStart(ctx context.Context, event Event) error
	OnEnd(ctx context.Context, event Event) error
}

func (e *Event) LogFields() []any {
	fields := []any{
		"component", "event",
		"id", e.ID,
		"start_time", e.StartTime,
		"end_time", e.EndTime,
		"utility_id", e.UtilityID,
	}
	return fields
}
