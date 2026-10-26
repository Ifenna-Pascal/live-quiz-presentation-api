package events

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// Channel is the Postgres NOTIFY channel used for presentation events.
const Channel = "presentation_events"

// Publisher announces that something about a presentation changed.
// Publishing never fails the caller: the change is already saved, and
// clients resync if they miss an event.
type Publisher interface {
	Publish(ctx context.Context, presentationID uuid.UUID, eventType string, data any)
}

type pgPublisher struct {
	db     *sql.DB
	logger *logrus.Logger
}

// NewPostgresPublisher publishes with pg_notify, so every API instance
// listening on Channel receives the event.
func NewPostgresPublisher(db *sql.DB, logger *logrus.Logger) Publisher {
	return &pgPublisher{db: db, logger: logger}
}

func (p *pgPublisher) Publish(ctx context.Context, presentationID uuid.UUID, eventType string, data any) {
	e := Event{PresentationID: presentationID.String(), Type: eventType}
	if data != nil {
		raw, err := json.Marshal(data)
		if err != nil {
			p.logger.Error("events: marshal: ", err)
			return
		}
		e.Data = raw
	}
	payload, err := json.Marshal(e)
	if err != nil {
		p.logger.Error("events: marshal: ", err)
		return
	}
	// Use a fresh context: the request may already be finishing.
	if _, err := p.db.ExecContext(context.WithoutCancel(ctx), "SELECT pg_notify($1, $2)", Channel, string(payload)); err != nil {
		p.logger.Error("events: notify: ", err)
	}
}

// NopPublisher drops every event. Useful in tests.
type NopPublisher struct{}

func (NopPublisher) Publish(context.Context, uuid.UUID, string, any) {}
