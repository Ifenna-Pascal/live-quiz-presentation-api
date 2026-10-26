package realtime

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/ifenna-pascal/interactive-presentation/internal/constant/model"
	db "github.com/ifenna-pascal/interactive-presentation/internal/storage/persistence"
	"github.com/ifenna-pascal/interactive-presentation/platform/events"
)

type Usecase interface {
	// Subscribe streams the events of one presentation until cancel is called.
	Subscribe(ctx context.Context, presentationID uuid.UUID) (stream <-chan events.Event, cancel func(), err error)
}

type service struct {
	store db.Store
	hub   *events.Hub
}

func Initialize(store db.Store, hub *events.Hub) Usecase {
	return &service{store: store, hub: hub}
}

func (s service) Subscribe(ctx context.Context, presentationID uuid.UUID) (<-chan events.Event, func(), error) {
	if _, err := s.store.GetPresentation(ctx, presentationID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil, model.ErrNotFound
		}
		return nil, nil, err
	}
	stream, cancel := s.hub.Subscribe(presentationID.String())
	return stream, cancel, nil
}
