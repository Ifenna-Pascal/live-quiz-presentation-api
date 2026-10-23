package persistence

import (
	"context"
	"database/sql"
	"errors"

	"github.com/ifenna-pascal/interactive-presentation/internal/constant/model"
)

func (store *SQLStore) VoteCurrentPollTx(ctx context.Context, arg VoteParams) error {
	return store.execTx(ctx, func(q *Queries) error {
		row, err := q.GetPresentationAndPoll(ctx, GetPresentationAndPollParams{
			PresentationID: arg.PresentationID,
			PollID:         arg.Pollid,
		})
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return model.ErrNotFound
			}
			return err
		}
		if !row.Currentpollindex.Valid || row.Currentpollindex.Int32 == 0 {
			return model.ErrNoPollDisplayed
		}
		if row.Currentpollindex.Int32 != row.Pollindex {
			return model.ErrIDMismatch
		}
		// CreateVote returns no row when the player is not registered in this
		// presentation or the presentation has not started.
		_, err = q.CreateVote(ctx, CreateVoteParams{
			PollID:    arg.Pollid,
			OptionKey: arg.Optionkey,
			PlayerID:  arg.PlayerID,
		})
		if errors.Is(err, sql.ErrNoRows) {
			return model.ErrPlayerNotRegistered
		}
		return err
	})
}
