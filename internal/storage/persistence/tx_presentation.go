package persistence

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	"github.com/ifenna-pascal/interactive-presentation/internal/constant/model"
)

func (store *SQLStore) GetCurrentPoll(ctx context.Context, id uuid.UUID) (model.CurrentPoll, error) {
	var result model.CurrentPoll
	err := store.execTx(ctx, func(q *Queries) error {
		poll, err := q.GetPoll(ctx, id)
		if err != nil {
			return err
		}
		result.ID = poll.ID.String()
		result.Question = poll.Question
		var optionRows []model.Option
		err = json.Unmarshal(poll.Options, &optionRows)
		if err != nil {
			return err
		}
		result.Options = optionRows
		return err
	})
	return result, err
}

func (store *SQLStore) UpdateCurrentPollToForwardTx(ctx context.Context, id uuid.UUID) (model.CurrentPoll, error) {
	var result model.CurrentPoll
	err := store.execTx(ctx, func(q *Queries) error {
		if _, err := q.GetPresentation(ctx, id); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return model.ErrNotFound
			}
			return err
		}
		poll, err := q.GetNextPoll(ctx, id)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return model.ErrRunOutOfIndex
			}
			return err
		}
		result.ID = poll.ID.String()
		result.Question = poll.Question
		return json.Unmarshal(poll.Options, &result.Options)
	})
	return result, err
}

func (store *SQLStore) UpdateCurrentPollToPreviousTx(ctx context.Context, id uuid.UUID) (model.CurrentPoll, error) {
	var result model.CurrentPoll
	err := store.execTx(ctx, func(q *Queries) error {
		if _, err := q.GetPresentation(ctx, id); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return model.ErrNotFound
			}
			return err
		}
		poll, err := q.GetPreviousPoll(ctx, id)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return model.ErrRunOutOfIndex
			}
			return err
		}
		result.ID = poll.ID.String()
		result.Question = poll.Question
		return json.Unmarshal(poll.Options, &result.Options)
	})
	return result, err
}

// GetOwnedPresentation returns the presentation if userID owns it.
// It returns model.ErrNotFound if it does not exist and model.ErrForbidden if
// someone else owns it. Presentations created before owners existed have no
// owner, so nobody can control them.
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
