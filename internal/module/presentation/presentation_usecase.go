package presentation

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	"github.com/ifenna-pascal/interactive-presentation/internal/constant/model"
	db "github.com/ifenna-pascal/interactive-presentation/internal/storage/persistence"
)

type Usecase interface {
	CreatePresentation(ctx context.Context, userID uuid.UUID, presentation *model.CreatePresentionRequestDTO) (uuid.UUID, error)
	GetPresentation(ctx context.Context, presentationID uuid.UUID) (model.PresentionResponseDTO, error)
	GetCurrentPoll(ctx context.Context, presentationID uuid.UUID) (model.CurrentPoll, error)
	UpdateCurrentPoll(ctx context.Context, userID, presentationID uuid.UUID) (model.CurrentPoll, error)
	ListMyPresentations(ctx context.Context, userID uuid.UUID, limit, offset int32) ([]model.MyPresentationResponse, error)
}

type service struct {
	store db.Store
}

func Initialize(
	store db.Store,
) Usecase {
	return &service{store: store}
}

func (s service) CreatePresentation(ctx context.Context, userID uuid.UUID, presentation *model.CreatePresentionRequestDTO) (uuid.UUID, error) {
	jsonb, err := json.Marshal(presentation.Polls)
	if err != nil {
		return uuid.Nil, err
	}

	presID, err := s.store.CreatePresentationAndPolls(ctx, db.CreatePresentationAndPollsParams{
		UserID: uuid.NullUUID{UUID: userID, Valid: true},
		Polls:  jsonb,
	})
	if err != nil {
		return uuid.Nil, err
	}
	return presID, nil
}

func (s service) GetPresentation(ctx context.Context, presentationID uuid.UUID) (model.PresentionResponseDTO, error) {
	pres, err := s.store.GetPresentation(ctx, presentationID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.PresentionResponseDTO{}, model.ErrNotFound
		}
		return model.PresentionResponseDTO{}, err
	}

	rows, err := s.store.ListPolls(ctx, presentationID)
	if err != nil {
		return model.PresentionResponseDTO{}, err
	}

	polls := make([]model.PresentationResponsePoll, 0, len(rows))
	for _, row := range rows {
		polls = append(polls, model.PresentationResponsePoll{
			Question: row.Question,
			Options:  row.Options,
		})
	}

	return model.PresentionResponseDTO{
		ID:               pres.ID.String(),
		CurrentPollIndex: pres.Currentpollindex.Int32,
		HasStarted:       pres.Hasstarted,
		Polls:            polls,
	}, nil
}

func (s service) GetCurrentPoll(ctx context.Context, presentationID uuid.UUID) (model.CurrentPoll, error) {
	if _, err := s.store.GetPresentation(ctx, presentationID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.CurrentPoll{}, model.ErrNotFound
		}
		return model.CurrentPoll{}, err
	}

	poll, err := s.store.GetCurrentPoll(ctx, presentationID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.CurrentPoll{}, model.ErrNoPollDisplayed
		}
		return model.CurrentPoll{}, err
	}
	return poll, nil
}

func (s service) UpdateCurrentPoll(ctx context.Context, userID, presentationID uuid.UUID) (model.CurrentPoll, error) {
	// Only the owner may move the presentation forward.
	if _, err := s.store.GetOwnedPresentation(ctx, userID, presentationID); err != nil {
		return model.CurrentPoll{}, err
	}
	return s.store.UpdateCurrentPollToForwardTx(ctx, presentationID)
}

func (s service) ListMyPresentations(ctx context.Context, userID uuid.UUID, limit, offset int32) ([]model.MyPresentationResponse, error) {
	rows, err := s.store.ListPresentationsByUser(ctx, db.ListPresentationsByUserParams{
		UserID:     uuid.NullUUID{UUID: userID, Valid: true},
		PageLimit:  limit,
		PageOffset: offset,
	})
	if err != nil {
		return []model.MyPresentationResponse{}, err
	}

	presentations := make([]model.MyPresentationResponse, 0, len(rows))
	for _, row := range rows {
		presentations = append(presentations, model.MyPresentationResponse{
			ID:               row.ID.String(),
			CurrentPollIndex: row.Currentpollindex.Int32,
			HasStarted:       row.Hasstarted,
			PollCount:        row.PollCount,
			CreatedAt:        row.Createdat,
		})
	}
	return presentations, nil
}
