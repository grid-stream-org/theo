package event

import (
	"context"
	"fmt"

	"github.com/grid-stream-org/go-commons/pkg/bqclient"
	"github.com/pkg/errors"
)

type Store interface {
	GetUpcoming(context.Context) ([]Event, error)
	Close() error
}

type bigQueryStore struct {
	client    bqclient.BQClient
	datasetID string
}

func NewBigQueryStore(client bqclient.BQClient, datasetID string) Store {
	return &bigQueryStore{
		client:    client,
		datasetID: datasetID,
	}
}

func (s *bigQueryStore) GetUpcoming(ctx context.Context) ([]Event, error) {
	query := fmt.Sprintf(`
		SELECT id, start_time, end_time, utility_id
		FROM %s.dr_events
		WHERE end_time >= CURRENT_TIMESTAMP()
		ORDER BY start_time ASC`,
		s.datasetID,
	)

	iter, err := s.client.Query(ctx, query, nil)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	var events []Event
	for {
		var event Event
		err := iter.Next(&event)
		if err != nil {
			break
		}
		events = append(events, event)
	}

	return events, nil
}

func (s *bigQueryStore) Close() error {
	return s.client.Close()
}
