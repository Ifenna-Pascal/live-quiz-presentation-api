package presentation

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	"github.com/ifenna-pascal/interactive-presentation/internal/constant/model"
	db "github.com/ifenna-pascal/interactive-presentation/internal/storage/persistence"
	"github.com/ifenna-pascal/interactive-presentation/platform/events"
)

// Usecase is everything a presentation's owner and audience can do with it.
type Usecase interface {
	CreatePresentation(ctx context.Context, userID uuid.UUID, presentation *model.CreatePresentionRequestDTO) (uuid.UUID, error)
	UpdatePresentation(ctx context.Context, userID, presentationID uuid.UUID, presentation *model.CreatePresentionRequestDTO) error
	DeletePresentation(ctx context.Context, userID, presentationID uuid.UUID) error
	// GetPresentation includes correct answers when viewerID is the owner.
	GetPresentation(ctx context.Context, viewerID *uuid.UUID, presentationID uuid.UUID) (model.PresentionResponseDTO, error)
	GetCurrentPoll(ctx context.Context, presentationID uuid.UUID) (model.CurrentPoll, error)
	UpdateCurrentPoll(ctx context.Context, userID, presentationID uuid.UUID) (model.CurrentPoll, error)
	PreviousPoll(ctx context.Context, userID, presentationID uuid.UUID) (model.CurrentPoll, error)
	RevealCurrentPoll(ctx context.Context, userID, presentationID uuid.UUID) (model.CurrentPoll, error)
	FinishPresentation(ctx context.Context, userID, presentationID uuid.UUID) error
	ListMyPresentations(ctx context.Context, userID uuid.UUID, limit, offset int32) ([]model.MyPresentationResponse, error)
}

type service struct {
	store     db.Store
	publisher events.Publisher
}

func Initialize(store db.Store, publisher events.Publisher) Usecase {
	return &service{store: store, publisher: publisher}
}

func (s service) CreatePresentation(ctx context.Context, userID uuid.UUID, req *model.CreatePresentionRequestDTO) (uuid.UUID, error) {
	if err := normalize(req); err != nil {
		return uuid.Nil, err
	}
	polls, err := json.Marshal(req.Polls)
	if err != nil {
		return uuid.Nil, err
	}
	return s.store.CreatePresentationTx(ctx, userID, details(req), polls)
}

func (s service) UpdatePresentation(ctx context.Context, userID, presentationID uuid.UUID, req *model.CreatePresentionRequestDTO) error {
	pres, err := s.store.GetOwnedPresentation(ctx, userID, presentationID)
	if err != nil {
		return err
	}
	if pres.Hasstarted {
		return model.ErrPresentationAlreadyStarted
	}
	if err := normalize(req); err != nil {
		return err
	}
	polls, err := json.Marshal(req.Polls)
	if err != nil {
		return err
	}
	if err := s.store.ReplacePresentationTx(ctx, presentationID, details(req), polls); err != nil {
		return err
	}
	s.publisher.Publish(ctx, presentationID, events.TypePresentation, nil)
	return nil
}

func (s service) DeletePresentation(ctx context.Context, userID, presentationID uuid.UUID) error {
	if _, err := s.store.GetOwnedPresentation(ctx, userID, presentationID); err != nil {
		return err
	}
	rows, err := s.store.DeletePresentation(ctx, presentationID)
	if err != nil {
		return err
	}
	if rows == 0 {
		return model.ErrNotFound
	}
	s.publisher.Publish(ctx, presentationID, events.TypeDeleted, nil)
	return nil
}

