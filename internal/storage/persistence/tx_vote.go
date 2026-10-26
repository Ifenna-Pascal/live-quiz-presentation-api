package persistence

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/ifenna-pascal/interactive-presentation/internal/constant/model"
)

// VoteCurrentPollTx records an answer on the poll on screen.
//
// The poll row stays locked until the transaction ends, so answers to one
// poll are handled one at a time, in the order the database receives them.
// The first correct answer wins the poll and closes it; anyone answering
// after that, even a millisecond later, gets ErrPollAlreadyWon and their
// answer is not recorded.
//
// On scored polls a wrong answer is recorded and the player may try another
// option until the time runs out. On opinion polls (no correct option) each
// player answers once.
func (store *SQLStore) VoteCurrentPollTx(ctx context.Context, arg VoteParams) (model.VoteResult, error) {
	var result model.VoteResult
	err := store.execTx(ctx, func(q *Queries) error {
		poll, err := q.LockPollForVote(ctx, LockPollForVoteParams{
			PresentationID: arg.PresentationID,
			PollID:         arg.Pollid,
		})
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return model.ErrNotFound
			}
			return err
		}
		switch {
		case !poll.Currentpollindex.Valid || poll.Currentpollindex.Int32 == 0:
			return model.ErrNoPollDisplayed
		case poll.Currentpollindex.Int32 != poll.Pollindex:
			return model.ErrIDMismatch
		case poll.Winnerplayerid.Valid:
			return model.ErrPollAlreadyWon
		case poll.Isrevealed || poll.Hasfinished:
			return model.ErrVotingClosed
		case poll.Openedat.Valid && !poll.Now.Before(poll.Openedat.Time.Add(time.Duration(poll.Timelimitseconds)*time.Second)):
			return model.ErrTimeUp
		}

		correct, err := q.IsOptionCorrect(ctx, IsOptionCorrectParams{PollID: arg.Pollid, OptionKey: arg.Optionkey})
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return model.ErrInvalidOption
			}
			return err
		}
		scored, err := q.PollHasCorrectOption(ctx, arg.Pollid)
		if err != nil {
			return err
		}
		if !scored {
			voted, err := q.PlayerHasVoted(ctx, PlayerHasVotedParams{PollID: arg.Pollid, PlayerID: arg.PlayerID})
			if err != nil {
				return err
			}
			if voted {
				return model.ErrAlreadyVoted
			}
		}

		if _, err = q.CreateVote(ctx, CreateVoteParams{
			PollID:           arg.Pollid,
			OptionKey:        arg.Optionkey,
			PlayerID:         arg.PlayerID,
			PlayerSecretHash: arg.PlayerSecretHash,
		}); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return model.ErrPlayerNotRegistered
			}
			return err
		}

		result = model.VoteResult{Scored: scored, Correct: scored && correct}
		if !result.Correct {
			return nil
		}
		won, err := q.SetPollWinner(ctx, SetPollWinnerParams{PlayerID: uuid.NullUUID{UUID: arg.PlayerID, Valid: true}, PollID: arg.Pollid})
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return model.ErrPollAlreadyWon
			}
			return err
		}
		result.Won = true
		if won.Wonat.Valid && won.Openedat.Valid {
			result.ElapsedMs = won.Wonat.Time.Sub(won.Openedat.Time).Milliseconds()
		}
		return nil
	})
	if err != nil {
		return model.VoteResult{}, err
	}
	return result, nil
}
