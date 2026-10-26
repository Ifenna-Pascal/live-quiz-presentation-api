package persistence

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/ifenna-pascal/interactive-presentation/internal/constant/model"
)

// GetCurrentPoll returns the poll on screen, including the correct flags.
// Callers hide them from players until the poll is revealed.
func (store *SQLStore) GetCurrentPoll(ctx context.Context, id uuid.UUID) (model.CurrentPoll, error) {
	poll, err := store.GetPoll(ctx, id)
	if err != nil {
		return model.CurrentPoll{}, err
	}
	return toCurrentPoll(poll)
}

func toCurrentPoll(poll GetPollRow) (model.CurrentPoll, error) {
	result := model.CurrentPoll{
		ID:         poll.ID.String(),
		Index:      poll.Pollindex,
		ServerTime: poll.ServerTime,
	}
	result.Question = poll.Question
	result.TimeLimitSeconds = poll.Timelimitseconds
	if err := json.Unmarshal(poll.Options, &result.Options); err != nil {
		return model.CurrentPoll{}, err
	}
	for _, o := range result.Options {
		result.Scored = result.Scored || o.Correct
	}

	if poll.Openedat.Valid {
		opened := poll.Openedat.Time
		closes := opened.Add(time.Duration(poll.Timelimitseconds) * time.Second)
		result.OpenedAt = &opened
		result.ClosesAt = &closes
	}
	if poll.Winnerplayerid.Valid && poll.Wonat.Valid {
		winner := &model.PollWinner{
			PlayerID:   poll.Winnerplayerid.UUID.String(),
			PlayerName: poll.WinnerName.String,
			Key:        poll.WinnerKey,
			WonAt:      poll.Wonat.Time,
		}
		if poll.Openedat.Valid {
			winner.ElapsedMs = poll.Wonat.Time.Sub(poll.Openedat.Time).Milliseconds()
		}
		result.Winner = winner
	}

	switch {
	case poll.Wonat.Valid:
		result.ClosedReason = model.ClosedByWinner
	case poll.Isrevealed:
		result.ClosedReason = model.ClosedByHost
	case result.ClosesAt != nil && !poll.ServerTime.Before(*result.ClosesAt):
		result.ClosedReason = model.ClosedByTimeUp
	case poll.Hasfinished:
		result.ClosedReason = model.ClosedByFinished
	}
	result.Revealed = result.ClosedReason != ""
	return result, nil
}

// StartPresentationTx closes registration, shows the first poll and starts its countdown.
func (store *SQLStore) StartPresentationTx(ctx context.Context, presentationID uuid.UUID) error {
	return store.execTx(ctx, func(q *Queries) error {
		if _, err := q.StartPresentation(ctx, presentationID); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return model.ErrPresentationAlreadyStarted
			}
			return err
		}
		return q.OpenCurrentPoll(ctx, presentationID)
	})
}

func (store *SQLStore) UpdateCurrentPollToForwardTx(ctx context.Context, id uuid.UUID) (model.CurrentPoll, error) {
	return store.movePoll(ctx, id, func(q *Queries) (int32, error) {
		index, err := q.MoveToNextPoll(ctx, id)
		return index.Int32, err
	})
}

func (store *SQLStore) UpdateCurrentPollToPreviousTx(ctx context.Context, id uuid.UUID) (model.CurrentPoll, error) {
	return store.movePoll(ctx, id, func(q *Queries) (int32, error) {
		index, err := q.MoveToPreviousPoll(ctx, id)
		return index.Int32, err
	})
}

func (store *SQLStore) movePoll(ctx context.Context, id uuid.UUID, move func(*Queries) (int32, error)) (model.CurrentPoll, error) {
	var result model.CurrentPoll
	err := store.execTx(ctx, func(q *Queries) error {
		if _, err := q.GetPresentation(ctx, id); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return model.ErrNotFound
			}
			return err
		}
		if _, err := move(q); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return model.ErrRunOutOfIndex
			}
			return err
		}
		// A poll's countdown starts the first time it is shown; going back to
		// a poll does not restart it.
		if err := q.OpenCurrentPoll(ctx, id); err != nil {
			return err
		}
		poll, err := q.GetPoll(ctx, id)
		if err != nil {
			return err
		}
		result, err = toCurrentPoll(poll)
		return err
	})
	return result, err
}

func (store *SQLStore) GetOwnedPresentation(ctx context.Context, userID, presentationID uuid.UUID) (Presentation, error) {
	pres, err := store.GetPresentation(ctx, presentationID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Presentation{}, model.ErrNotFound
		}
		return Presentation{}, err
	}
	if !pres.Userid.Valid || pres.Userid.UUID != userID {
		return Presentation{}, model.ErrForbidden
	}
	return pres, nil
}

// CreatePresentationTx creates a presentation and all its polls at once.
func (store *SQLStore) CreatePresentationTx(ctx context.Context, userID uuid.UUID, details PresentationDetails, polls []byte) (uuid.UUID, error) {
	var id uuid.UUID
	err := store.execTx(ctx, func(q *Queries) error {
		var err error
		id, err = q.CreatePresentation(ctx, CreatePresentationParams{
			UserID:        uuid.NullUUID{UUID: userID, Valid: true},
			Title:         details.Title,
			PrizePool:     details.PrizePool,
			PrizeCurrency: details.PrizeCurrency,
		})
		if err != nil {
			return err
		}
		return q.InsertPolls(ctx, InsertPollsParams{Polls: polls, PresentationID: id})
	})
	return id, err
}

// ReplacePresentationTx sets new details (title, prize) and replaces every
// poll. Only call it before the presentation starts: there are no votes to
// lose yet.
func (store *SQLStore) ReplacePresentationTx(ctx context.Context, presentationID uuid.UUID, details PresentationDetails, polls []byte) error {
	return store.execTx(ctx, func(q *Queries) error {
		if err := q.UpdatePresentationDetails(ctx, UpdatePresentationDetailsParams{
			Title:          details.Title,
			PrizePool:      details.PrizePool,
			PrizeCurrency:  details.PrizeCurrency,
			PresentationID: presentationID,
		}); err != nil {
			return err
		}
		if err := q.DeletePolls(ctx, presentationID); err != nil {
			return err
		}
		return q.InsertPolls(ctx, InsertPollsParams{Polls: polls, PresentationID: presentationID})
	})
}