func (s service) GetPresentation(ctx context.Context, viewerID *uuid.UUID, presentationID uuid.UUID) (model.PresentionResponseDTO, error) {
	pres, err := s.store.GetPresentation(ctx, presentationID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.PresentionResponseDTO{}, model.ErrNotFound
		}
		return model.PresentionResponseDTO{}, err
	}
	isOwner := viewerID != nil && pres.Userid.Valid && pres.Userid.UUID == *viewerID

	rows, err := s.store.ListPolls(ctx, presentationID)
	if err != nil {
		return model.PresentionResponseDTO{}, err
	}

	polls := make([]model.PresentationResponsePoll, 0, len(rows))
	for _, row := range rows {
		var options []model.Option
		if err := json.Unmarshal(row.Options, &options); err != nil {
			return model.PresentionResponseDTO{}, err
		}
		if !isOwner && !row.Isrevealed {
			options = model.HideAnswers(options)
		}
		polls = append(polls, model.PresentationResponsePoll{
			ID:               row.ID.String(),
			Question:         row.Question,
			TimeLimitSeconds: row.Timelimitseconds,
			Revealed:         row.Isrevealed,
			Options:          options,
		})
	}

	return model.PresentionResponseDTO{
		ID:               pres.ID.String(),
		Title:            pres.Title,
		CurrentPollIndex: pres.Currentpollindex.Int32,
		HasStarted:       pres.Hasstarted,
		HasFinished:      pres.Hasfinished,
		IsOwner:          isOwner,
		Prize:            model.NewPrize(pres.Prizepool.Int64, pres.Prizepool.Valid, pres.Prizecurrency),
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
	if !poll.Revealed {
		poll.Options = model.HideAnswers(poll.Options)
	}
	return poll, nil
}

func (s service) UpdateCurrentPoll(ctx context.Context, userID, presentationID uuid.UUID) (model.CurrentPoll, error) {
	if _, err := s.store.GetOwnedPresentation(ctx, userID, presentationID); err != nil {
		return model.CurrentPoll{}, err
	}
	poll, err := s.store.UpdateCurrentPollToForwardTx(ctx, presentationID)
	if err != nil {
		return model.CurrentPoll{}, err
	}
	s.publisher.Publish(ctx, presentationID, events.TypePresentation, nil)
	return poll, nil
}

func (s service) PreviousPoll(ctx context.Context, userID, presentationID uuid.UUID) (model.CurrentPoll, error) {
	if _, err := s.store.GetOwnedPresentation(ctx, userID, presentationID); err != nil {
		return model.CurrentPoll{}, err
	}
	poll, err := s.store.UpdateCurrentPollToPreviousTx(ctx, presentationID)
	if err != nil {
		return model.CurrentPoll{}, err
	}
	s.publisher.Publish(ctx, presentationID, events.TypePresentation, nil)
	return poll, nil
}

func (s service) RevealCurrentPoll(ctx context.Context, userID, presentationID uuid.UUID) (model.CurrentPoll, error) {
	pres, err := s.store.GetOwnedPresentation(ctx, userID, presentationID)
	if err != nil {
		return model.CurrentPoll{}, err
	}
	if !pres.Hasstarted {
		return model.CurrentPoll{}, model.ErrPresentationNotRunning
	}
	if _, err := s.store.RevealCurrentPoll(ctx, presentationID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.CurrentPoll{}, model.ErrNoPollDisplayed
		}
		return model.CurrentPoll{}, err
	}
	poll, err := s.store.GetCurrentPoll(ctx, presentationID)
	if err != nil {
		return model.CurrentPoll{}, err
	}
	s.publisher.Publish(ctx, presentationID, events.TypePresentation, nil)
	return poll, nil
}

func (s service) FinishPresentation(ctx context.Context, userID, presentationID uuid.UUID) error {
	if _, err := s.store.GetOwnedPresentation(ctx, userID, presentationID); err != nil {
		return err
	}
	if _, err := s.store.FinishPresentation(ctx, presentationID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.ErrPresentationNotRunning
		}
		return err
	}
	s.publisher.Publish(ctx, presentationID, events.TypePresentation, nil)
	return nil
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
			Title:            row.Title,
			CurrentPollIndex: row.Currentpollindex.Int32,
			HasStarted:       row.Hasstarted,
			HasFinished:      row.Hasfinished,
			PollCount:        row.PollCount,
			PlayerCount:      row.PlayerCount,
			Prize:            model.NewPrize(row.Prizepool.Int64, row.Prizepool.Valid, row.Prizecurrency),
			CreatedAt:        row.Createdat,
		})
	}
	return presentations, nil
}
